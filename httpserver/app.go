package httpserver

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

	aliyun "../aliyun"
	pay "../pay"
	common "../proto/common"
	fetch "../proto/fetch"
	hot "../proto/hot"
	modify "../proto/modify"
	verify "../proto/verify"
	util "../util"
	simplejson "github.com/bitly/go-simplejson"
	pingpp "github.com/pingplusplus/pingpp-go/pingpp"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const (
	wxHost     = "http://wx.yunxingzh.com/"
	maxZipcode = 820000
)

func login(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
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

	body, err := genResponseBody(res, true)
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
	var req request
	req.init(r.Body)
	phone := req.GetParamString("phone")
	ctype := req.GetParamInt("type")

	if !util.IsIllegalPhone(phone) {
		log.Printf("getPhoneCode illegal phone:%s", phone)
		return &util.AppError{util.LogicErr, 109, "请输入正确的手机号"}
	}

	flag, err := getCode(phone, int32(ctype))
	if err != nil || !flag {
		return &util.AppError{util.LogicErr, 103, "获取验证码失败"}
	}
	w.Write([]byte(`{"errno":0}`))
	return nil
}

func getCheckCode(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.init(r.Body)
	phone := req.GetParamString("phone")

	if !util.IsIllegalPhone(phone) {
		log.Printf("getCheckCode illegal phone:%s", phone)
		return &util.AppError{util.LogicErr, 109, "请输入正确的手机号"}
	}

	address := getNameServer(0, util.VerifyServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := verify.NewVerifyClient(conn)

	uuid := util.GenUUID()
	res, err := c.GetCheckCode(context.Background(),
		&verify.CodeRequest{Head: &common.Head{Sid: uuid},
			Phone: phone})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}

	if res.Head.Retcode != 0 {
		return &util.AppError{util.LogicErr, 4, "logout failed"}
	}

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func logout(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
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

func connectWifi(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckApp(r.Body)
	uid := req.GetParamInt("uid")
	acname := req.GetParamString("wlanacname")
	acip := req.GetParamString("wlanacip")
	userip := req.GetParamString("wlanuserip")
	usermac := req.GetParamString("wlanusermac")
	apmac := req.GetParamString("apmac")

	address := getNameServer(uid, util.ModifyServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := modify.NewModifyClient(conn)

	uuid := util.GenUUID()
	res, err := c.WifiAccess(context.Background(),
		&modify.AccessRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Info: &modify.AccessInfo{Userip: userip, Usermac: usermac, Acname: acname, Acip: acip,
				Apmac: apmac}})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}

	if res.Head.Retcode != 0 {
		return &util.AppError{util.LogicErr, 4, "WifiAccess failed"}
	}

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func addAddress(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckApp(r.Body)
	uid := req.GetParamInt("uid")
	province := req.GetParamInt("province")
	city := req.GetParamInt("city")
	zone := req.GetParamInt("zone")
	if province >= maxZipcode || city >= maxZipcode || zone >= maxZipcode {
		return &util.AppError{util.JSONErr, 2, "illegal zipcode"}
	}
	zip := req.GetParamInt("zip")
	detail := req.GetParamString("detail")
	mobile := req.GetParamString("mobile")
	user := req.GetParamString("user")
	addr := req.GetParamString("addr")
	def := req.GetParamBoolDef("def", false)

	address := getNameServer(uid, util.ModifyServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := modify.NewModifyClient(conn)

	uuid := util.GenUUID()
	res, err := c.AddAddress(context.Background(),
		&modify.AddressRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Info: &common.AddressInfo{Province: province, City: city, Zone: zone, Zip: zip,
				Addr: addr, Detail: detail, Def: def, User: user, Mobile: mobile}})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}

	if res.Head.Retcode != 0 {
		return &util.AppError{util.LogicErr, 4, "AddAddress failed"}
	}

	body, err := genResponseBody(res, false)
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}

	w.Write(body)
	return nil
}

