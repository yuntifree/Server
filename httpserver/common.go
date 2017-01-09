package httpserver

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"reflect"
	"strings"

	"google.golang.org/grpc"

	common "../proto/common"
	discover "../proto/discover"
	fetch "../proto/fetch"
	hot "../proto/hot"
	modify "../proto/modify"
	push "../proto/push"
	verify "../proto/verify"
	util "../util"
	simplejson "github.com/bitly/go-simplejson"
)

const (
	hotNewsKey     = "hot:news"
	hotVideoKey    = "hot:video"
	hotWeatherKey  = "hot:weather"
	hotServiceKey  = "hot:service"
	hotDgNewsKey   = "hot:news:dg"
	hotAmuseKey    = "hot:news:amuse"
	hotJokeKey     = "hot:joke"
	hotNewsCompKey = "hot:news:comp"
	expireInterval = 30
)
const (
	hotNewsType = iota
	hotVideoType
	hotAppType
	hotGameType
	hotDgType
	hotAmuseType
	hotJokeType
)

const (
	errOk = iota
	errMissParam
	errInvalidParam
	errDatabase
	errInner
	errPanic
)
const (
	errToken = iota + 101
	errCode
	errGetCode
	errUsedPhone
	errWxMpLogin
	errUnionID
	errWxTicket
	errNotFound
	errIllegalPhone
	errZteLogin
	errZteRemove
	errNoNewVersion
)

func genParamErr(key string) string {
	return "get param:" + key + " failed"
}

func getJSONString(js *simplejson.Json, key string) string {
	if val, err := js.Get(key).String(); err == nil {
		return val
	}

	if val, err := js.Get("data").Get(key).String(); err == nil {
		return val
	}
	panic(util.AppError{Code: errMissParam, Msg: genParamErr(key)})
}

func getJSONStringDef(js *simplejson.Json, key, def string) string {
	if val, err := js.Get(key).String(); err == nil {
		return val
	}

	if val, err := js.Get("data").Get(key).String(); err == nil {
		return val
	}
	return def
}

func getJSONInt(js *simplejson.Json, key string) int64 {
	if val, err := js.Get(key).Int64(); err == nil {
		return val
	}

	if val, err := js.Get("data").Get(key).Int64(); err == nil {
		return val
	}
	panic(util.AppError{Code: errMissParam, Msg: genParamErr(key)})
}

func getJSONIntDef(js *simplejson.Json, key string, def int64) int64 {
	if val, err := js.Get(key).Int64(); err == nil {
		return val
	}

	if val, err := js.Get("data").Get(key).Int64(); err == nil {
		return val
	}
	return def
}

func getJSONBool(js *simplejson.Json, key string) bool {
	if val, err := js.Get(key).Bool(); err == nil {
		return val
	}

	if val, err := js.Get("data").Get(key).Bool(); err == nil {
		return val
	}
	panic(util.AppError{Code: errMissParam, Msg: genParamErr(key)})
}

func getJSONBoolDef(js *simplejson.Json, key string, def bool) bool {
	if val, err := js.Get(key).Bool(); err == nil {
		return val
	}

	if val, err := js.Get("data").Get(key).Bool(); err == nil {
		return val
	}
	return def
}

func getJSONFloat(js *simplejson.Json, key string) float64 {
	if val, err := js.Get(key).Float64(); err == nil {
		return val
	}

	if val, err := js.Get("data").Get(key).Float64(); err == nil {
		return val
	}
	panic(util.AppError{Code: errMissParam, Msg: genParamErr(key)})
}

func getJSONFloatDef(js *simplejson.Json, key string, def float64) float64 {
	if val, err := js.Get(key).Float64(); err == nil {
		return val
	}

	if val, err := js.Get("data").Get(key).Float64(); err == nil {
		return val
	}
	return def
}

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
		panic(util.AppError{errInvalidParam, "invalid param"})
	}

	uid := getJSONInt(r.Post, "uid")
	token := getJSONString(r.Post, "token")

	var ctype int32
	if back {
		ctype = 1
	}

	flag := checkToken(uid, token, ctype)
	if !flag {
		log.Printf("checkToken failed, uid:%d token:%s\n", uid, token)
		panic(util.AppError{errToken, "token验证失败"})
	}
}

func (r *request) initCheckApp(body io.ReadCloser) {
	r.initCheck(body, false)
}

func (r *request) initCheckOss(body io.ReadCloser) {
	r.initCheck(body, true)
}

func (r *request) GetParamInt(key string) int64 {
	return getJSONInt(r.Post, key)
}

