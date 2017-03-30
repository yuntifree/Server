package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"Server/aliyun"
	"Server/httpserver"
	"Server/pay"
	"Server/proto/advertise"
	"Server/proto/common"
	"Server/proto/config"
	"Server/proto/fetch"
	"Server/proto/punch"
	"Server/proto/userinfo"

	"Server/proto/hot"

	"Server/proto/modify"
	"Server/proto/verify"
	"Server/util"

	simplejson "github.com/bitly/go-simplejson"
	pingpp "github.com/pingplusplus/pingpp-go/pingpp"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const (
	wxHost       = "http://wx.yunxingzh.com/"
	maxZipcode   = 820000
	portalDst    = "http://120.25.133.234/"
	postLoginURL = "http://wx.yunxingzh.com/wx/wxlogin201703271528/wxpostlogin.html"
	succLoginURL = "http://wx.yunxingzh.com/wx/wxlogin201703271528/wxloginsucc.html"
	defLoginURL  = "http://192.168.100.4:8080/login201703171857/"
)
const (
	hotNewsKey         = "hot:news"
	hotVideoKey        = "hot:video"
	hotWeatherKey      = "hot:weather"
	hotServiceKey      = "hot:service"
	hotDgNewsKey       = "hot:news:dg"
	hotAmuseKey        = "hot:news:amuse"
	hotJokeKey         = "hot:joke"
	hotNewsCompKey     = "hot:news:comp"
	hotAllApsKey       = "hot:all:aps"
	configDiscoveryKey = "config:discovery"
	expireInterval     = 30
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

type portalDir struct {
	Dir    string
	Expire int64
}

var pdir = portalDir{
	Dir:    "dist/",
	Expire: time.Now().Unix(),
}

func login(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.Init(r)
	username := req.GetParamString("username")
	password := req.GetParamString("password")
	model := req.GetParamString("model")
	udid := req.GetParamString("udid")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.VerifyServerType, 0, "Login",
		&verify.LoginRequest{Head: &common.Head{Sid: uuid},
			Username: username, Password: password, Model: model, Udid: udid})
	httpserver.CheckRPCErr(rpcerr, "Login")
	res := resp.Interface().(*verify.LoginReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "Login")

	body := httpserver.GenResponseBody(res, true)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func getCode(phone string, ctype int64) (bool, error) {
	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.VerifyServerType, 0, "GetPhoneCode",
		&verify.CodeRequest{Head: &common.Head{Sid: uuid},
			Phone: phone, Ctype: ctype})
	httpserver.CheckRPCErr(rpcerr, "GetPhoneCode")
	res := resp.Interface().(*verify.VerifyReply)

	return res.Result, nil
}