func addShare(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckApp(r.Body)
	uid := req.GetParamInt("uid")
	bid := req.GetParamInt("bid")
	title := req.GetParamString("title")
	text := req.GetParamString("text")
	images, err := req.Post.Get("data").Get("images").Array()
	if err != nil {
		return &util.AppError{util.JSONErr, 2, err.Error()}
	}
	var imgs []string
	for i := 0; i < len(images); i++ {
		img := images[i].(string)
		imgs = append(imgs, img)
	}

	address := getNameServer(uid, util.ModifyServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := modify.NewModifyClient(conn)

	uuid := util.GenUUID()
	res, err := c.AddShare(context.Background(),
		&modify.ShareRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Bid: bid, Title: title, Text: text, Images: imgs})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}

	if res.Head.Retcode != 0 {
		return &util.AppError{util.LogicErr, 4, "AddShare failed"}
	}

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func setWinStatus(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckApp(r.Body)
	uid := req.GetParamInt("uid")
	bid := req.GetParamInt("bid")
	status := req.GetParamInt("status")
	aid := req.GetParamIntDef("aid", 0)
	account := req.GetParamStringDef("account", "")

	address := getNameServer(uid, util.ModifyServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := modify.NewModifyClient(conn)

	uuid := util.GenUUID()
	res, err := c.SetWinStatus(context.Background(),
		&modify.WinStatusRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Bid: bid, Status: status, Aid: aid, Account: account})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}

	if res.Head.Retcode != 0 {
		return &util.AppError{util.LogicErr, 4, "AddAddress failed"}
	}

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func addFeedback(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckApp(r.Body)
	uid := req.GetParamInt("uid")
	content := req.GetParamString("content")
	contact := req.GetParamStringDef("contact", "")

	address := getNameServer(uid, util.ModifyServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := modify.NewModifyClient(conn)

	uuid := util.GenUUID()
	res, err := c.AddFeedback(context.Background(),
		&modify.FeedRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Content: content, Contact: contact})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}

	if res.Head.Retcode != 0 {
		return &util.AppError{util.LogicErr, 4, "AddFeedback failed"}
	}

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func purchaseSales(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckApp(r.Body)
	uid := req.GetParamInt("uid")
	bid := req.GetParamInt("bid")

	address := getNameServer(uid, util.ModifyServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := modify.NewModifyClient(conn)

	uuid := util.GenUUID()
	res, err := c.PurchaseSales(context.Background(),
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Id: bid})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}

	if res.Head.Retcode != 0 {
		return &util.AppError{util.LogicErr, 4, "purchaseSales failed"}
	}

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{util.JSONErr, 4, err.Error()}
	}
	js.Set("data", res.Info)
	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}

	w.Write(body)
	return nil
}

func modAddress(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckApp(r.Body)
	uid := req.GetParamInt("uid")
	aid := req.GetParamInt("aid")
	province := req.GetParamInt("province")
	city := req.GetParamInt("city")
	zone := req.GetParamInt("zone")
	if province >= maxZipcode || city >= maxZipcode || zone >= maxZipcode {
		return &util.AppError{util.JSONErr, 2, "illegal zipcode"}
	}
	zip := req.GetParamInt("zip")
	detail := req.GetParamString("detail")
	mobile := req.GetParamString("mobile")
	user := req.GetParamString("user")
	addr := req.GetParamString("addr")
	def := req.GetParamBoolDef("def", false)

	address := getNameServer(uid, util.ModifyServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := modify.NewModifyClient(conn)

	uuid := util.GenUUID()
	res, err := c.ModAddress(context.Background(),
		&modify.AddressRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Info: &common.AddressInfo{Aid: aid, Province: province, City: city, Zone: zone,
				Zip: zip, Addr: addr, Detail: detail, Def: def, User: user, Mobile: mobile}})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}

	if res.Head.Retcode != 0 {
		return &util.AppError{util.LogicErr, 4, "ModAddress failed"}
	}

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func delAddress(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckApp(r.Body)
	uid := req.GetParamInt("uid")
	aid := req.GetParamInt("aid")

	address := getNameServer(uid, util.ModifyServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := modify.NewModifyClient(conn)

	uuid := util.GenUUID()
	res, err := c.DelAddress(context.Background(),
		&modify.AddressRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Info: &common.AddressInfo{Aid: aid}})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}

	if res.Head.Retcode != 0 {
		return &util.AppError{util.LogicErr, 4, "DelAddress failed"}
	}

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func applyImageUpload(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckApp(r.Body)
	uid := req.GetParamInt("uid")
	format := req.GetParamString("format")

	fname := util.GenUUID() + "." + format
	var names = []string{fname}
	err := addImages(uid, names)
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{util.JSONErr, 4, err.Error()}
	}
	data, err := simplejson.NewJson([]byte(`{}`))
	if err != nil {
		return &util.AppError{util.JSONErr, 4, err.Error()}
	}
	aliyun.FillCallbackInfo(data)
	data.Set("name", fname)
	js.Set("data", data)

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func pingppPay(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckApp(r.Body)
	uid := req.GetParamInt("uid")
	amount := req.GetParamInt("amount")
	channel := req.GetParamString("channel")
	log.Printf("pingppPay uid:%d amount:%d channel:%s", uid, amount, channel)

	res := pay.GetPingPPCharge(int(amount), channel)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(res))
	return nil
}

