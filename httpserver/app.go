package httpserver

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	common "../proto/common"
	fetch "../proto/fetch"
	hot "../proto/hot"
	modify "../proto/modify"
	verify "../proto/verify"
	util "../util"
	simplejson "github.com/bitly/go-simplejson"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

func login(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	defer func() {
		if r := recover(); r != nil {
			apperr = extractError(r)
		}
	}()
	var req request
	req.init(r.Body)
	username := req.GetParamString("username")
	password := req.GetParamString("password")
	model := req.GetParamString("model")
	udid := req.GetParamString("udid")

	address := getNameServer(0, util.VerifyServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := verify.NewVerifyClient(conn)

	uuid := util.GenUUID()
	res, err := c.Login(context.Background(), &verify.LoginRequest{Head: &common.Head{Sid: uuid}, Username: username, Password: password, Model: model, Udid: udid})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}

	if res.Head.Retcode != 0 {
		return &util.AppError{util.LogicErr, int(res.Head.Retcode), "登录失败"}
	}

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{util.JSONErr, 4, err.Error()}
	}

	js.SetPath([]string{"data", "uid"}, res.Head.Uid)
	js.SetPath([]string{"data", "token"}, res.Token)
	js.SetPath([]string{"data", "privdata"}, res.Privdata)
	js.SetPath([]string{"data", "expire"}, res.Expire)
	js.SetPath([]string{"data", "wifipass"}, res.Wifipass)
	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{util.JSONErr, 4, err.Error()}
	}
	w.Write(body)
	return nil
}

func getCode(phone string, ctype int32) (bool, error) {
	address := getNameServer(0, util.VerifyServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Printf("did not connect: %v", err)
		return false, err
	}
	defer conn.Close()
	c := verify.NewVerifyClient(conn)

	uuid := util.GenUUID()
	r, err := c.GetPhoneCode(context.Background(), &verify.CodeRequest{Head: &common.Head{Sid: uuid}, Phone: phone, Ctype: ctype})
	if err != nil {
		log.Printf("could not get phone code: %v", err)
		return false, err
	}

	return r.Result, nil
}

func getPhoneCode(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	defer func() {
		if r := recover(); r != nil {
			apperr = extractError(r)
		}
	}()
	var req request
	req.init(r.Body)
	phone := req.GetParamString("phone")
	ctype := req.GetParamInt("type")

	flag, err := getCode(phone, int32(ctype))
	if err != nil || !flag {
		return &util.AppError{util.LogicErr, 103, "获取验证码失败"}
	}
	w.Write([]byte(`{"errno":0}`))
	return nil
}

