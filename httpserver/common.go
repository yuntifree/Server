package httpserver

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"

	"google.golang.org/grpc"

	common "../proto/common"
	discover "../proto/discover"
	fetch "../proto/fetch"
	verify "../proto/verify"
	util "../util"
	simplejson "github.com/bitly/go-simplejson"
)

const (
	discoverAddress = "localhost:50054"
	hotNewsKey      = "hot:news"
	hotVideoKey     = "hot:video"
	hotWeatherKey   = "hot:weather"
	expireInterval  = 300
)

type request struct {
	Post *simplejson.Json
}

func (r *request) init(body io.ReadCloser) (err error) {
	r.Post, err = simplejson.NewFromReader(body)
	return
}

func (r *request) initCheck(body io.ReadCloser, back bool) {
	var err error
	r.Post, err = simplejson.NewFromReader(body)
	if err != nil {
		log.Printf("parse reqbody failed:%v", err)
		panic(util.AppError{util.JSONErr, 4, "invalid param"})
	}

	uid := util.GetJSONInt(r.Post, "uid")
	token := util.GetJSONString(r.Post, "token")

	var ctype int32
	if back {
		ctype = 1
	}

	flag := checkToken(uid, token, ctype)
	if !flag {
		log.Printf("checkToken failed, uid:%d token:%s\n", uid, token)
		panic(util.AppError{util.LogicErr, 101, "token验证失败"})
	}
}

func (r *request) initCheckApp(body io.ReadCloser) {
	r.initCheck(body, false)
}

func (r *request) initCheckOss(body io.ReadCloser) {
	r.initCheck(body, true)
}

func (r *request) GetParamInt(key string) int64 {
	return util.GetJSONInt(r.Post, key)
}

func (r *request) GetParamIntDef(key string, def int64) int64 {
	return util.GetJSONIntDef(r.Post, key, def)
}

func (r *request) GetParamString(key string) string {
	return util.GetJSONString(r.Post, key)
}
func (r *request) GetParamStringDef(key string, def string) string {
	return util.GetJSONStringDef(r.Post, key, def)
}

func (r *request) GetParamFloat(key string) float64 {
	return util.GetJSONFloat(r.Post, key)
}
func (r *request) GetParamFloatDef(key string, def float64) float64 {
	return util.GetJSONFloatDef(r.Post, key, def)
}

func extractError(r interface{}) *util.AppError {
	if v, ok := r.(util.ParamError); ok {
		return &util.AppError{util.ParamErr, 2, v.Error()}
	} else if k, ok := r.(util.AppError); ok {
		return &k
	} else {
		log.Printf("unexpected panic:%v", r)
		return &util.AppError{util.ParamErr, 2, v.Error()}
	}

	return nil
}

type appHandler func(http.ResponseWriter, *http.Request) *util.AppError

func (fn appHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if e := fn(w, r); e != nil {
		log.Printf("error type:%d code:%d msg:%s", e.Type, e.Code, e.Msg)

		js, _ := simplejson.NewJson([]byte(`{}`))
		js.Set("errno", e.Code)
		if e.Code < 100 {
			js.Set("desc", "服务器又傲娇了~")
		} else {
			js.Set("desc", e.Msg)
		}
		body, err := js.MarshalJSON()
		if err != nil {
			log.Printf("MarshalJSON failed: %v", err)
			w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
			return
		}
		w.Write(body)
	}
}

func checkToken(uid int64, token string, ctype int32) bool {
	address := getNameServer(uid, util.VerifyServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Printf("did not connect: %v", err)
		return false
	}
	defer conn.Close()
	c := verify.NewVerifyClient(conn)

	uuid := util.GenUUID()
	res, err := c.CheckToken(context.Background(), &verify.TokenRequest{Head: &common.Head{Sid: uuid, Uid: uid}, Token: token, Type: ctype})
	if err != nil {
		log.Printf("failed: %v", err)
		return false
	}

	if res.Head.Retcode != 0 {
		log.Printf("check token failed")
		return false
	}

	return true
}

func getAps(w http.ResponseWriter, r *http.Request, back bool) (apperr *util.AppError) {
	defer func() {
		if r := recover(); r != nil {
			apperr = extractError(r)
		}
	}()

	var req request
	if back {
		req.initCheckOss(r.Body)
	} else {
		req.initCheckApp(r.Body)
	}
	uid := req.GetParamInt("uid")
	longitude := req.GetParamFloat("longitude")
	latitude := req.GetParamFloat("latitude")

	address := getNameServer(uid, util.FetchServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := fetch.NewFetchClient(conn)
	uuid := util.GenUUID()
	res, err := c.FetchAps(context.Background(), &fetch.ApRequest{Head: &common.Head{Uid: uid, Sid: uuid}, Longitude: longitude, Latitude: latitude})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}

	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "服务器又傲娇了"}
	}

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "init json failed"}
	}
	infos := make([]interface{}, len(res.Infos))
	for i := 0; i < len(res.Infos); i++ {
		json, _ := simplejson.NewJson([]byte(`{}`))
		json.Set("longitude", res.Infos[i].Longitude)
		json.Set("latitude", res.Infos[i].Latitude)
		json.Set("address", res.Infos[i].Address)
		infos[i] = json
	}
	js.SetPath([]string{"data", "infos"}, infos)

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func getDiscoverAddress() string {
	return discoverAddress
}

func getNameServer(uid int64, name string) string {
	address := getDiscoverAddress()
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Printf("did not connect %s: %v", discoverAddress, err)
		panic(util.AppError{util.RPCErr, 4, err.Error()})
	}
	defer conn.Close()
	c := discover.NewDiscoverClient(conn)

	uuid := util.GenUUID()
	res, err := c.Resolve(context.Background(), &discover.ServerRequest{Head: &common.Head{Sid: uuid}, Sname: name})
	if err != nil {
		log.Printf("Resolve failed %s: %v", name, err)
		panic(util.AppError{util.RPCErr, 4, err.Error()})
	}

	if res.Head.Retcode != 0 {
		log.Printf("Resolve failed  name:%s errcode:%d\n", name, res.Head.Retcode)
		panic(util.AppError{util.RPCErr, 4, fmt.Sprintf("Resolve failed  name:%s errcode:%d\n", name, res.Head.Retcode)})
	}

	return res.Host
}