func (r *request) GetParamIntDef(key string, def int64) int64 {
	return getJSONIntDef(r.Post, key, def)
}

func (r *request) GetParamBool(key string) bool {
	return getJSONBool(r.Post, key)
}

func (r *request) GetParamBoolDef(key string, def bool) bool {
	return getJSONBoolDef(r.Post, key, def)
}

func (r *request) GetParamString(key string) string {
	return getJSONString(r.Post, key)
}
func (r *request) GetParamStringDef(key string, def string) string {
	return getJSONStringDef(r.Post, key, def)
}

func (r *request) GetParamFloat(key string) float64 {
	return getJSONFloat(r.Post, key)
}
func (r *request) GetParamFloatDef(key string, def float64) float64 {
	return getJSONFloatDef(r.Post, key, def)
}

func extractError(r interface{}) *util.AppError {
	if k, ok := r.(util.AppError); ok {
		return &k
	}
	log.Printf("unexpected panic:%v", r)
	return &util.AppError{errPanic, r.(error).Error()}
}

func handleError(w http.ResponseWriter, e *util.AppError) {
	log.Printf("error code:%d msg:%s", e.Code, e.Msg)

	js, _ := simplejson.NewJson([]byte(`{}`))
	js.Set("errno", e.Code)
	if e.Code == errInvalidParam || e.Code == errMissParam {
		js.Set("errno", errToken)
		js.Set("desc", "服务器又傲娇了~")
	} else if e.Code < errToken {
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

type appHandler func(http.ResponseWriter, *http.Request) *util.AppError

func (fn appHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			apperr := extractError(r)
			handleError(w, apperr)
		}
	}()
	if e := fn(w, r); e != nil {
		handleError(w, e)
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
	res, err := c.CheckToken(context.Background(),
		&verify.TokenRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Token: token, Type: ctype})
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
		return &util.AppError{errInner, err.Error()}
	}
	defer conn.Close()
	c := fetch.NewFetchClient(conn)
	uuid := util.GenUUID()
	res, err := c.FetchAps(context.Background(),
		&fetch.ApRequest{Head: &common.Head{Uid: uid, Sid: uuid},
			Longitude: longitude, Latitude: latitude})
	if err != nil {
		return &util.AppError{errInner, err.Error()}
	}

	if res.Head.Retcode != 0 {
		return &util.AppError{errInner, "服务器又傲娇了"}
	}

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{errInner, "init json failed"}
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
		return &util.AppError{errInner, "marshal json failed"}
	}
	rspGzip(w, body)
	return nil
}

func getDiscoverAddress() string {
	ip := util.GetInnerIP()
	if ip != util.DebugHost {
		hosts := strings.Split(util.APIHosts, ",")
		if len(hosts) > 0 {
			idx := util.Randn(int32(len(hosts)))
			return hosts[idx] + util.DiscoverServerPort
		}
	}
	return "localhost" + util.DiscoverServerPort
}

func getNameServer(uid int64, name string) string {
	address := getDiscoverAddress()
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Printf("did not connect %s: %v", address, err)
		panic(util.AppError{errInner, err.Error()})
	}
	defer conn.Close()
	c := discover.NewDiscoverClient(conn)

	ip := util.GetInnerIP()
	if ip == util.DebugHost {
		name += ":debug"
	}
	uuid := util.GenUUID()
	res, err := c.Resolve(context.Background(),
		&discover.ServerRequest{Head: &common.Head{Sid: uuid}, Sname: name})
	if err != nil {
		log.Printf("Resolve failed %s: %v", name, err)
		panic(util.AppError{errInner, err.Error()})
	}

	if res.Head.Retcode != 0 {
		log.Printf("Resolve failed  name:%s errcode:%d\n", name, res.Head.Retcode)
		panic(util.AppError{errInner,
			fmt.Sprintf("Resolve failed  name:%s errcode:%d\n", name, res.Head.Retcode)})
	}

	return res.Host
}

func rspGzip(w http.ResponseWriter, body []byte) {
	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Set("Content-Type", "application/json")
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	gw.Write(body)
	gw.Close()
	w.Write(buf.Bytes())
}

func addImages(uid int64, names []string) error {
	address := getNameServer(uid, util.ModifyServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return err
	}
	defer conn.Close()
	c := modify.NewModifyClient(conn)

	uuid := util.GenUUID()
	res, err := c.AddImage(context.Background(),
		&modify.AddImageRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Fnames: names})
	if err != nil {
		return err
	}
	if res.Head.Retcode != 0 {
		return errors.New("添加图片失败")
	}
	return nil
}