func getPhoneCode(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.Init(r)
	phone := req.GetParamString("phone")
	ctype := req.GetParamIntDef("type", 0)

	if !util.IsIllegalPhone(phone) {
		log.Printf("getPhoneCode illegal phone:%s", phone)
		return &util.AppError{Code: httpserver.ErrIllegalPhone,
			Msg: "请输入正确的手机号"}
	}

	flag, err := getCode(phone, ctype)
	if err != nil || !flag {
		return &util.AppError{Code: httpserver.ErrCode, Msg: "获取验证码失败"}
	}
	w.Write([]byte(`{"errno":0}`))
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func getCheckCode(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.Init(r)
	phone := req.GetParamString("phone")
	acname := req.GetParamStringDef("wlanacname", "")
	term := req.GetParamInt("term")

	if !util.IsIllegalPhone(phone) {
		log.Printf("getCheckCode illegal phone:%s", phone)
		return &util.AppError{Code: httpserver.ErrIllegalPhone,
			Msg: "请输入正确的手机号", Callback: req.Callback}
	}

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPCCallback(util.VerifyServerType,
		0, "GetCheckCode", req.Callback,
		&verify.PortalLoginRequest{Head: &common.Head{Sid: uuid, Term: term},
			Info: &verify.PortalInfo{Phone: phone, Acname: acname}})
	httpserver.CheckRPCErrCallback(rpcerr, "GetPhoneCode", req.Callback)
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCodeCallback(res.Head.Retcode, "GetPhoneCode",
		req.Callback)

	req.WriteRsp(w, []byte(`{"errno":0}`))
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func logout(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.Init(r)
	uid := req.GetParamInt("uid")
	token := req.GetParamString("token")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.VerifyServerType, uid, "Logout",
		&verify.LogoutRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Token: token})
	httpserver.CheckRPCErr(rpcerr, "Logout")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "Logout")

	w.Write([]byte(`{"errno":0}`))
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func reportWifi(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")
	ssid := req.GetParamString("ssid")
	password := req.GetParamString("password")
	longitude := req.GetParamFloat("longitude")
	latitude := req.GetParamFloat("latitude")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ModifyServerType, uid, "AddWifi",
		&modify.WifiRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Info: &common.WifiInfo{Ssid: ssid, Password: password, Longitude: longitude,
				Latitude: latitude}})
	httpserver.CheckRPCErr(rpcerr, "AddWifi")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "AddWifi")

	w.Write([]byte(`{"errno":0}`))
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func connectWifi(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")
	acname := req.GetParamString("wlanacname")
	acip := req.GetParamString("wlanacip")
	userip := req.GetParamString("wlanuserip")
	usermac := req.GetParamString("wlanusermac")
	apmac := req.GetParamString("apmac")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.VerifyServerType, uid, "WifiAccess",
		&verify.AccessRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Info: &verify.PortalInfo{Userip: userip, Usermac: usermac, Acname: acname,
				Acip: acip, Apmac: apmac}})
	httpserver.CheckRPCErr(rpcerr, "WifiAccess")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "WifiAccess")

	w.Write([]byte(`{"errno":0}`))
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func addAddress(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")
	province := req.GetParamInt("province")
	city := req.GetParamInt("city")
	zone := req.GetParamInt("zone")
	if province >= maxZipcode || city >= maxZipcode || zone >= maxZipcode {
		return &util.AppError{Code: httpserver.ErrInvalidParam,
			Msg: "illegal zipcode"}
	}
	zip := req.GetParamInt("zip")
	detail := req.GetParamString("detail")
	mobile := req.GetParamString("mobile")
	user := req.GetParamString("user")
	addr := req.GetParamString("addr")
	def := req.GetParamBoolDef("def", false)

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ModifyServerType, uid, "AddAddress",
		&modify.AddressRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Info: &common.AddressInfo{Province: province, City: city,
				Zone: zone, Zip: zip, Addr: addr, Detail: detail,
				Def: def, User: user, Mobile: mobile}})
	httpserver.CheckRPCErr(rpcerr, "AddAddress")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "AddAddress")

	body := httpserver.GenResponseBody(res, false)

	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func addShare(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")
	bid := req.GetParamInt("bid")
	title := req.GetParamString("title")
	text := req.GetParamString("text")
	images, err := req.Post.Get("data").Get("images").Array()
	if err != nil {
		return &util.AppError{Code: httpserver.ErrInvalidParam,
			Msg: err.Error()}
	}
	var imgs []string
	for i := 0; i < len(images); i++ {
		img := images[i].(string)
		imgs = append(imgs, img)
	}

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ModifyServerType, uid, "AddShare",
		&modify.ShareRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Bid: bid, Title: title, Text: text, Images: imgs})
	httpserver.CheckRPCErr(rpcerr, "AddShare")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "AddShare")

	w.Write([]byte(`{"errno":0}`))
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func setWinStatus(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")
	bid := req.GetParamInt("bid")
	status := req.GetParamInt("status")
	aid := req.GetParamIntDef("aid", 0)
	account := req.GetParamStringDef("account", "")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ModifyServerType, uid, "SetWinStatus",
		&modify.WinStatusRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Bid: bid, Status: status, Aid: aid, Account: account})
	httpserver.CheckRPCErr(rpcerr, "SetWinStatus")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "SetWinStatus")

	w.Write([]byte(`{"errno":0}`))
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func addFeedback(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")
	content := req.GetParamString("content")
	contact := req.GetParamStringDef("contact", "")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ModifyServerType, uid, "AddFeedback",
		&modify.FeedRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Content: content, Contact: contact})
	httpserver.CheckRPCErr(rpcerr, "AddFeedback")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "AddFeedback")

	w.Write([]byte(`{"errno":0}`))
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func purchaseSales(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")
	bid := req.GetParamInt("bid")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ModifyServerType, uid, "PurchaseSales",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Id: bid})
	httpserver.CheckRPCErr(rpcerr, "PurchaseSales")
	res := resp.Interface().(*modify.PurchaseReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "PurchaseSales")

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{Code: httpserver.ErrInner, Msg: err.Error()}
	}
	js.Set("data", res.Info)
	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{Code: httpserver.ErrInner, Msg: "marshal json failed"}
	}

	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func modAddress(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")
	aid := req.GetParamInt("aid")
	province := req.GetParamInt("province")
	city := req.GetParamInt("city")
	zone := req.GetParamInt("zone")
	if province >= maxZipcode || city >= maxZipcode || zone >= maxZipcode {
		return &util.AppError{Code: httpserver.ErrInvalidParam, Msg: "illegal zipcode"}
	}
	zip := req.GetParamInt("zip")
	detail := req.GetParamString("detail")
	mobile := req.GetParamString("mobile")
	user := req.GetParamString("user")
	addr := req.GetParamString("addr")
	def := req.GetParamBoolDef("def", false)

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ModifyServerType, uid, "ModAddress",
		&modify.AddressRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Info: &common.AddressInfo{Aid: aid, Province: province,
				City: city, Zone: zone, Zip: zip, Addr: addr,
				Detail: detail, Def: def, User: user, Mobile: mobile}})
	httpserver.CheckRPCErr(rpcerr, "ModAddress")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "ModAddress")

	w.Write([]byte(`{"errno":0}`))
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func delAddress(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")
	aid := req.GetParamInt("aid")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ModifyServerType, uid, "DelAddress",
		&modify.AddressRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Info: &common.AddressInfo{Aid: aid}})
	httpserver.CheckRPCErr(rpcerr, "DelAddress")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "DelAddress")

	w.Write([]byte(`{"errno":0}`))
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func applyImageUpload(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")
	format := req.GetParamString("format")

	fname := util.GenUUID() + "." + format
	var names = []string{fname}
	err := httpserver.AddImages(uid, names)
	if err != nil {
		return &util.AppError{Code: httpserver.ErrInner, Msg: err.Error()}
	}

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{Code: httpserver.ErrInner, Msg: err.Error()}
	}
	data, err := simplejson.NewJson([]byte(`{}`))
	if err != nil {
		return &util.AppError{Code: httpserver.ErrInner, Msg: err.Error()}
	}
	aliyun.FillCallbackInfo(data)
	data.Set("name", fname)
	js.Set("data", data)

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{Code: httpserver.ErrInner,
			Msg: "marshal json failed"}
	}
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func pingppPay(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")
	amount := req.GetParamInt("amount")
	channel := req.GetParamString("channel")
	log.Printf("pingppPay uid:%d amount:%d channel:%s", uid, amount, channel)

	res := pay.GetPingPPCharge(int(amount), channel)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(res))
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func reportApmac(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")
	apmac := req.GetParamString("apmac")
	log.Printf("report_apmac uid:%d apmac:%s\n", uid, apmac)

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ModifyServerType, uid, "ReportApmac",
		&modify.ApmacRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Apmac: apmac})
	httpserver.CheckRPCErr(rpcerr, "ReportApmac")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "ReportApmac")

	w.Write([]byte(`{"errno":0}`))
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func uploadCallback(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	r.ParseForm()
	httpserver.ReportRequest(r.RequestURI)
	fname := r.Form["filename"]
	if len(fname) < 1 {
		log.Printf("parse filename failed\n")
		w.Write([]byte(`{"Status":"OK"}`))
		return nil
	}
	size := r.Form["size"]
	fsize, _ := strconv.ParseInt(size[0], 10, 64)
	height := r.Form["height"]
	fheight, _ := strconv.ParseInt(height[0], 10, 64)
	width := r.Form["width"]
	fwidth, _ := strconv.ParseInt(width[0], 10, 64)
	log.Printf("upload_callback fname:%s size:%d height:%d width:%d\n", fname, fsize,
		fheight, fwidth)

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ModifyServerType, 0, "FinImage",
		&modify.ImageRequest{Head: &common.Head{Sid: uuid},
			Info: &modify.ImageInfo{Name: fname[0], Size: fsize,
				Height: fheight, Width: fwidth}})
	httpserver.CheckRPCErr(rpcerr, "FinImage")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "FinImage")

	w.Write([]byte(`{"Status":"OK"}`))
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func reportClick(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")
	id := req.GetParamIntDef("id", 0)
	ctype := req.GetParamInt("type")
	name := req.GetParamStringDef("name", "")
	log.Printf("reportClick uid:%d type:%d id:%d name:%s", uid, ctype, id, name)

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ModifyServerType, uid, "ReportClick",
		&modify.ClickRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Id: id, Type: ctype, Name: name})
	httpserver.CheckRPCErr(rpcerr, "ReportClick")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "ReportClick")

	w.Write([]byte(`{"errno":0}`))
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func reportAdClick(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")
	id := req.GetParamInt("id")
	usermac := req.GetParamString("wlanusermac")
	userip := req.GetParamString("wlanuserip")
	apmac := req.GetParamString("wlanapmac")
	log.Printf("reportAdClick uid:%d id:%d ", uid, id)

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.AdvertiseServerType, uid, "ClickAd",
		&advertise.AdRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Aid: id, Usermac: usermac, Userip: userip, Apmac: apmac})
	httpserver.CheckRPCErr(rpcerr, "ClickAd")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "ClickAd")

	w.Write([]byte(`{"errno":0}`))
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func fetchWifi(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")
	longitude := req.GetParamFloat("longitude")
	latitude := req.GetParamFloat("latitude")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.FetchServerType, uid, "FetchWifi",
		&fetch.WifiRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Longitude: longitude, Latitude: latitude})
	httpserver.CheckRPCErr(rpcerr, "FetchWifi")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "FetchWifi")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func checkUpdate(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")
	term := req.GetParamInt("term")
	version := req.GetParamInt("version")
	channel := req.GetParamString("channel")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.FetchServerType, uid, "FetchLatestVersion",
		&fetch.VersionRequest{
			Head:    &common.Head{Sid: uuid, Uid: uid, Term: term, Version: version},
			Channel: channel})
	httpserver.CheckRPCErr(rpcerr, "FetchLatestVersion")
	res := resp.Interface().(*fetch.VersionReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "FetchLatestVersion")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func checkLogin(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.Init(r)
	usermac := req.GetParamString("wlanusermac")
	acname := req.GetParamString("wlanacname")
	log.Printf("checkLogin usermac:%s acname:%s", usermac, acname)

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPCCallback(util.VerifyServerType,
		0, "CheckLogin", req.Callback,
		&verify.AccessRequest{
			Head: &common.Head{Sid: uuid},
			Info: &verify.PortalInfo{Usermac: usermac, Acname: acname}})
	httpserver.CheckRPCErrCallback(rpcerr, "FetchLatestVersion", req.Callback)
	res := resp.Interface().(*verify.CheckReply)
	httpserver.CheckRPCCodeCallback(res.Head.Retcode, "FetchLatestVersion",
		req.Callback)

	body := httpserver.GenResponseBodyCallback(res, req.Callback, false)
	req.WriteRsp(w, body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func getFrontInfo(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.HotServerType, uid, "GetFrontInfo",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid}})
	httpserver.CheckRPCErr(rpcerr, "GetFrontInfo")
	res := resp.Interface().(*hot.FrontReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "GetFrontInfo")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func getFlashAd(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")
	version := req.GetParamInt("version")
	term := req.GetParamInt("term")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.FetchServerType, uid, "FetchFlashAd",
		&fetch.AdRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Term: term, Version: version})
	httpserver.CheckRPCErr(rpcerr, "GetFlashAd")
	res := resp.Interface().(*fetch.AdReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "GetFlashAd")

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{Code: httpserver.ErrInner,
			Msg: "invalid param"}
	}
	if res.Info != nil && res.Info.Img != "" {
		js.Set("data", res.Info)
	}

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{Code: httpserver.ErrInner,
			Msg: "marshal json failed"}
	}
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func getOpening(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.HotServerType, uid, "GetOpening",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid}})
	httpserver.CheckRPCErr(rpcerr, "GetOpening")
	res := resp.Interface().(*hot.OpeningReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "GetOpening")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func getOpened(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")
	seq := req.GetParamInt("seq")
	num := req.GetParamInt("num")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.HotServerType, uid, "GetOpened",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Seq: seq, Num: num})
	httpserver.CheckRPCErr(rpcerr, "GetOpened")
	res := resp.Interface().(*hot.OpenedReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "GetOpened")

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{Code: httpserver.ErrInner,
			Msg: "invalid param"}
	}
	js.SetPath([]string{"data", "opened"}, res.Opened)
	if len(res.Opened) >= int(num) {
		js.SetPath([]string{"data", "hasmore"}, 1)
	}

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{Code: httpserver.ErrInner,
			Msg: "marshal json failed"}
	}
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func getRunning(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")
	seq := req.GetParamInt("seq")
	num := req.GetParamInt("num")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.HotServerType, uid, "GetRunning",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Seq: seq, Num: num})
	httpserver.CheckRPCErr(rpcerr, "GetRunning")
	res := resp.Interface().(*hot.RunningReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "GetRunning")

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{Code: httpserver.ErrInner,
			Msg: "invalid param"}
	}
	js.SetPath([]string{"data", "running"}, res.Running)
	if len(res.Running) >= int(num) {
		js.SetPath([]string{"data", "hasmore"}, 1)
	}

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{Code: httpserver.ErrInner, Msg: "marshal json failed"}
	}
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func getMarquee(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.HotServerType, uid, "GetMarquee",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid}})
	httpserver.CheckRPCErr(rpcerr, "GetMarquee")
	res := resp.Interface().(*hot.MarqueeReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "GetMarquee")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func getHotList(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.HotServerType, uid, "GetHotList",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid}})
	httpserver.CheckRPCErr(rpcerr, "GetHotList")
	res := resp.Interface().(*hot.HotListReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "GetHotList")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func getWifiPass(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")
	longitude := req.GetParamFloat("longitude")
	latitude := req.GetParamFloat("latitude")
	ssids, err := req.Post.Get("data").Get("ssids").Array()
	if err != nil {
		return &util.AppError{Code: httpserver.ErrInner, Msg: err.Error()}
	}
	var ids []string
	if len(ssids) == 0 {
		return &util.AppError{Code: httpserver.ErrInvalidParam,
			Msg: "illegal param:empty ssids"}
	}
	for i := 0; i < len(ssids); i++ {
		ssid := ssids[i].(string)
		ids = append(ids, ssid)
	}

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.FetchServerType, uid, "FetchWifiPass",
		&fetch.WifiPassRequest{
			Head:      &common.Head{Sid: uuid, Uid: uid},
			Longitude: longitude,
			Latitude:  latitude,
			Ssids:     ids})
	httpserver.CheckRPCErr(rpcerr, "FetchWifiPass")
	res := resp.Interface().(*fetch.WifiPassReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "FetchWifiPass")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func getShare(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")
	gid := req.GetParamIntDef("gid", 0)
	seq := req.GetParamInt("seq")
	num := req.GetParamIntDef("num", util.MaxListSize)
	path := r.URL.Path
	log.Printf("path:%s", path)
	var stype int64
	if path == "/get_share_gid" {
		stype = util.GidShareType
	} else if path == "/get_share_list" {
		stype = util.ListShareType
	} else if path == "/get_share_uid" {
		stype = util.UidShareType
	}

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.FetchServerType, uid, "FetchShare",
		&fetch.ShareRequest{
			Head: &common.Head{Sid: uuid, Uid: uid},
			Type: stype, Seq: seq, Num: num, Id: gid})
	httpserver.CheckRPCErr(rpcerr, "FetchShare")
	res := resp.Interface().(*fetch.ShareReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "FetchShare")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func getShareDetail(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")
	sid := req.GetParamInt("sid")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.FetchServerType, uid, "FetchShareDetail",
		&common.CommRequest{
			Head: &common.Head{Sid: uuid, Uid: uid},
			Id:   sid})
	httpserver.CheckRPCErr(rpcerr, "FetchShareDetail")
	res := resp.Interface().(*fetch.ShareDetailReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "FetchShareDetail")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func getDetail(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")
	gid := req.GetParamIntDef("gid", 0)
	bid := req.GetParamIntDef("bid", 0)
	if gid == 0 && bid == 0 {
		return &util.AppError{Code: httpserver.ErrInvalidParam,
			Msg: "invalid param"}
	}

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.HotServerType, uid, "GetDetail",
		&hot.DetailRequest{
			Head: &common.Head{Sid: uuid, Uid: uid},
			Bid:  bid, Gid: gid})
	httpserver.CheckRPCErr(rpcerr, "GetDetail")
	res := resp.Interface().(*hot.DetailReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "GetDetail")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func getImageToken(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.FetchServerType, uid, "FetchStsCredentials",
		&common.CommRequest{
			Head: &common.Head{Sid: uuid, Uid: uid}})
	httpserver.CheckRPCErr(rpcerr, "FetchStsCredentials")
	res := resp.Interface().(*fetch.StsReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "FetchStsCredentials")

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{Code: httpserver.ErrInner, Msg: "invalid param"}
	}
	js.Set("data", res.Credential)

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{Code: httpserver.ErrInner, Msg: "marshal json failed"}
	}
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func getWeatherNews(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")
	response, err := getRspFromSSDB(hotWeatherKey)
	if err == nil {
		log.Printf("getRspFromSSDB succ key:%s\n", hotWeatherKey)
		httpserver.RspGzip(w, []byte(response))
		httpserver.ReportSuccResp(r.RequestURI)
		return nil
	}

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.HotServerType, uid, "GetWeatherNews",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid}})
	httpserver.CheckRPCErr(rpcerr, "GetWeatherNews")
	res := resp.Interface().(*hot.WeatherNewsReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "GetWeatherNews")

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{Code: httpserver.ErrInner, Msg: "invalid param"}
	}
	js.SetPath([]string{"data", "news"}, res.News)
	js.SetPath([]string{"data", "weather"}, res.Weather)
	js.SetPath([]string{"data", "notice"}, res.Notice)

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{Code: httpserver.ErrInner,
			Msg: "marshal json failed"}
	}
	httpserver.RspGzip(w, body)
	data := js.Get("data")
	setSSDBCache(hotWeatherKey, data)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func getLiveInfo(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")
	seq := req.GetParamInt("seq")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.HotServerType, uid, "GetLive",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid}, Seq: seq})
	httpserver.CheckRPCErr(rpcerr, "GetLive")
	res := resp.Interface().(*hot.LiveReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "GetLive")

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{Code: httpserver.ErrInner,
			Msg: "invalid param"}
	}
	js.SetPath([]string{"data", "list"}, res.List)
	if len(res.List) >= util.MaxListSize {
		js.SetPath([]string{"data", "hasmore"}, 1)
	}

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{Code: httpserver.ErrInner,
			Msg: "marshal json failed"}
	}
	httpserver.RspGzip(w, body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func getJokes(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")
	seq := req.GetParamInt("seq")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.HotServerType, uid, "GetJoke",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid}, Seq: seq})
	httpserver.CheckRPCErr(rpcerr, "GetJoke")
	res := resp.Interface().(*hot.JokeReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "GetJoke")

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{Code: httpserver.ErrInner,
			Msg: "invalid param"}
	}
	js.SetPath([]string{"data", "infos"}, res.Infos)
	if len(res.Infos) >= util.MaxListSize {
		js.SetPath([]string{"data", "hasmore"}, 1)
	}

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{Code: httpserver.ErrInner,
			Msg: "marshal json failed"}
	}
	httpserver.RspGzip(w, body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func getZipcode(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")
	ziptype := req.GetParamInt("type")
	code := req.GetParamInt("code")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.FetchServerType, uid, "FetchZipcode",
		&fetch.ZipcodeRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Type: ziptype, Code: code})
	httpserver.CheckRPCErr(rpcerr, "FetchZipcode")
	res := resp.Interface().(*fetch.ZipcodeReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "FetchZipcode")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func getActivity(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.FetchServerType, uid, "FetchActivity",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid}})
	httpserver.CheckRPCErr(rpcerr, "FetchActivity")
	res := resp.Interface().(*fetch.ActivityReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "FetchActivity")

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{Code: httpserver.ErrInner,
			Msg: "invalid param"}
	}
	js.Set("data", res.Activity)

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{Code: httpserver.ErrInner,
			Msg: "marshal json failed"}
	}
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func getGoodsIntro(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")
	gid := req.GetParamInt("gid")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.FetchServerType, uid, "FetchGoodsIntro",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Id: gid})
	httpserver.CheckRPCErr(rpcerr, "FetchGoodsIntro")
	res := resp.Interface().(*fetch.GoodsIntroReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "FetchGoodsIntro")

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{Code: httpserver.ErrInner, Msg: "invalid param"}
	}
	js.Set("data", res.Info)

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{Code: httpserver.ErrInner,
			Msg: "marshal json failed"}
	}
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func getBetHistory(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")
	gid := req.GetParamInt("gid")
	seq := req.GetParamInt("seq")
	num := req.GetParamInt("num")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.FetchServerType, uid, "FetchBetHistory",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Seq: seq, Num: num, Id: gid})
	httpserver.CheckRPCErr(rpcerr, "FetchBetHistory")
	res := resp.Interface().(*fetch.BetHistoryReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "FetchBetHistory")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func getPurchaseRecord(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")
	bid := req.GetParamInt("bid")
	seq := req.GetParamInt("seq")
	num := req.GetParamInt("num")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.FetchServerType, uid, "FetchPurchaseRecord",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Seq: seq, Num: num, Id: bid})
	httpserver.CheckRPCErr(rpcerr, "FetchPurchaseRecord")
	res := resp.Interface().(*fetch.PurchaseRecordReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "FetchPurchaseRecord")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func getUserBet(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")
	seq := req.GetParamInt("seq")
	num := req.GetParamInt("num")
	path := r.URL.Path
	var stype int64
	if path == "/get_user_award" {
		stype = util.UserAwardType
	} else {
		stype = util.UserBetType
	}

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.FetchServerType, uid, "FetchUserBet",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Seq: seq, Num: num, Type: stype})
	httpserver.CheckRPCErr(rpcerr, "FetchUserBet")
	res := resp.Interface().(*fetch.UserBetReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "FetchUserBet")

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{Code: httpserver.ErrInner,
			Msg: "invalid param"}
	}
	js.SetPath([]string{"data", "infos"}, res.Infos)
	if len(res.Infos) >= util.MaxListSize {
		js.SetPath([]string{"data", "hasmore"}, 1)
	}

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{Code: httpserver.ErrInner,
			Msg: "marshal json failed"}
	}
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func getKvConf(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")
	key := req.GetParamString("key")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.FetchServerType, uid, "FetchKvConf",
		&fetch.KvRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Key: key})
	httpserver.CheckRPCErr(rpcerr, "FetchKvConf")
	res := resp.Interface().(*fetch.KvReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "FetchKvConf")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func getMenu(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")
	term := req.GetParamInt("term")
	version := req.GetParamInt("version")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.FetchServerType, uid, "FetchMenu",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid,
			Term: term, Version: version}})
	httpserver.CheckRPCErr(rpcerr, "FetchMenu")
	res := resp.Interface().(*fetch.MenuReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "FetchMenu")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func getAddress(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.FetchServerType, uid, "FetchAddress",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid}})
	httpserver.CheckRPCErr(rpcerr, "FetchAddress")
	res := resp.Interface().(*fetch.AddressReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "FetchAddress")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func getWinStatus(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")
	bid := req.GetParamInt("bid")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.FetchServerType, uid, "FetchWinStatus",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Id: bid})
	httpserver.CheckRPCErr(rpcerr, "FetchWinStatus")
	res := resp.Interface().(*fetch.WinStatusReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "FetchWinStatus")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func genSsdbKey(ctype int64, newFlag bool) string {
	switch ctype {
	default:
		if newFlag {
			return hotNewsKey
		}
		return hotNewsCompKey
	case hotVideoType:
		return hotVideoKey
	case hotDgType:
		return hotDgNewsKey
	case hotAmuseType:
		return hotAmuseKey
	case hotJokeType:
		return hotJokeKey
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

func getHospitalNews(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")
	ctype := req.GetParamInt("type")
	term := req.GetParamInt("term")
	version := req.GetParamInt("version")
	seq := req.GetParamInt("seq")
	log.Printf("uid:%d ctype:%d seq:%d term:%d version:%d\n", uid, ctype, seq, term, version)

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.HotServerType, uid, "GetHospitalNews",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid, Term: term, Version: version},
			Type: ctype, Seq: seq})
	httpserver.CheckRPCErr(rpcerr, "GetHots")
	res := resp.Interface().(*hot.HotsReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "GetHots")

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{Code: httpserver.ErrInner,
			Msg: "invalid param"}
	}
	js.SetPath([]string{"data", "infos"}, res.Infos)
	if len(res.Infos) >= util.MaxListSize ||
		(seq == 0 && ctype == 0 && len(res.Infos) >= util.MaxListSize/2) {
		js.SetPath([]string{"data", "hasmore"}, 1)
	}

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{Code: httpserver.ErrInner,
			Msg: "marshal json failed"}
	}
	httpserver.RspGzip(w, body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func getHot(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")
	ctype := req.GetParamInt("type")
	term := req.GetParamInt("term")
	version := req.GetParamInt("version")
	seq := req.GetParamInt("seq")
	adtype := req.GetParamIntDef("adtype", 0)
	log.Printf("uid:%d ctype:%d seq:%d term:%d version:%d\n", uid, ctype, seq, term, version)
	if seq == 0 && adtype == 0 {
		flag := util.CheckTermVersion(term, version)
		key := genSsdbKey(ctype, flag)
		log.Printf("key:%s", key)
		resp, err := getRspFromSSDB(key)
		if err == nil {
			log.Printf("getRspFromSSDB succ key:%s\n", key)
			httpserver.RspGzip(w, []byte(resp))
			httpserver.ReportSuccResp(r.RequestURI)
			return nil
		}
		log.Printf("getRspFromSSDB failed key:%s err:%v\n", key, err)
	}

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.HotServerType, uid, "GetHots",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid, Term: term, Version: version},
			Type: ctype, Seq: seq, Subtype: adtype})
	httpserver.CheckRPCErr(rpcerr, "GetHots")
	res := resp.Interface().(*hot.HotsReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "GetHots")

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{Code: httpserver.ErrInner,
			Msg: "invalid param"}
	}
	js.SetPath([]string{"data", "infos"}, res.Infos)
	js.SetPath([]string{"data", "top"}, res.Top)
	if len(res.Infos) >= util.MaxListSize ||
		(seq == 0 && ctype == 0 && len(res.Infos) >= util.MaxListSize/2) {
		js.SetPath([]string{"data", "hasmore"}, 1)
	}

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{Code: httpserver.ErrInner,
			Msg: "marshal json failed"}
	}
	httpserver.RspGzip(w, body)
	if seq == 0 && adtype == 0 {
		flag := util.CheckTermVersion(term, version)
		key := genSsdbKey(ctype, flag)
		data := js.Get("data")
		setSSDBCache(key, data)
	}
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func autoLogin(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.Init(r)
	uid := req.GetParamInt("uid")
	token := req.GetParamString("token")
	privdata := req.GetParamString("privdata")
	log.Printf("autoLogin uid:%d token:%s privdata:%s", uid, token, privdata)

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.VerifyServerType, uid, "AutoLogin",
		&verify.AutoRequest{Head: &common.Head{Uid: uid, Sid: uuid},
			Token: token, Privdata: privdata})
	httpserver.CheckRPCErr(rpcerr, "AutoLogin")
	res := resp.Interface().(*verify.RegisterReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "GetHots")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func portalLogin(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.Init(r)
	phone := req.GetParamString("phone")
	code := req.GetParamString("code")
	acname := req.GetParamString("wlanacname")
	acip := req.GetParamString("wlanacip")
	userip := req.GetParamString("wlanuserip")
	usermac := req.GetParamString("wlanusermac")
	apmac := req.GetParamStringDef("wlanapmac", "")
	apmac = strings.Replace(strings.ToLower(apmac), ":", "", -1)
	log.Printf("portalLogin phone:%s code:%s acname:%s acip:%s userip:%s usermac:%s apmac:%s",
		phone, code, acname, acip, userip, usermac, apmac)

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPCCallback(util.VerifyServerType,
		0, "PortalLogin", req.Callback,
		&verify.PortalLoginRequest{Head: &common.Head{Sid: uuid},
			Info: &verify.PortalInfo{
				Acname: acname, Acip: acip, Usermac: usermac, Userip: userip,
				Phone: phone, Code: code, Apmac: apmac}})
	httpserver.CheckRPCErrCallback(rpcerr, "PortalLogin", req.Callback)
	res := resp.Interface().(*verify.PortalLoginReply)
	httpserver.CheckRPCCodeCallback(res.Head.Retcode, "PortalLogin", req.Callback)

	body := httpserver.GenResponseBodyCallback(res, req.Callback, true)
	req.WriteRsp(w, body)
	httpserver.ReportSuccResp(r.RequestURI)
	log.Printf("portalLogin succ phone:%s code:%s acname:%s acip:%s userip:%s usermac:%s res:%v",
		phone, code, acname, acip, userip, usermac, res)
	return nil
}