func reportApmac(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
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
	var req request
	req.initCheckApp(r.Body)
	uid := req.GetParamInt("uid")
	id := req.GetParamInt("id")
	ctype := req.GetParamInt("type")
	log.Printf("reportClick uid:%d type:%d id:%d", uid, ctype, id)

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

	body, err := genResponseBody(res, false)
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func getFrontInfo(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
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
	res, err := c.GetFrontInfo(context.Background(),
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid}})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "获取首页信息失败"}
	}

	body, err := genResponseBody(res, false)
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func getFlashAd(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckApp(r.Body)
	uid := req.GetParamInt("uid")
	version := req.GetParamInt("version")
	term := req.GetParamInt("term")

	address := getNameServer(uid, util.FetchServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := fetch.NewFetchClient(conn)

	uuid := util.GenUUID()
	res, err := c.FetchFlashAd(context.Background(),
		&fetch.AdRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Term: term, Version: version})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "获取闪屏广告失败"}
	}

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "invalid param"}
	}
	if res.Info != nil && res.Info.Img != "" {
		js.Set("data", res.Info)
	}

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func getOpening(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
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
	res, err := c.GetOpening(context.Background(),
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid}})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "获取已经揭晓失败"}
	}

	body, err := genResponseBody(res, false)
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func getOpened(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckApp(r.Body)
	uid := req.GetParamInt("uid")
	seq := req.GetParamInt("seq")
	num := req.GetParamInt("num")

	address := getNameServer(uid, util.HotServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := hot.NewHotClient(conn)

	uuid := util.GenUUID()
	res, err := c.GetOpened(context.Background(),
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Seq: seq, Num: int32(num)})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "获取即将揭晓失败"}
	}

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "invalid param"}
	}
	js.SetPath([]string{"data", "opened"}, res.Opened)
	if len(res.Opened) >= int(num) {
		js.SetPath([]string{"data", "hasmore"}, 1)
	}

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func getRunning(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckApp(r.Body)
	uid := req.GetParamInt("uid")
	seq := req.GetParamInt("seq")
	num := req.GetParamInt("num")

	address := getNameServer(uid, util.HotServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := hot.NewHotClient(conn)

	uuid := util.GenUUID()
	res, err := c.GetRunning(context.Background(),
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Seq: seq, Num: int32(num)})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "获取正在抢购数据失败"}
	}

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "invalid param"}
	}
	js.SetPath([]string{"data", "running"}, res.Running)
	if len(res.Running) >= int(num) {
		js.SetPath([]string{"data", "hasmore"}, 1)
	}

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func getMarquee(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
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
	res, err := c.GetMarquee(context.Background(),
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid}})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "获取跑马灯数据失败"}
	}

	body, err := genResponseBody(res, false)
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func getHotList(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
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
	res, err := c.GetHotList(context.Background(),
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid}})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "获取火热开抢失败"}
	}

	body, err := genResponseBody(res, false)
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func getWifiPass(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckApp(r.Body)
	uid := req.GetParamInt("uid")
	longitude := req.GetParamFloat("longitude")
	latitude := req.GetParamFloat("latitude")
	ssids, err := req.Post.Get("data").Get("ssids").Array()
	if err != nil {
		return &util.AppError{util.JSONErr, 2, err.Error()}
	}
	var ids []string
	for i := 0; i < len(ssids); i++ {
		ssid := ssids[i].(string)
		ids = append(ids, ssid)
	}

	address := getNameServer(uid, util.FetchServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := fetch.NewFetchClient(conn)

	uuid := util.GenUUID()
	res, err := c.FetchWifiPass(context.Background(),
		&fetch.WifiPassRequest{
			Head:      &common.Head{Sid: uuid, Uid: uid},
			Longitude: longitude,
			Latitude:  latitude,
			Ssids:     ids})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "获取Wifi密码失败"}
	}

	body, err := genResponseBody(res, false)
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func getShare(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckApp(r.Body)
	uid := req.GetParamInt("uid")
	gid := req.GetParamIntDef("gid", 0)
	seq := req.GetParamInt("seq")
	num := req.GetParamIntDef("num", util.MaxListSize)
	path := r.URL.Path
	log.Printf("path:%s", path)
	var stype int32
	if path == "/get_share_gid" {
		stype = util.GidShareType
	} else if path == "/get_share_list" {
		stype = util.ListShareType
	} else if path == "/get_share_uid" {
		stype = util.UidShareType
	}

	address := getNameServer(uid, util.FetchServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := fetch.NewFetchClient(conn)

	uuid := util.GenUUID()
	res, err := c.FetchShare(context.Background(),
		&fetch.ShareRequest{
			Head: &common.Head{Sid: uuid, Uid: uid},
			Type: stype, Seq: seq, Num: int32(num), Id: gid})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "获取晒单信息失败"}
	}

	body, err := genResponseBody(res, false)
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func getShareDetail(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckApp(r.Body)
	uid := req.GetParamInt("uid")
	sid := req.GetParamInt("sid")

	address := getNameServer(uid, util.FetchServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := fetch.NewFetchClient(conn)

	uuid := util.GenUUID()
	res, err := c.FetchShareDetail(context.Background(),
		&common.CommRequest{
			Head: &common.Head{Sid: uuid, Uid: uid},
			Id:   sid})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "获取晒单详情失败"}
	}

	body, err := genResponseBody(res, false)
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func getDetail(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckApp(r.Body)
	uid := req.GetParamInt("uid")
	gid := req.GetParamIntDef("gid", 0)
	bid := req.GetParamIntDef("bid", 0)
	if gid == 0 && bid == 0 {
		return &util.AppError{util.JSONErr, 2, "invalid param"}
	}

	address := getNameServer(uid, util.HotServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := hot.NewHotClient(conn)

	uuid := util.GenUUID()
	res, err := c.GetDetail(context.Background(),
		&hot.DetailRequest{
			Head: &common.Head{Sid: uuid, Uid: uid},
			Bid:  bid, Gid: gid})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "获取详情信息失败"}
	}

	body, err := genResponseBody(res, false)
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func getImageToken(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckApp(r.Body)
	uid := req.GetParamInt("uid")

	address := getNameServer(uid, util.FetchServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := fetch.NewFetchClient(conn)

	uuid := util.GenUUID()
	res, err := c.FetchStsCredentials(context.Background(),
		&common.CommRequest{
			Head: &common.Head{Sid: uuid, Uid: uid}})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "获取sts credentials失败"}
	}

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "invalid param"}
	}
	js.Set("data", res.Credential)

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func getWeatherNews(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
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
	res, err := c.GetWeatherNews(context.Background(),
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid}})
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
	js.SetPath([]string{"data", "news"}, res.News)
	js.SetPath([]string{"data", "weather"}, res.Weather)

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	rspGzip(w, body)
	data := js.Get("data")
	setSSDBCache(hotWeatherKey, data)
	return nil
}