func genResponseBody(res interface{}, flag bool) []byte {
	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		panic(util.AppError{errInner, err.Error()})
	}
	val := reflect.ValueOf(res).Elem()
	log.Printf("val:%v", val)
	for i := 0; i < val.NumField(); i++ {
		valueField := val.Field(i)
		typeField := val.Type().Field(i)
		if typeField.Name == "Head" {
			if flag {
				headVal := reflect.Indirect(valueField)
				uid := headVal.FieldByName("Uid")
				js.SetPath([]string{"data", "uid"}, uid.Interface())

			} else {
				continue
			}
		} else {
			js.SetPath([]string{"data", strings.ToLower(typeField.Name)}, valueField.Interface())
		}
	}
	data, err := js.MarshalJSON()
	if err != nil {
		panic(util.AppError{errInner, err.Error()})
	}

	return data
}

func checkRPCRsp(err error, retcode common.ErrCode, method string) {
	if err != nil {
		log.Printf("RPC %s failed:%v", method, err)
		panic(util.AppError{errInner, err.Error()})
	}

	if retcode != 0 {
		log.Printf("%s failed retcode:%d", method, retcode)
		panic(util.AppError{int(retcode), "登录失败"})
	}
}

func checkRPCErr(err reflect.Value, method string) {
	if err.Interface() != nil {
		log.Printf("RPC %s failed:%v", method, err)
		panic(util.AppError{errInner, "grpc failed " + method})
	}
}

func checkRPCCode(retcode common.ErrCode, method string) {
	if retcode != 0 {
		log.Printf("%s failed retcode:%d", method, retcode)
	}
	if retcode == common.ErrCode_INVALID_TOKEN {
		panic(util.AppError{errToken, "token验证失败"})
	} else if retcode == common.ErrCode_USED_PHONE {
		panic(util.AppError{errUsedPhone, "该账号已注册，请直接登录"})
	} else if retcode == common.ErrCode_CHECK_CODE {
		panic(util.AppError{errCode, "验证码错误"})
	} else if retcode == common.ErrCode_ZTE_LOGIN {
		panic(util.AppError{errZteLogin, "登录失败"})
	} else if retcode == common.ErrCode_ZTE_REMOVE {
		panic(util.AppError{errZteRemove, "删除中兴账号失败"})
	} else if retcode == common.ErrCode_NO_NEW_VERSION {
		panic(util.AppError{errNoNewVersion, "当前已是最新版本"})
	} else if retcode != 0 {
		panic(util.AppError{int(retcode), "服务器又傲娇了~"})
	}
}

func genServerName(rtype int64) string {
	switch rtype {
	case util.DiscoverServerType:
		return util.DiscoverServerName
	case util.VerifyServerType:
		return util.VerifyServerName
	case util.HotServerType:
		return util.HotServerName
	case util.FetchServerType:
		return util.FetchServerName
	case util.ModifyServerType:
		return util.ModifyServerName
	case util.PushServerType:
		return util.PushServerName
	default:
		panic(util.AppError{errInvalidParam, "illegal server type"})
	}
}

func genClient(rtype int64, conn *grpc.ClientConn) interface{} {
	var cli interface{}
	switch rtype {
	case util.DiscoverServerType:
		cli = discover.NewDiscoverClient(conn)
	case util.VerifyServerType:
		cli = verify.NewVerifyClient(conn)
	case util.HotServerType:
		cli = hot.NewHotClient(conn)
	case util.FetchServerType:
		cli = fetch.NewFetchClient(conn)
	case util.ModifyServerType:
		cli = modify.NewModifyClient(conn)
	case util.PushServerType:
		cli = push.NewPushClient(conn)
	default:
		panic(util.AppError{errInvalidParam, "illegal server type"})
	}
	return cli
}

func callRPC(rtype, uid int64, method string, request interface{}) (reflect.Value, reflect.Value) {
	var resp reflect.Value
	serverName := genServerName(rtype)
	address := getNameServer(uid, serverName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return resp, reflect.ValueOf(err)
	}
	defer conn.Close()
	cli := genClient(rtype, conn)
	ctx := context.Background()

	inputs := make([]reflect.Value, 2)
	inputs[0] = reflect.ValueOf(ctx)
	inputs[1] = reflect.ValueOf(request)
	arr := reflect.ValueOf(cli).MethodByName(method).Call(inputs)
	if len(arr) != 2 {
		log.Printf("callRPC arr len%d", len(arr))
		return resp, reflect.ValueOf(errors.New("illegal grpc call response"))
	}
	return arr[0], arr[1]
}