func oneClickLogin(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.Init(r)
	acname := req.GetParamString("wlanacname")
	acip := req.GetParamString("wlanacip")
	userip := req.GetParamString("wlanuserip")
	usermac := req.GetParamString("wlanusermac")
	apmac := req.GetParamStringDef("wlanapmac", "")
	apmac = strings.Replace(strings.ToLower(apmac), ":", "", -1)
	log.Printf("oneClickLogin acname:%s acip:%s userip:%s usermac:%s apmac:%s",
		acname, acip, userip, usermac, apmac)

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPCCallback(util.VerifyServerType,
		0, "OneClickLogin", req.Callback,
		&verify.AccessRequest{Head: &common.Head{Sid: uuid},
			Info: &verify.PortalInfo{
				Acname: acname, Acip: acip, Usermac: usermac,
				Userip: userip, Apmac: apmac}})
	httpserver.CheckRPCErrCallback(rpcerr, "OneClickLogin", req.Callback)
	res := resp.Interface().(*verify.PortalLoginReply)
	httpserver.CheckRPCCodeCallback(res.Head.Retcode, "OneClickLogin",
		req.Callback)

	body := httpserver.GenResponseBodyCallback(res, req.Callback, true)
	req.WriteRsp(w, body)
	httpserver.ReportSuccResp(r.RequestURI)
	log.Printf("oneClickLogin succ acname:%s acip:%s userip:%s usermac:%s res:%v",
		acname, acip, userip, usermac, res)
	return nil
}