func logout(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	defer func() {
		if r := recover(); r != nil {
			apperr = extractError(r)
		}
	}()
	var req request
	req.init(r.Body)
	uid := req.GetParamInt("uid")
	token := req.GetParamString("token")

	address := getNameServer(uid, util.VerifyServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := verify.NewVerifyClient(conn)

	uuid := util.GenUUID()
	res, err := c.Logout(context.Background(), &verify.LogoutRequest{Head: &common.Head{Sid: uuid, Uid: uid}, Token: token})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}

	if res.Head.Retcode != 0 {
		return &util.AppError{util.LogicErr, 4, "logout failed"}
	}

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func reportWifi(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	defer func() {
		if r := recover(); r != nil {
			apperr = extractError(r)
		}
	}()
	var req request
	req.initCheckApp(r.Body)
	uid := req.GetParamInt("uid")
	ssid := req.GetParamString("ssid")
	password := req.GetParamString("password")
	longitude := req.GetParamFloat("longitude")
	latitude := req.GetParamFloat("latitude")

	address := getNameServer(uid, util.ModifyServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := modify.NewModifyClient(conn)

	uuid := util.GenUUID()
	res, err := c.AddWifi(context.Background(), &modify.WifiRequest{Head: &common.Head{Sid: uuid, Uid: uid}, Info: &common.WifiInfo{Ssid: ssid, Password: password, Longitude: longitude, Latitude: latitude}})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}

	if res.Head.Retcode != 0 {
		return &util.AppError{util.LogicErr, 4, "AddWifi failed"}
	}

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func reportApmac(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	defer func() {
		if r := recover(); r != nil {
			apperr = extractError(r)
		}
	}()
	var req request
	req.initCheckApp(r.Body)
	uid := req.GetParamInt("uid")
	apmac := req.GetParamString("apmac")
	log.Printf("report_apmac uid:%d apmac:%s\n", uid, apmac)

	address := getNameServer(uid, util.ModifyServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := modify.NewModifyClient(conn)

	uuid := util.GenUUID()
	res, err := c.ReportApmac(context.Background(), &modify.ApmacRequest{Head: &common.Head{Sid: uuid, Uid: uid}, Apmac: apmac})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}

	if res.Head.Retcode != 0 {
		return &util.AppError{util.LogicErr, 4, "ReportApmac failed"}
	}

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func uploadCallback(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	defer func() {
		if r := recover(); r != nil {
			apperr = extractError(r)
		}
	}()
	r.ParseForm()
	fname := r.Form["filename"]
	if len(fname) < 1 {
		log.Printf("parse filename failed\n")
		w.Write([]byte(`{"Status":"OK"}`))
		return nil
	}
	size := r.Form["size"]
	fsize, _ := strconv.Atoi(size[0])
	height := r.Form["height"]
	fheight, _ := strconv.Atoi(height[0])
	width := r.Form["width"]
	fwidth, _ := strconv.Atoi(width[0])
	log.Printf("upload_callback fname:%s size:%d height:%d width:%d\n", fname, fsize,
		fheight, fwidth)

	address := getNameServer(0, util.ModifyServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := modify.NewModifyClient(conn)

	uuid := util.GenUUID()
	res, err := c.FinImage(context.Background(),
		&modify.ImageRequest{Head: &common.Head{Sid: uuid},
			Info: &modify.ImageInfo{Name: fname[0], Size: int64(fsize),
				Height: int32(fheight), Width: int32(fwidth)}})
	if err != nil {
		log.Printf("FinImage failed:%v", err)
	}

	if res.Head.Retcode != 0 {
		log.Printf("FinImage failed retcode:%d", res.Head.Retcode)
	}

	w.Write([]byte(`{"Status":"OK"}`))
	return nil
}

func reportClick(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	defer func() {
		if r := recover(); r != nil {
			apperr = extractError(r)
		}
	}()
	var req request
	req.initCheckApp(r.Body)
	uid := req.GetParamInt("uid")
	id := req.GetParamInt("id")
	ctype := req.GetParamInt("type")

	address := getNameServer(uid, util.ModifyServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := modify.NewModifyClient(conn)

	uuid := util.GenUUID()
	res, err := c.ReportClick(context.Background(), &modify.ClickRequest{Head: &common.Head{Sid: uuid, Uid: uid}, Id: id, Type: int32(ctype)})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}

	if res.Head.Retcode != 0 {
		return &util.AppError{util.LogicErr, 4, "ReportClick failed"}
	}

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func fetchWifi(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	defer func() {
		if r := recover(); r != nil {
			apperr = extractError(r)
		}
	}()
	var req request
	req.initCheckApp(r.Body)
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
	res, err := c.FetchWifi(context.Background(), &fetch.WifiRequest{Head: &common.Head{Sid: uuid, Uid: uid}, Longitude: longitude, Latitude: latitude})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "获取共享wifi失败"}
	}

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "invalid param"}
	}
	js.SetPath([]string{"data", "infos"}, res.Infos)

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func getFrontInfo(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	defer func() {
		if r := recover(); r != nil {
			apperr = extractError(r)
		}
	}()
	var req request
	req.initCheckApp(r.Body)
	uid := req.GetParamInt("uid")

	address := getNameServer(uid, util.HotServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := hot.NewHotClient(conn)

	uuid := util.GenUUID()
	res, err := c.GetFrontInfo(context.Background(), &hot.HotsRequest{Head: &common.Head{Sid: uuid, Uid: uid}})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "获取首页信息失败"}
	}

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "invalid param"}
	}
	js.SetPath([]string{"data", "user"}, res.Uinfo)
	js.SetPath([]string{"data", "banner"}, res.Binfos)

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func getWeatherNews(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	defer func() {
		if r := recover(); r != nil {
			apperr = extractError(r)
		}
	}()
	var req request
	req.initCheckApp(r.Body)
	uid := req.GetParamInt("uid")
	resp, err := getRspFromSSDB(hotWeatherKey)
	if err == nil {
		log.Printf("getRspFromSSDB succ key:%s\n", hotWeatherKey)
		rspGzip(w, []byte(resp))
		return nil
	}

	address := getNameServer(uid, util.HotServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := hot.NewHotClient(conn)

	uuid := util.GenUUID()
	res, err := c.GetWeatherNews(context.Background(), &hot.HotsRequest{Head: &common.Head{Sid: uuid, Uid: uid}})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "获取新闻失败"}
	}

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "invalid param"}
	}
	infos := make([]interface{}, len(res.News))
	for i := 0; i < len(res.News); i++ {
		json, _ := simplejson.NewJson([]byte(`{}`))
		json.Set("id", res.News[i].Seq)
		json.Set("title", res.News[i].Title)
		if len(res.News[i].Images) > 0 {
			json.Set("images", res.News[i].Images)
		}
		json.Set("source", res.News[i].Source)
		json.Set("dst", res.News[i].Dst)
		json.Set("ctime", res.News[i].Ctime)
		json.Set("play", res.News[i].Play)
		infos[i] = json
	}
	js.SetPath([]string{"data", "news"}, infos)

	json, _ := simplejson.NewJson([]byte(`{}`))
	json.Set("temp", res.Weather.Temp)
	json.Set("type", res.Weather.Type)
	json.Set("info", res.Weather.Info)
	js.SetPath([]string{"data", "weather"}, json)

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	rspGzip(w, body)
	data := js.Get("data")
	setSSDBCache(hotWeatherKey, data)
	return nil
}