func getZipcode(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckApp(r.Body)
	uid := req.GetParamInt("uid")
	ziptype := req.GetParamInt("type")
	code := req.GetParamInt("code")

	address := getNameServer(uid, util.FetchServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := fetch.NewFetchClient(conn)

	uuid := util.GenUUID()
	res, err := c.FetchZipcode(context.Background(),
		&fetch.ZipcodeRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Type: int32(ziptype), Code: int32(code)})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "获取邮政编码失败"}
	}

	body, err := genResponseBody(res, false)
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func getActivity(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckApp(r.Body)
	uid := req.GetParamInt("uid")

	address := getNameServer(uid, util.FetchServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := fetch.NewFetchClient(conn)

	uuid := util.GenUUID()
	res, err := c.FetchActivity(context.Background(),
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid}})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "获取活动页面失败"}
	}

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "invalid param"}
	}
	js.Set("data", res.Activity)

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func getGoodsIntro(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckApp(r.Body)
	uid := req.GetParamInt("uid")
	gid := req.GetParamInt("gid")

	address := getNameServer(uid, util.FetchServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := fetch.NewFetchClient(conn)

	uuid := util.GenUUID()
	res, err := c.FetchGoodsIntro(context.Background(),
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Id: gid})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "获取商品详情失败"}
	}

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "invalid param"}
	}
	js.Set("data", res.Info)

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func getBetHistory(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckApp(r.Body)
	uid := req.GetParamInt("uid")
	gid := req.GetParamInt("gid")
	seq := req.GetParamInt("seq")
	num := req.GetParamInt("num")

	address := getNameServer(uid, util.FetchServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := fetch.NewFetchClient(conn)

	uuid := util.GenUUID()
	res, err := c.FetchBetHistory(context.Background(),
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Seq: seq, Num: int32(num), Id: gid})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "获取往期记录失败"}
	}

	body, err := genResponseBody(res, false)
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func getPurchaseRecord(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckApp(r.Body)
	uid := req.GetParamInt("uid")
	bid := req.GetParamInt("bid")
	seq := req.GetParamInt("seq")
	num := req.GetParamInt("num")

	address := getNameServer(uid, util.FetchServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := fetch.NewFetchClient(conn)

	uuid := util.GenUUID()
	res, err := c.FetchPurchaseRecord(context.Background(),
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Seq: seq, Num: int32(num), Id: bid})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "获取抢购记录失败"}
	}

	body, err := genResponseBody(res, false)
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func getUserInfo(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckApp(r.Body)
	uid := req.GetParamInt("uid")

	address := getNameServer(uid, util.FetchServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := fetch.NewFetchClient(conn)

	uuid := util.GenUUID()
	res, err := c.FetchUserInfo(context.Background(),
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid}})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "获取用户信息失败"}
	}

	body, err := genResponseBody(res, false)
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func getUserBet(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckApp(r.Body)
	uid := req.GetParamInt("uid")
	seq := req.GetParamInt("seq")
	num := req.GetParamInt("num")
	path := r.URL.Path
	var stype int32
	if path == "/get_user_award" {
		stype = util.UserAwardType
	} else {
		stype = util.UserBetType
	}

	address := getNameServer(uid, util.FetchServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := fetch.NewFetchClient(conn)

	uuid := util.GenUUID()
	res, err := c.FetchUserBet(context.Background(),
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Seq: seq, Num: int32(num), Type: stype})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "获取用户信息失败"}
	}

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "invalid param"}
	}
	js.SetPath([]string{"data", "infos"}, res.Infos)
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