func getService(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")
	term := req.GetParamInt("term")
	if term != util.WxTerm {
		response, err := getRspFromSSDB(hotServiceKey)
		if err == nil {
			log.Printf("getRspFromSSDB succ key:%s\n", hotServiceKey)
			httpserver.RspGzip(w, []byte(response))
			httpserver.ReportSuccResp(r.RequestURI)
			return nil
		}
	}

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.HotServerType, uid, "GetServices",
		&common.CommRequest{Head: &common.Head{Uid: uid, Sid: uuid, Term: term}})
	httpserver.CheckRPCErr(rpcerr, "GetServices")
	res := resp.Interface().(*hot.ServiceReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "GetServices")

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{Code: httpserver.ErrInner,
			Msg: "init json failed"}
	}
	js.SetPath([]string{"data", "services"}, res.Services)
	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{Code: httpserver.ErrInner,
			Msg: "marshal json failed"}
	}
	httpserver.RspGzip(w, body)
	if term != util.WxTerm {
		data := js.Get("data")
		setSSDBCache(hotServiceKey, data)
	}
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func getDiscovery(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")
	term := req.GetParamIntDef("term", 0)

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ConfigServerType, uid, "GetDiscovery",
		&common.CommRequest{Head: &common.Head{Uid: uid, Sid: uuid, Term: term}})
	httpserver.CheckRPCErr(rpcerr, "GetDiscovery")
	res := resp.Interface().(*config.DiscoveryReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "GetDiscovery")

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{Code: httpserver.ErrInner,
			Msg: "init json failed"}
	}
	js.SetPath([]string{"data", "services"}, res.Services)
	js.SetPath([]string{"data", "banners"}, res.Banners)
	js.SetPath([]string{"data", "recommends"}, res.Recommends)
	js.SetPath([]string{"data", "urbanservices"}, res.Urbanservices)
	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{Code: httpserver.ErrInner,
			Msg: "marshal json failed"}
	}
	httpserver.RspGzip(w, body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func punchAp(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")
	apmac := req.GetParamString("apmac")
	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.PunchServerType, uid, "Punch",
		&punch.PunchRequest{Head: &common.Head{Uid: uid, Sid: uuid}, Apmac: apmac})
	httpserver.CheckRPCErr(rpcerr, "Punch")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "Punch")

	w.Write([]byte(`{"errno":0}`))
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func correctAp(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")
	aid := req.GetParamInt("aid")
	etype := req.GetParamInt("type")
	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.PunchServerType, uid, "Correct",
		&punch.ApRequest{Head: &common.Head{Uid: uid, Sid: uuid}, Aid: aid,
			Etype: etype})
	httpserver.CheckRPCErr(rpcerr, "Correct")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "Correct")

	w.Write([]byte(`{"errno":0}`))
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func getMyPunch(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.PunchServerType, uid, "GetPunch",
		&common.CommRequest{
			Head: &common.Head{Sid: uuid, Uid: uid}})
	httpserver.CheckRPCErr(rpcerr, "GetPunch")
	res := resp.Interface().(*punch.PunchReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "GetPunch")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func getUserinfo(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.FetchServerType, uid, "FetchUserInfo",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid}})
	httpserver.CheckRPCErr(rpcerr, "FetchUserInfo")
	res := resp.Interface().(*fetch.UserInfoReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "FetchUserInfo")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func getUserInfo(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")
	tuid := req.GetParamInt("tuid")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.UserinfoServerType, uid, "GetInfo",
		&common.CommRequest{
			Head: &common.Head{Sid: uuid, Uid: tuid}})
	httpserver.CheckRPCErr(rpcerr, "GetInfo")
	res := resp.Interface().(*userinfo.InfoReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "GetInfo")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func getRandNick(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.UserinfoServerType, uid, "GenRandNick",
		&common.CommRequest{
			Head: &common.Head{Sid: uuid, Uid: uid}})
	httpserver.CheckRPCErr(rpcerr, "GenRandNick")
	res := resp.Interface().(*userinfo.NickReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "GenRandNick")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func getDefHead(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.UserinfoServerType, uid, "GetDefHead",
		&common.CommRequest{
			Head: &common.Head{Sid: uuid, Uid: uid}})
	httpserver.CheckRPCErr(rpcerr, "GetDefHead")
	res := resp.Interface().(*userinfo.HeadReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "GetDefHead")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func getPortalMenu(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ConfigServerType, uid, "GetPortalMenu",
		&common.CommRequest{
			Head: &common.Head{Sid: uuid, Uid: uid}})
	httpserver.CheckRPCErr(rpcerr, "GetPortalMenu")
	res := resp.Interface().(*config.PortalMenuReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "GetPortalMenu")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func getPortalConf(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")
	portaltype := req.GetParamIntDef("portaltype", 0)
	adtype := req.GetParamIntDef("adtype", 0)

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ConfigServerType, uid, "GetPortalConf",
		&common.CommRequest{
			Head: &common.Head{Sid: uuid, Uid: uid}, Type: portaltype,
			Subtype: adtype})
	httpserver.CheckRPCErr(rpcerr, "GetPortalConf")
	res := resp.Interface().(*config.PortalConfReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "GetPortalConf")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func getEducationVideo(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ConfigServerType, uid, "GetEducationVideo",
		&common.CommRequest{
			Head: &common.Head{Sid: uuid, Uid: uid},
		})
	httpserver.CheckRPCErr(rpcerr, "GetEducationVideo")
	res := resp.Interface().(*config.EducationVideoReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "GetEducationVideo")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func getHospitalDepartment(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")
	hid := req.GetParamInt("hid")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ConfigServerType, uid, "GetHospitalDepartment",
		&common.CommRequest{
			Head: &common.Head{Sid: uuid, Uid: uid}, Type: hid,
		})
	httpserver.CheckRPCErr(rpcerr, "GetHospitalDepartment")
	res := resp.Interface().(*config.HospitalDepartmentReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "GetHospitalDepartment")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func modUserInfo(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")
	nickname := req.GetParamStringDef("nickname", "")
	headurl := req.GetParamStringDef("headurl", "")

	if headurl == "" && nickname == "" {
		w.Write([]byte(`{"errno":0}`))
		return nil
	}

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.UserinfoServerType, uid, "ModInfo",
		&userinfo.InfoRequest{
			Head:    &common.Head{Sid: uuid, Uid: uid},
			Headurl: headurl, Nickname: nickname})
	httpserver.CheckRPCErr(rpcerr, "ModInfo")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "ModInfo")

	w.Write([]byte(`{"errno":0}`))
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func getPunchStat(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")
	apmac := req.GetParamString("apmac")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.PunchServerType, uid, "GetStat",
		&punch.PunchRequest{
			Head: &common.Head{Sid: uuid, Uid: uid}, Apmac: apmac})
	httpserver.CheckRPCErr(rpcerr, "GetStat")
	res := resp.Interface().(*punch.PunchStatReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "GetStat")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func reportIssue(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.Init(r)
	acname := req.GetParamString("wlanacname")
	apmac := req.GetParamString("wlanapmac")
	usermac := req.GetParamString("wlanusermac")
	content := req.GetParamString("content")
	contact := req.GetParamString("contact")
	ids := req.GetParamString("ids")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ModifyServerType, 0, "ReportIssue",
		&modify.IssueRequest{
			Head: &common.Head{Sid: uuid}, Acname: acname,
			Apmac: apmac, Usermac: usermac, Content: content,
			Contact: contact, Ids: ids})
	httpserver.CheckRPCErr(rpcerr, "ReportIssue")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "ReportIssue")

	body := httpserver.GenResponseBody(res, false)
	req.WriteRsp(w, body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func submitXcxCode(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.Init(r)
	code := req.GetParamString("code")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.PunchServerType, 0, "SubmitCode",
		&punch.CodeRequest{
			Head: &common.Head{Sid: uuid}, Code: code})
	httpserver.CheckRPCErr(rpcerr, "SubmitCode")
	res := resp.Interface().(*punch.LoginReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "SubmitCode")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func xcxLogin(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.Init(r)
	sid := req.GetParamString("sid")
	rawData := req.GetParamString("rawData")
	signature := req.GetParamString("signature")
	encryptedData := req.GetParamString("encryptedData")
	iv := req.GetParamString("iv")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.PunchServerType, 0, "Login",
		&punch.LoginRequest{
			Head: &common.Head{Sid: uuid}, Sid: sid,
			Rawdata: rawData, Signature: signature,
			Encrypteddata: encryptedData, Iv: iv})
	httpserver.CheckRPCErr(rpcerr, "Login")
	res := resp.Interface().(*punch.LoginReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "Login")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func getAllAps(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.Init(r)
	uid := req.GetParamInt("uid")
	response, err := getRspFromSSDB(hotAllApsKey)
	if err == nil {
		log.Printf("getRspFromSSDB succ key:%s\n", hotAllApsKey)
		httpserver.RspGzip(w, []byte(response))
		httpserver.ReportSuccResp(r.RequestURI)
		return nil
	}

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.FetchServerType, uid, "FetchAllAps",
		&common.CommRequest{
			Head: &common.Head{Sid: uuid, Uid: uid}})
	httpserver.CheckRPCErr(rpcerr, "FetchAllAps")
	res := resp.Interface().(*fetch.ApReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "FetchAllAps")

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{Code: httpserver.ErrInner,
			Msg: "invalid param"}
	}
	js.SetPath([]string{"data", "infos"}, res.Infos)

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{Code: httpserver.ErrInner,
			Msg: "marshal json failed"}
	}
	httpserver.RspGzip(w, body)
	data := js.Get("data")
	setSSDBCache(hotAllApsKey, data)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func getAppAps(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	return httpserver.GetAps(w, r, false)
}