func genSsdbKey(ctype int64) string {
	switch ctype {
	default:
		return hotNewsKey
	case 1:
		return hotVideoKey
	}
}

func getRspFromSSDB(key string) (string, error) {
	val, err := util.GetSSDBVal(key)
	if err != nil {
		log.Printf("getRspFromSSDB GetSSDBVal key:%s failed:%v", key, err)
		return "", err
	}
	js, err := simplejson.NewJson([]byte(val))
	if err != nil {
		log.Printf("getRspFromSSDB parse json failed:%v", err)
		return "", err
	}
	expire, err := js.Get("expire").Int64()
	if err != nil {
		log.Printf("getRspFromSSDB get expire failed:%v", err)
		return "", err
	}
	if time.Now().Unix() > expire {
		log.Printf("getRspFromSSDB data expire :%d", expire)
		return "", errors.New("ssdb data expired")
	}
	rsp, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		log.Printf("getRspFromSSDB NewJson failed:%v", err)
		return "", err
	}
	data := js.Get("data")
	rsp.Set("data", data)

	body, err := rsp.MarshalJSON()
	if err != nil {
		log.Printf("getRspFromSSDB MarshalJson failed:%v", err)
		return "", err
	}

	return string(body), nil
}

func setSSDBCache(key string, data *simplejson.Json) {
	expire := time.Now().Unix() + expireInterval
	js, err := simplejson.NewJson([]byte(`{}`))
	if err != nil {
		log.Printf("setSSDBCache key:%s NewJson failed:%v\n", key, err)
		return
	}
	js.Set("expire", expire)
	js.Set("data", data)
	body, err := js.MarshalJSON()
	if err != nil {
		log.Printf("setSSDBCache MarshalJson failed:%v", err)
		return
	}
	util.SetSSDBVal(key, string(body))
	return
}