func getKvConf(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckApp(r.Body)
	uid := req.GetParamInt("uid")
	key := req.GetParamString("key")

	address := getNameServer(uid, util.FetchServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := fetch.NewFetchClient(conn)

	uuid := util.GenUUID()
	res, err := c.FetchKvConf(context.Background(),
		&fetch.KvRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Key: key})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "获取配置失败"}
	}

	body, err := genResponseBody(res, false)
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func getMenu(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckApp(r.Body)
	uid := req.GetParamInt("uid")

	address := getNameServer(uid, util.FetchServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := fetch.NewFetchClient(conn)

	uuid := util.GenUUID()
	res, err := c.FetchMenu(context.Background(),
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid}})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "获取菜单失败"}
	}

	body, err := genResponseBody(res, false)
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func getAddress(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckApp(r.Body)
	uid := req.GetParamInt("uid")

	address := getNameServer(uid, util.FetchServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := fetch.NewFetchClient(conn)

	uuid := util.GenUUID()
	res, err := c.FetchAddress(context.Background(),
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid}})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "获取用户地址失败"}
	}

	body, err := genResponseBody(res, false)
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func getWinStatus(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckApp(r.Body)
	uid := req.GetParamInt("uid")
	bid := req.GetParamInt("bid")

	address := getNameServer(uid, util.FetchServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := fetch.NewFetchClient(conn)

	uuid := util.GenUUID()
	res, err := c.FetchWinStatus(context.Background(),
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Id: bid})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "获取奖品状态失败"}
	}

	body, err := genResponseBody(res, false)
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
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