func extractIP(addr string) string {
	arr := strings.Split(addr, ":")
	return arr[0]
}

func register(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.Init(r)
	username := req.GetParamString("username")
	password := req.GetParamString("password")
	udid := req.GetParamString("udid")
	model := req.GetParamString("model")
	channel := req.GetParamString("channel")
	version := req.GetParamInt("version")
	term := req.GetParamInt("term")
	regip := extractIP(r.RemoteAddr)
	code := req.GetParamStringDef("code", "")
	log.Printf("register request username:%s password:%s udid:%s model:%s channel:%s version:%d term:%d",
		username, password, udid, model, channel, version, term)

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.VerifyServerType, 0, "Register",
		&verify.RegisterRequest{Head: &common.Head{Sid: uuid},
			Username: username, Password: password, Code: code,
			Client: &verify.ClientInfo{Udid: udid, Model: model,
				Channel: channel, Regip: regip,
				Version: version, Term: term}})
	httpserver.CheckRPCErr(rpcerr, "Register")
	res := resp.Interface().(*verify.RegisterReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "Register")

	body := httpserver.GenResponseBody(res, true)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func wxMpLogin(w http.ResponseWriter, r *http.Request) {
	httpserver.ReportRequest(r.RequestURI)
	r.ParseForm()
	code := r.Form["code"]
	if len(code) == 0 {
		log.Printf("get code failed\n")
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}
	echostr := r.Form["echostr"]

	address := httpserver.GetNameServer(0, util.VerifyServerName)
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
	res, err := c.WxMpLogin(context.Background(),
		&verify.LoginRequest{Head: &common.Head{Sid: uuid}, Code: code[0]})
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

	dst := fmt.Sprintf("%s?uid=%d&token=%s&union=%s&open=%s", echostr[0],
		res.Head.Uid, res.Token, res.Privdata, res.Openid)
	log.Printf("wxMpLogin dst:%s", dst)
	http.Redirect(w, r, dst, http.StatusMovedPermanently)
}