func getHot(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	defer func() {
		if r := recover(); r != nil {
			apperr = extractError(r)
		}
	}()
	var req request
	req.initCheckApp(r.Body)
	uid := req.GetParamInt("uid")
	ctype := req.GetParamInt("type")
	seq := req.GetParamInt("seq")
	log.Printf("uid:%d ctype:%d seq:%d\n", uid, ctype, seq)
	if seq == 0 {
		key := genSsdbKey(ctype)
		log.Printf("key:%s", key)
		resp, err := getRspFromSSDB(key)
		if err == nil {
			log.Printf("getRspFromSSDB succ key:%s\n", key)
			rspGzip(w, []byte(resp))
			return nil
		}
		log.Printf("getRspFromSSDB failed key:%s err:%v\n", key, err)
	}

	address := getNameServer(uid, util.HotServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := hot.NewHotClient(conn)

	uuid := util.GenUUID()
	res, err := c.GetHots(context.Background(), &hot.HotsRequest{Head: &common.Head{Sid: uuid, Uid: uid}, Type: int32(ctype), Seq: int32(seq)})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "获取新闻失败"}
	}

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "invalid param"}
	}
	infos := make([]interface{}, len(res.Infos))
	for i := 0; i < len(res.Infos); i++ {
		json, _ := simplejson.NewJson([]byte(`{}`))
		json.Set("seq", res.Infos[i].Seq)
		json.Set("id", res.Infos[i].Seq)
		json.Set("title", res.Infos[i].Title)
		if len(res.Infos[i].Images) > 0 {
			json.Set("images", res.Infos[i].Images)
		}
		json.Set("source", res.Infos[i].Source)
		json.Set("dst", res.Infos[i].Dst)
		json.Set("ctime", res.Infos[i].Ctime)
		if res.Infos[i].Stype == 11 {
			json.Set("stype", 1)
		}
		json.Set("play", res.Infos[i].Play)
		infos[i] = json
	}
	js.SetPath([]string{"data", "infos"}, infos)
	if len(res.Infos) >= util.MaxListSize {
		js.SetPath([]string{"data", "hasmore"}, 1)
	}

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	rspGzip(w, body)
	if seq == 0 {
		key := genSsdbKey(ctype)
		data := js.Get("data")
		setSSDBCache(key, data)
	}
	return nil
}

func autoLogin(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	defer func() {
		if r := recover(); r != nil {
			apperr = extractError(r)
		}
	}()
	var req request
	req.init(r.Body)
	uid := req.GetParamInt("uid")
	token := req.GetParamString("token")
	privdata := req.GetParamString("privdata")

	address := getNameServer(uid, util.VerifyServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := verify.NewVerifyClient(conn)

	uuid := util.GenUUID()
	res, err := c.AutoLogin(context.Background(), &verify.AutoRequest{Head: &common.Head{Uid: uid, Sid: uuid}, Token: token, Privdata: privdata})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}

	if res.Head.Retcode == common.ErrCode_INVALID_TOKEN {
		return &util.AppError{util.LogicErr, 4, "token验证失败"}
	} else if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "服务器又傲娇了"}
	}

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "init json failed"}
	}

	js.SetPath([]string{"data", "token"}, res.Token)
	js.SetPath([]string{"data", "privdata"}, res.Privdata)
	js.SetPath([]string{"data", "expire"}, res.Expire)
	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func getService(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	defer func() {
		if r := recover(); r != nil {
			apperr = extractError(r)
		}
	}()
	var req request
	req.initCheckApp(r.Body)
	uid := req.GetParamInt("uid")

	address := getNameServer(uid, util.HotServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := hot.NewHotClient(conn)
	uuid := util.GenUUID()
	res, err := c.GetServices(context.Background(), &hot.ServiceRequest{Head: &common.Head{Uid: uid, Sid: uuid}})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}

	if res.Head.Retcode == common.ErrCode_INVALID_TOKEN {
		return &util.AppError{util.LogicErr, 4, "token验证失败"}
	} else if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "服务器又傲娇了"}
	}

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "init json failed"}
	}
	js.SetPath([]string{"data", "services"}, res.Services)
	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func getAppAps(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	return getAps(w, r, false)
}

