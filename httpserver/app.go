package httpserver

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	common "../proto/common"
	discover "../proto/discover"
	fetch "../proto/fetch"
	helloworld "../proto/hello"
	hot "../proto/hot"
	modify "../proto/modify"
	verify "../proto/verify"
	util "../util"
	simplejson "github.com/bitly/go-simplejson"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

func getMessage(name string) string {
	conn, err := grpc.Dial(helloAddress, grpc.WithInsecure())
	if err != nil {
		log.Printf("did not connect: %v", err)
		return ""
	}
	defer conn.Close()
	c := helloworld.NewGreeterClient(conn)

	r, err := c.SayHello(context.Background(), &helloworld.HelloRequest{Name: name})
	if err != nil {
		log.Printf("could not greet: %v", err)
		return ""
	}

	return r.Message
}

func welcome(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Welcome to Yunti~"))
}

func hello(w http.ResponseWriter, r *http.Request) {
	name := "hello"
	message := getMessage(name)
	w.Write([]byte(message))
}

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

	conn, err := grpc.Dial(verifyAddress, grpc.WithInsecure())
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
	conn, err := grpc.Dial(verifyAddress, grpc.WithInsecure())
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

	conn, err := grpc.Dial(verifyAddress, grpc.WithInsecure())
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
	username := req.GetParamString("username")
	password := req.GetParamString("password")
	longitude := req.GetParamFloat("longitude")
	latitude := req.GetParamFloat("latitude")

	conn, err := grpc.Dial(modifyAddress, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := modify.NewModifyClient(conn)

	uuid := util.GenUUID()
	res, err := c.AddWifi(context.Background(), &modify.WifiRequest{Head: &common.Head{Sid: uuid, Uid: uid}, Info: &common.WifiInfo{Ssid: ssid, Username: username, Password: password, Longitude: longitude, Latitude: latitude}})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}

	if res.Head.Retcode != 0 {
		return &util.AppError{util.LogicErr, 4, "AddWifi failed"}
	}

	w.Write([]byte(`{"errno":0}`))
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

	conn, err := grpc.Dial(modifyAddress, grpc.WithInsecure())
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

	conn, err := grpc.Dial(fetchAddress, grpc.WithInsecure())
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
	infos := make([]interface{}, len(res.Infos))
	for i := 0; i < len(res.Infos); i++ {
		json, _ := simplejson.NewJson([]byte(`{}`))
		json.Set("ssid", res.Infos[i].Ssid)
		json.Set("username", res.Infos[i].Username)
		json.Set("password", res.Infos[i].Password)
		json.Set("longitude", res.Infos[i].Longitude)
		json.Set("latitude", res.Infos[i].Latitude)
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

func getFrontInfo(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	defer func() {
		if r := recover(); r != nil {
			apperr = extractError(r)
		}
	}()
	var req request
	req.initCheckApp(r.Body)
	uid := req.GetParamInt("uid")

	conn, err := grpc.Dial(hotAddress, grpc.WithInsecure())
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
	json, _ := simplejson.NewJson([]byte(`{}`))
	json.Set("total", res.Uinfo.Total)
	json.Set("save", res.Uinfo.Save)
	js.SetPath([]string{"data", "uinfo"}, json)

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

	conn, err := grpc.Dial(hotAddress, grpc.WithInsecure())
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
	w.Write(body)
	return nil
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

	conn, err := grpc.Dial(hotAddress, grpc.WithInsecure())
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
	w.Write(body)
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

	conn, err := grpc.Dial(verifyAddress, grpc.WithInsecure())
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

	conn, err := grpc.Dial(hotAddress, grpc.WithInsecure())
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
	tops := make([]interface{}, len(res.Tops))
	for i := 0; i < len(res.Tops); i++ {
		json, _ := simplejson.NewJson([]byte(`{}`))
		json.Set("title", res.Tops[i].Title)
		json.Set("icon", res.Tops[i].Icon)
		json.Set("dst", res.Tops[i].Dst)
		tops[i] = json
	}
	js.SetPath([]string{"data", "top"}, tops)

	services := make([]interface{}, len(res.Services))
	for i := 0; i < len(res.Services); i++ {
		json, _ := simplejson.NewJson([]byte(`{}`))
		json.Set("title", res.Services[i].Title)
		items := make([]interface{}, len(res.Services[i].Infos))
		for j := 0; j < len(res.Services[i].Infos); j++ {
			in, _ := simplejson.NewJson([]byte(`{}`))
			in.Set("title", res.Services[i].Infos[j].Title)
			in.Set("icon", res.Services[i].Infos[j].Icon)
			in.Set("dst", res.Services[i].Infos[j].Dst)
			items[j] = in
		}
		json.Set("items", items)
		services[i] = json
	}
	js.SetPath([]string{"data", "services"}, services)
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
	code := req.GetParamInt("code")
	udid := req.GetParamString("udid")
	model := req.GetParamString("model")
	channel := req.GetParamString("channel")
	regip := extractIP(r.RemoteAddr)

	conn, err := grpc.Dial(verifyAddress, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := verify.NewVerifyClient(conn)

	uuid := util.GenUUID()
	res, err := c.Register(context.Background(), &verify.RegisterRequest{Head: &common.Head{Sid: uuid}, Username: username, Password: password, Code: int32(code), Udid: udid, Model: model, Channel: channel, Regip: regip})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}

	if res.Head.Retcode == common.ErrCode_USED_PHONE {
		return &util.AppError{util.LogicErr, 104, "该手机号已注册，请直接登录"}
	} else if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "服务器又傲娇了"}
	}

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "init json failed"}
	}

	js.SetPath([]string{"data", "uid"}, res.Head.Uid)
	js.SetPath([]string{"data", "token"}, res.Token)
	js.SetPath([]string{"data", "privdata"}, res.Privdata)
	js.SetPath([]string{"data", "expire"}, res.Expire)
	js.SetPath([]string{"data", "wifipass"}, res.Wifipass)
	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func discoverServer(w http.ResponseWriter, r *http.Request) {
	post, err := simplejson.NewFromReader(r.Body)
	if err != nil {
		log.Printf("parse request body failed:%v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}

	name, err := post.Get("name").String()
	if err != nil {
		log.Printf("get name failed:%v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}

	conn, err := grpc.Dial(discoverAddress, grpc.WithInsecure())
	if err != nil {
		log.Printf("did not connect: %v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}
	defer conn.Close()
	c := discover.NewDiscoverClient(conn)

	uuid := util.GenUUID()
	res, err := c.Resolve(context.Background(), &discover.ServerRequest{Head: &common.Head{Sid: uuid}, Sname: name})
	if err != nil {
		log.Printf("Login failed: %v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}

	if res.Head.Retcode == common.ErrCode_USED_PHONE {
		w.Write([]byte(`{"errno":104,"desc":"该手机号已注册，请直接登录"}`))
		return
	}

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		log.Printf("NewJson failed: %v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}

	js.SetPath([]string{"data", "host"}, res.Host)
	js.SetPath([]string{"data", "port"}, res.Port)
	body, err := js.MarshalJSON()
	if err != nil {
		log.Printf("MarshalJSON failed: %v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}
	w.Write(body)

}

func wxMpLogin(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	code := r.Form["code"]
	if len(code) == 0 {
		log.Printf("get code failed\n")
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}
	echostr := r.Form["echostr"]

	conn, err := grpc.Dial(verifyAddress, grpc.WithInsecure())
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

	dst := fmt.Sprintf("%s?uid=%d&token=%s", echostr[0], res.Head.Uid, res.Token)
	http.Redirect(w, r, dst, http.StatusMovedPermanently)
}

func live(w http.ResponseWriter, r *http.Request) {
	redirect := "http://wx.youcaitv.cn/wx_mp_login"
	dst := util.GenRedirectURL(redirect)
	http.Redirect(w, r, dst, http.StatusMovedPermanently)
}

//ServeApp do app server work
func ServeApp() {
	http.HandleFunc("/hello", hello)
	http.Handle("/login", appHandler(login))
	http.Handle("/get_phone_code", appHandler(getPhoneCode))
	http.Handle("/register", appHandler(register))
	http.Handle("/logout", appHandler(logout))
	http.Handle("/hot", appHandler(getHot))
	http.Handle("/get_weather_news", appHandler(getWeatherNews))
	http.Handle("/get_front_info", appHandler(getFrontInfo))
	http.Handle("/fetch_wifi", appHandler(fetchWifi))
	http.Handle("/auto_login", appHandler(autoLogin))
	http.Handle("/get_nearby_aps", appHandler(getAppAps))
	http.Handle("/report_wifi", appHandler(reportWifi))
	http.Handle("/report_click", appHandler(reportClick))
	http.Handle("/services", appHandler(getService))
	http.HandleFunc("/discover", discoverServer)
	http.HandleFunc("/live", live)
	http.HandleFunc("/wx_mp_login", wxMpLogin)
	http.Handle("/", http.FileServer(http.Dir("/data/server/html")))
	http.ListenAndServe(":80", nil)
}