func jumpOnline(w http.ResponseWriter, r *http.Request) {
	httpserver.ReportRequest(r.RequestURI)
	r.ParseForm()
	file := r.Form["echofile"]
	var echostr string
	if len(file) > 0 {
		echostr = file[0]
		echostr = wxHost + echostr
	}
	redirect := wxHost + "wx_mp_login"
	redirect += "?echostr=" + echostr
	dst := util.GenRedirectURL(redirect)
	log.Printf("jumpOnline redirect:%s", dst)
	http.Redirect(w, r, dst, http.StatusMovedPermanently)
}

func checkSubscribe(w http.ResponseWriter, r *http.Request) {
	httpserver.ReportRequest(r.RequestURI)
	r.ParseForm()
	openids := r.Form["open"]
	var openid string
	if len(openids) > 0 {
		openid = openids[0]
	}

	dst := postLoginURL
	if openid == "" {
		http.Redirect(w, r, dst, http.StatusMovedPermanently)
	}
	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.VerifyServerType, 0, "CheckSubscribe",
		&verify.SubscribeRequest{Head: &common.Head{Sid: uuid}, Type: 0,
			Openid: openid})
	if rpcerr.Interface() != nil {
		http.Redirect(w, r, dst, http.StatusMovedPermanently)
	}
	res := resp.Interface().(*verify.CheckReply)
	if res.Head.Retcode != 0 {
		http.Redirect(w, r, dst, http.StatusMovedPermanently)
	}
	http.Redirect(w, r, res.Dst, http.StatusMovedPermanently)
}