func extractIP(addr string) string {
	arr := strings.Split(addr, ":")
	return arr[0]
}

func register(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	defer func() {
		if r := recover(); r != nil {
			apperr = extractError(r)
		}
	}()
	var req request
	req.init(r.Body)
	username := req.GetParamString("username")
	password := req.GetParamString("password")
	udid := req.GetParamString("udid")
	model := req.GetParamString("model")
	channel := req.GetParamString("channel")
	regip := extractIP(r.RemoteAddr)
	log.Printf("register request username:%s password:%s udid:%s model:%s channel:%s", username, password, udid, model, channel)

	address := getNameServer(0, util.VerifyServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := verify.NewVerifyClient(conn)

	uuid := util.GenUUID()
	res, err := c.Register(context.Background(), &verify.RegisterRequest{Head: &common.Head{Sid: uuid}, Username: username, Password: password, Udid: udid, Model: model, Channel: channel, Regip: regip})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}

	if res.Head.Retcode == common.ErrCode_USED_PHONE {
		return &util.AppError{util.LogicErr, 104, "该账号已注册，请直接登录"}
	} else if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "服务器又傲娇了"}
	}

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "init json failed"}
	}

	log.Printf("register resp uid:%d token:%s privdata:%s", res.Head.Uid, res.Token, res.Privdata)
	js.SetPath([]string{"data", "uid"}, res.Head.Uid)
	js.SetPath([]string{"data", "token"}, res.Token)
	js.SetPath([]string{"data", "privdata"}, res.Privdata)
	js.SetPath([]string{"data", "expire"}, res.Expire)
	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func wxMpLogin(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
		}
	}()

	r.ParseForm()
	code := r.Form["code"]
	if len(code) == 0 {
		log.Printf("get code failed\n")
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}
	echostr := r.Form["echostr"]

	address := getNameServer(0, util.VerifyServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Printf("did not connect: %v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}
	defer conn.Close()
	c := verify.NewVerifyClient(conn)

	log.Printf("code:%s\n", code[0])
	uuid := util.GenUUID()
	res, err := c.WxMpLogin(context.Background(), &verify.LoginRequest{Head: &common.Head{Sid: uuid}, Code: code[0]})
	if err != nil {
		log.Printf("Login failed: %v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}

	if res.Head.Retcode != 0 {
		w.Write([]byte(`{"errno":105,"desc":"微信公众号登录失败"}`))
		return
	}

	if len(echostr) == 0 {
		rs := fmt.Sprintf(`{"errno":0, "uid":%d, "token":%s"}`, res.Head.Uid, res.Token)
		w.Write([]byte(rs))
		return
	}

	dst := fmt.Sprintf("%s?uid=%d&token=%s&union=%s", echostr[0], res.Head.Uid, res.Token, res.Privdata)
	http.Redirect(w, r, dst, http.StatusMovedPermanently)
}

func jump(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	file := r.Form["echofile"]
	var echostr string
	if len(file) > 0 {
		echostr = file[0]
		echostr = "http://wx.youcaitv.cn/" + echostr
	}
	ck, err := r.Cookie("UNION")
	if err == nil {
		log.Printf("get cookie UNION succ:%s", ck.Value)
		address := getNameServer(0, util.VerifyServerName)
		conn, err := grpc.Dial(address, grpc.WithInsecure())
		if err != nil {
			log.Printf("did not connect: %v", err)
			w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
			return
		}
		defer conn.Close()
		c := verify.NewVerifyClient(conn)

		uuid := util.GenUUID()
		res, err := c.UnionLogin(context.Background(), &verify.LoginRequest{Head: &common.Head{Sid: uuid}, Unionid: ck.Value})
		if err != nil {
			log.Printf("UnionLogin failed: %v", err)
			w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
			return
		}

		if res.Head.Retcode != 0 {
			w.Write([]byte(`{"errno":106,"desc":"微信公众号登录失败"}`))
			return
		}
		dst := fmt.Sprintf("%s?uid=%d&token=%s", echostr, res.Head.Uid, res.Token)
		http.Redirect(w, r, dst, http.StatusMovedPermanently)
		return
	}
	redirect := "http://wx.youcaitv.cn/wx_mp_login"
	redirect += "?echostr=" + echostr
	dst := util.GenRedirectURL(redirect)
	http.Redirect(w, r, dst, http.StatusMovedPermanently)
}

func genNonce() string {
	nonce := "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	var res []byte
	r := rand.New(rand.NewSource(time.Now().Unix()))
	for i := 0; i < 12; i++ {
		ch := nonce[r.Int31n(int32(len(nonce)))]
		res = append(res, ch)
	}
	return string(res)
}

func getJsapiSign(w http.ResponseWriter, r *http.Request) {
	address := getNameServer(0, util.VerifyServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Printf("did not connect: %v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}
	defer conn.Close()
	c := verify.NewVerifyClient(conn)

	uuid := util.GenUUID()
	res, err := c.GetWxTicket(context.Background(), &verify.TicketRequest{Head: &common.Head{Sid: uuid}})
	if err != nil {
		log.Printf("GetWxTicket failed: %v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}

	if res.Head.Retcode != 0 {
		w.Write([]byte(`{"errno":107,"desc":"获取微信ticket失败"}`))
		return
	}

	noncestr := genNonce()
	ts := time.Now().Unix()
	url := r.Referer()
	pos := strings.Index(url, "#")
	if pos != -1 {
		url = url[:pos]
	}

	ori := fmt.Sprintf("jsapi_ticket=%s&noncestr=%s&timestamp=%d&url=%s", res.Ticket, noncestr, ts, url)
	sign := util.Sha1(ori)
	log.Printf("origin:%s sign:%s\n", ori, sign)
	out := fmt.Sprintf("var wx_cfg={\"debug\":false, \"appId\":\"%s\",\"timestamp\":%d,\"nonceStr\":\"%s\",\"signature\":\"%s\",\"jsApiList\":[],\"jsapi_ticket\":\"%s\"};", util.WxAppid, ts, noncestr, sign, res.Ticket)
	w.Write([]byte(out))
	return
}

//NewAppServer return app http handler
func NewAppServer() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/login", appHandler(login))
	mux.Handle("/get_phone_code", appHandler(getPhoneCode))
	mux.Handle("/register", appHandler(register))
	mux.Handle("/logout", appHandler(logout))
	mux.Handle("/hot", appHandler(getHot))
	mux.Handle("/get_weather_news", appHandler(getWeatherNews))
	mux.Handle("/get_front_info", appHandler(getFrontInfo))
	mux.Handle("/fetch_wifi", appHandler(fetchWifi))
	mux.Handle("/auto_login", appHandler(autoLogin))
	mux.Handle("/get_nearby_aps", appHandler(getAppAps))
	mux.Handle("/report_wifi", appHandler(reportWifi))
	mux.Handle("/report_click", appHandler(reportClick))
	mux.Handle("/report_apmac", appHandler(reportApmac))
	mux.Handle("/upload_callback", appHandler(uploadCallback))
	mux.Handle("/services", appHandler(getService))
	mux.HandleFunc("/jump", jump)
	mux.HandleFunc("/wx_mp_login", wxMpLogin)
	mux.HandleFunc("/get_jsapi_sign", getJsapiSign)
	mux.Handle("/", http.FileServer(http.Dir("/data/server/html")))
	return mux
}