func getHot(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckApp(r.Body)
	uid := req.GetParamInt("uid")
	ctype := req.GetParamInt("type")
	term := req.GetParamInt("term")
	version := req.GetParamInt("version")
	seq := req.GetParamInt("seq")
	log.Printf("uid:%d ctype:%d seq:%d\n", uid, ctype, seq)
	if seq == 0 {
		flag := util.CheckTermVersion(term, version)
		key := genSsdbKey(ctype, flag)
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
	res, err := c.GetHots(context.Background(),
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid, Term: term, Version: version},
			Type: int32(ctype), Seq: seq})
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
	js.SetPath([]string{"data", "infos"}, res.Infos)
	if len(res.Infos) >= util.MaxListSize ||
		(seq == 0 && ctype == 0 && len(res.Infos) >= util.MaxListSize/2) {
		js.SetPath([]string{"data", "hasmore"}, 1)
	}

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	rspGzip(w, body)
	if seq == 0 {
		flag := util.CheckTermVersion(term, version)
		key := genSsdbKey(ctype, flag)
		data := js.Get("data")
		setSSDBCache(key, data)
	}
	return nil
}

func autoLogin(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.init(r.Body)
	uid := req.GetParamInt("uid")
	token := req.GetParamString("token")
	privdata := req.GetParamString("privdata")
	log.Printf("autoLogin uid:%d token:%s privdata:%s", uid, token, privdata)

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

	body, err := genResponseBody(res, false)
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func portalLogin(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.init(r.Body)
	phone := req.GetParamString("phone")
	code := req.GetParamString("code")
	acname := req.GetParamString("wlanacname")
	acip := req.GetParamString("wlanacip")
	userip := req.GetParamString("wlanuserip")
	usermac := req.GetParamString("wlanusermac")

	address := getNameServer(0, util.VerifyServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := verify.NewVerifyClient(conn)

	uuid := util.GenUUID()
	res, err := c.PortalLogin(context.Background(),
		&verify.PortalLoginRequest{Head: &common.Head{Sid: uuid},
			Info: &verify.PortalInfo{
				Acname: acname, Acip: acip, Usermac: usermac, Userip: userip,
				Phone: phone, Code: code},
		})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}

	if res.Head.Retcode == common.ErrCode_CHECK_CODE {
		return &util.AppError{util.LogicErr, errCode, "验证码错误"}
	} else if res.Head.Retcode == common.ErrCode_ZTE_LOGIN {
		return &util.AppError{util.DataErr, errZteLogin, "登录失败"}
	} else if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, errInner, "登录失败"}
	}

	body, err := genResponseBody(res, true)
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func getService(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckApp(r.Body)
	uid := req.GetParamInt("uid")
	resp, err := getRspFromSSDB(hotServiceKey)
	if err == nil {
		log.Printf("getRspFromSSDB succ key:%s\n", hotServiceKey)
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
	res, err := c.GetServices(context.Background(),
		&common.CommRequest{Head: &common.Head{Uid: uid, Sid: uuid}})
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
	rspGzip(w, body)
	data := js.Get("data")
	setSSDBCache(hotServiceKey, data)
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
	var req request
	req.init(r.Body)
	username := req.GetParamString("username")
	password := req.GetParamString("password")
	udid := req.GetParamString("udid")
	model := req.GetParamString("model")
	channel := req.GetParamString("channel")
	version := req.GetParamInt("version")
	term := req.GetParamInt("term")
	regip := extractIP(r.RemoteAddr)
	log.Printf("register request username:%s password:%s udid:%s model:%s channel:%s version:%d term:%d",
		username, password, udid, model, channel, version, term)

	address := getNameServer(0, util.VerifyServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := verify.NewVerifyClient(conn)

	uuid := util.GenUUID()
	res, err := c.Register(context.Background(),
		&verify.RegisterRequest{Head: &common.Head{Sid: uuid}, Username: username, Password: password,
			Client: &verify.ClientInfo{Udid: udid, Model: model, Channel: channel, Regip: regip,
				Version: int32(version), Term: int32(term)}})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}

	if res.Head.Retcode == common.ErrCode_USED_PHONE {
		return &util.AppError{util.LogicErr, 104, "该账号已注册，请直接登录"}
	} else if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "服务器又傲娇了"}
	}

	body, err := genResponseBody(res, true)
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
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
		echostr = wxHost + echostr
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
	redirect := wxHost + "wx_mp_login"
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