func auth(w http.ResponseWriter, r *http.Request) {
	httpserver.ReportRequest(r.RequestURI)
	r.ParseForm()
	openids := r.Form["openId"]
	extends := r.Form["extend"]
	tids := r.Form["tid"]
	var openid, extend, tid string
	if len(openids) > 0 {
		openid = openids[0]
	}
	if len(extends) > 0 {
		extend = extends[0]
	}
	if len(tids) > 0 {
		tid = tids[0]
	}
	arr := strings.Split(extend, ",")
	if len(arr) != 5 {
		log.Printf("auth parse extend failed:%s", extend)
	} else {
		log.Printf("form:%v", r.Form)

		uuid := util.GenUUID()
		_, rpcerr := httpserver.CallRPC(util.VerifyServerType, 0, "RecordWxConn",
			&verify.WxConnRequest{Head: &common.Head{Sid: uuid},
				Openid: openid, Acname: arr[0], Userip: arr[1],
				Acip: arr[2], Usermac: arr[3], Apmac: arr[4], Tid: tid})
		if rpcerr.Interface() != nil {
			log.Printf("auth RecordWxConn failed:%v", rpcerr)
		}
	}
	w.Write([]byte("OK"))
}

func jump(w http.ResponseWriter, r *http.Request) {
	httpserver.ReportRequest(r.RequestURI)
	r.ParseForm()
	file := r.Form["echofile"]
	var echostr string
	if len(file) > 0 {
		echostr = file[0]
		echostr = wxHost + echostr
	}
	ck, err := r.Cookie("UNION")
	if err == nil {
		log.Printf("get cookie UNION succ:%s", ck.Value)
		address := httpserver.GetNameServer(0, util.VerifyServerName)
		conn, err := grpc.Dial(address, grpc.WithInsecure())
		if err != nil {
			log.Printf("did not connect: %v", err)
			w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
			return
		}
		defer conn.Close()
		c := verify.NewVerifyClient(conn)

		uuid := util.GenUUID()
		res, err := c.UnionLogin(context.Background(),
			&verify.LoginRequest{Head: &common.Head{Sid: uuid}, Unionid: ck.Value})
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
	redirect := wxHost + "wx_mp_login"
	redirect += "?echostr=" + echostr
	dst := util.GenRedirectURL(redirect)
	http.Redirect(w, r, dst, http.StatusMovedPermanently)
}

func getPortalDir(acname, apmac string) string {
	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ConfigServerType, 0, "GetPortalDir",
		&config.PortalDirRequest{Head: &common.Head{Sid: uuid},
			Type: util.LoginType, Acname: acname, Apmac: apmac})
	if rpcerr.Interface() != nil {
		return defLoginURL
	}
	res := resp.Interface().(*config.PortalDirReply)
	if res.Head.Retcode != 0 {
		return defLoginURL
	}
	return res.Dir
}