//NewAppServer return app http handler
func NewAppServer() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/login", appHandler(login))
	mux.Handle("/get_phone_code", appHandler(getPhoneCode))
	mux.Handle("/get_check_code", appHandler(getCheckCode))
	mux.Handle("/register", appHandler(register))
	mux.Handle("/logout", appHandler(logout))
	mux.Handle("/hot", appHandler(getHot))
	mux.Handle("/get_weather_news", appHandler(getWeatherNews))
	mux.Handle("/get_conf", appHandler(getKvConf))
	mux.Handle("/get_menu", appHandler(getMenu))
	mux.Handle("/get_front_info", appHandler(getFrontInfo))
	mux.Handle("/get_flash_ad", appHandler(getFlashAd))
	mux.Handle("/get_opening", appHandler(getOpening))
	mux.Handle("/get_opened", appHandler(getOpened))
	mux.Handle("/get_hotlist", appHandler(getHotList))
	mux.Handle("/get_wifi_pass", appHandler(getWifiPass))
	mux.Handle("/get_zipcode", appHandler(getZipcode))
	mux.Handle("/get_activity", appHandler(getActivity))
	mux.Handle("/get_intro", appHandler(getGoodsIntro))
	mux.Handle("/get_bet_history", appHandler(getBetHistory))
	mux.Handle("/get_record", appHandler(getPurchaseRecord))
	mux.Handle("/get_userinfo", appHandler(getUserInfo))
	mux.Handle("/get_user_bet", appHandler(getUserBet))
	mux.Handle("/get_user_award", appHandler(getUserBet))
	mux.Handle("/get_address", appHandler(getAddress))
	mux.Handle("/get_win_status", appHandler(getWinStatus))
	mux.Handle("/post_share", appHandler(addShare))
	mux.Handle("/set_win_status", appHandler(setWinStatus))
	mux.Handle("/get_share_gid", appHandler(getShare))
	mux.Handle("/get_share_list", appHandler(getShare))
	mux.Handle("/get_share_uid", appHandler(getShare))
	mux.Handle("/get_share_detail", appHandler(getShareDetail))
	mux.Handle("/get_detail", appHandler(getDetail))
	mux.Handle("/get_detail_gid", appHandler(getDetail))
	mux.Handle("/add_address", appHandler(addAddress))
	mux.Handle("/feedback", appHandler(addFeedback))
	mux.Handle("/delete_address", appHandler(delAddress))
	mux.Handle("/update_address", appHandler(modAddress))
	mux.Handle("/get_image_token", appHandler(getImageToken))
	mux.Handle("/fetch_wifi", appHandler(fetchWifi))
	mux.Handle("/auto_login", appHandler(autoLogin))
	mux.Handle("/portal_login", appHandler(portalLogin))
	mux.Handle("/get_nearby_aps", appHandler(getAppAps))
	mux.Handle("/report_wifi", appHandler(reportWifi))
	mux.Handle("/report_click", appHandler(reportClick))
	mux.Handle("/report_apmac", appHandler(reportApmac))
	mux.Handle("/connect_wifi", appHandler(connectWifi))
	mux.Handle("/upload_callback", appHandler(uploadCallback))
	mux.Handle("/purchase_sales", appHandler(purchaseSales))
	mux.Handle("/apply_image_upload", appHandler(applyImageUpload))
	mux.Handle("/pingpp_pay", appHandler(pingppPay))
	mux.Handle("/services", appHandler(getService))
	mux.HandleFunc("/jump", jump)
	mux.HandleFunc("/wx_mp_login", wxMpLogin)
	mux.HandleFunc("/get_jsapi_sign", getJsapiSign)
	mux.HandleFunc("/pingpp_webhook", pingppWebhook)
	mux.Handle("/", http.FileServer(http.Dir("/data/server/html")))
	return mux
}