func portal(w http.ResponseWriter, r *http.Request) {
	httpserver.ReportRequest(r.RequestURI)
	r.ParseForm()
	var acname, usermac, apmac string
	names := r.Form["wlanacname"]
	macs := r.Form["wlanusermac"]
	aps := r.Form["wlanapmac"]
	if len(names) > 0 {
		acname = names[0]
	}
	if len(macs) > 0 {
		usermac = macs[0]
	}
	if len(aps) > 0 {
		apmac = aps[0]
	}
	log.Printf("acname:%s usermac:%s apmac:%s", acname, usermac, apmac)
	pos := strings.Index(r.RequestURI, "?")
	var postfix string
	if pos != -1 {
		postfix = r.RequestURI[pos:]
	}
	var dst string
	if acname == "AC_SSH_B_10" {
		dst = "http://192.168.100.4:8080/login201703171857/" + postfix
	} else if util.IsWjjAcname(acname) {
		dir := getPortalDir(acname, apmac)
		dst = dir + postfix
	} else if util.IsTestAcname(acname) {
		dir := getPortalDir(acname, apmac)
		dst = dir + postfix
	} else if util.IsSshAcname(acname) {
		dir := getPortalDir(acname, apmac)
		dst = dir + postfix
	} else {
		dir := getPortalDir(acname, apmac)
		dst = dir + postfix
	}

	dst += fmt.Sprintf("&ts=%d", time.Now().Unix())
	log.Printf("portal dst:%s", dst)
	w.Header().Set("Cache-Control", "no-cache")
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

func printHead(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.Init(r)
	log.Printf("printHead head:%v", r.Header)
	req.WriteRsp(w, []byte(`{"errno":0}`))
	return nil
}

func getJsapiSign(w http.ResponseWriter, r *http.Request) {
	httpserver.ReportRequest(r.RequestURI)
	address := httpserver.GetNameServer(0, util.VerifyServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Printf("did not connect: %v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}
	defer conn.Close()
	c := verify.NewVerifyClient(conn)

	uuid := util.GenUUID()
	res, err := c.GetWxTicket(context.Background(),
		&verify.TicketRequest{Head: &common.Head{Sid: uuid}})
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
	out := fmt.Sprintf("var wx_cfg={\"debug\":false, \"appId\":\"%s\",\"timestamp\":%d,\"nonceStr\":\"%s\",\"signature\":\"%s\",\"jsApiList\":[],\"jsapi_ticket\":\"%s\"};", util.WxDgAppid, ts, noncestr, sign, res.Ticket)
	w.Write([]byte(out))
	return
}

func pingppWebhook(w http.ResponseWriter, r *http.Request) {
	if strings.ToUpper(r.Method) == "POST" {
		buf := new(bytes.Buffer)
		buf.ReadFrom(r.Body)
		webhook, err := pingpp.ParseWebhooks(buf.Bytes())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "fail")
			return
		}
		fmt.Println(webhook.Type)
		if webhook.Type == "charge.succeeded" {
			//TODO for charge success
			w.WriteHeader(http.StatusOK)
		} else if webhook.Type == "refund.succeeded" {
			//TODO for refund success
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
	return
}

func getAppConf(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	return httpserver.GetConf(w, r, false)
}

//NewAppServer return app http handler
func NewAppServer() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/login", httpserver.AppHandler(login))
	mux.Handle("/get_phone_code", httpserver.AppHandler(getPhoneCode))
	mux.Handle("/get_check_code", httpserver.AppHandler(getCheckCode))
	mux.Handle("/register", httpserver.AppHandler(register))
	mux.Handle("/logout", httpserver.AppHandler(logout))
	mux.Handle("/hot", httpserver.AppHandler(getHot))
	mux.Handle("/get_hospital_news", httpserver.AppHandler(getHospitalNews))
	mux.Handle("/get_weather_news", httpserver.AppHandler(getWeatherNews))
	mux.Handle("/get_live_info", httpserver.AppHandler(getLiveInfo))
	mux.Handle("/get_jokes", httpserver.AppHandler(getJokes))
	mux.Handle("/get_conf", httpserver.AppHandler(getKvConf))
	mux.Handle("/get_menu", httpserver.AppHandler(getMenu))
	mux.Handle("/get_front_info", httpserver.AppHandler(getFrontInfo))
	mux.Handle("/get_flash_ad", httpserver.AppHandler(getFlashAd))
	mux.Handle("/get_opening", httpserver.AppHandler(getOpening))
	mux.Handle("/get_opened", httpserver.AppHandler(getOpened))
	mux.Handle("/get_hotlist", httpserver.AppHandler(getHotList))
	mux.Handle("/get_wifi_pass", httpserver.AppHandler(getWifiPass))
	mux.Handle("/get_zipcode", httpserver.AppHandler(getZipcode))
	mux.Handle("/get_activity", httpserver.AppHandler(getActivity))
	mux.Handle("/get_intro", httpserver.AppHandler(getGoodsIntro))
	mux.Handle("/get_bet_history", httpserver.AppHandler(getBetHistory))
	mux.Handle("/get_record", httpserver.AppHandler(getPurchaseRecord))
	mux.Handle("/get_user_bet", httpserver.AppHandler(getUserBet))
	mux.Handle("/get_user_award", httpserver.AppHandler(getUserBet))
	mux.Handle("/get_address", httpserver.AppHandler(getAddress))
	mux.Handle("/get_win_status", httpserver.AppHandler(getWinStatus))
	mux.Handle("/post_share", httpserver.AppHandler(addShare))
	mux.Handle("/set_win_status", httpserver.AppHandler(setWinStatus))
	mux.Handle("/get_share_gid", httpserver.AppHandler(getShare))
	mux.Handle("/get_share_list", httpserver.AppHandler(getShare))
	mux.Handle("/get_share_uid", httpserver.AppHandler(getShare))
	mux.Handle("/get_share_detail", httpserver.AppHandler(getShareDetail))
	mux.Handle("/get_detail", httpserver.AppHandler(getDetail))
	mux.Handle("/get_detail_gid", httpserver.AppHandler(getDetail))
	mux.Handle("/add_address", httpserver.AppHandler(addAddress))
	mux.Handle("/feedback", httpserver.AppHandler(addFeedback))
	mux.Handle("/delete_address", httpserver.AppHandler(delAddress))
	mux.Handle("/update_address", httpserver.AppHandler(modAddress))
	mux.Handle("/get_image_token", httpserver.AppHandler(getImageToken))
	mux.Handle("/fetch_wifi", httpserver.AppHandler(fetchWifi))
	mux.Handle("/check_update", httpserver.AppHandler(checkUpdate))
	mux.Handle("/check_login", httpserver.AppHandler(checkLogin))
	mux.Handle("/one_click_login", httpserver.AppHandler(oneClickLogin))
	mux.Handle("/auto_login", httpserver.AppHandler(autoLogin))
	mux.Handle("/portal_login", httpserver.AppHandler(portalLogin))
	mux.Handle("/get_nearby_aps", httpserver.AppHandler(getAppAps))
	mux.Handle("/get_all_aps", httpserver.AppHandler(getAllAps))
	mux.Handle("/report_wifi", httpserver.AppHandler(reportWifi))
	mux.Handle("/report_click", httpserver.AppHandler(reportClick))
	mux.Handle("/report_ad_click", httpserver.AppHandler(reportAdClick))
	mux.Handle("/report_apmac", httpserver.AppHandler(reportApmac))
	mux.Handle("/connect_wifi", httpserver.AppHandler(connectWifi))
	mux.Handle("/upload_callback", httpserver.AppHandler(uploadCallback))
	mux.Handle("/purchase_sales", httpserver.AppHandler(purchaseSales))
	mux.Handle("/apply_image_upload", httpserver.AppHandler(applyImageUpload))
	mux.Handle("/pingpp_pay", httpserver.AppHandler(pingppPay))
	mux.Handle("/services", httpserver.AppHandler(getService))
	mux.Handle("/get_discovery", httpserver.AppHandler(getDiscovery))
	mux.Handle("/punch", httpserver.AppHandler(punchAp))
	mux.Handle("/get_my_punch", httpserver.AppHandler(getMyPunch))
	mux.Handle("/get_user_info", httpserver.AppHandler(getUserInfo))
	mux.Handle("/get_rand_nick", httpserver.AppHandler(getRandNick))
	mux.Handle("/mod_user_info", httpserver.AppHandler(modUserInfo))
	mux.Handle("/get_def_head", httpserver.AppHandler(getDefHead))
	mux.Handle("/get_portal_menu", httpserver.AppHandler(getPortalMenu))
	mux.Handle("/get_portal_conf", httpserver.AppHandler(getPortalConf))
	mux.Handle("/get_education_video", httpserver.AppHandler(getEducationVideo))
	mux.Handle("/get_hospital_department", httpserver.AppHandler(getHospitalDepartment))
	mux.Handle("/get_userinfo", httpserver.AppHandler(getUserinfo))
	mux.Handle("/get_punch_stat", httpserver.AppHandler(getPunchStat))
	mux.Handle("/submit_xcx_code", httpserver.AppHandler(submitXcxCode))
	mux.Handle("/report_issue", httpserver.AppHandler(reportIssue))
	mux.Handle("/xcx_login", httpserver.AppHandler(xcxLogin))
	mux.Handle("/correct_ap", httpserver.AppHandler(correctAp))
	mux.HandleFunc("/jump", jump)
	mux.HandleFunc("/auth", auth)
	mux.HandleFunc("/jump_online", jumpOnline)
	mux.HandleFunc("/check_subscribe", checkSubscribe)
	mux.Handle("/test", httpserver.AppHandler(printHead))
	mux.HandleFunc("/portal", portal)
	mux.HandleFunc("/wx_mp_login", wxMpLogin)
	mux.HandleFunc("/get_jsapi_sign", getJsapiSign)
	mux.HandleFunc("/pingpp_webhook", pingppWebhook)
	mux.Handle("/", http.FileServer(http.Dir("/data/server/html")))
	return mux
}
