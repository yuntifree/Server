package main

import (
	"Server/httpserver"
	"Server/proto/common"
	"Server/proto/inquiry"
	"Server/proto/punch"
	"Server/util"
	"log"
	"net/http"
	"strings"
)

func extractAction(path string) string {
	pos := strings.LastIndex(path, "/")
	var action string
	if pos != -1 {
		action = path[pos+1:]
	}
	return action
}

func submitCode(w http.ResponseWriter, r *http.Request) {
	var req httpserver.Request
	req.Init(r)
	code := req.GetParamString("code")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.InquiryServerType, 0, "SubmitCode",
		&inquiry.CodeRequest{
			Head: &common.Head{Sid: uuid}, Code: code})
	httpserver.CheckRPCErr(rpcerr, "SubmitCode")
	res := resp.Interface().(*inquiry.LoginReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "SubmitCode")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
}

func inquiryLogin(w http.ResponseWriter, r *http.Request) {
	var req httpserver.Request
	req.Init(r)
	sid := req.GetParamString("sid")
	rawData := req.GetParamString("rawData")
	signature := req.GetParamString("signature")
	encryptedData := req.GetParamString("encryptedData")
	iv := req.GetParamString("iv")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.InquiryServerType, 0, "Login",
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
}

func getInquiryPhoneCode(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitInquiry(r)
	phone := req.GetParamString("phone")
	uid := req.GetParamInt("uid")

	if !util.IsIllegalPhone(phone) {
		log.Printf("getPhoneCode illegal phone:%s", phone)
		return &util.AppError{Code: httpserver.ErrIllegalPhone,
			Msg: "请输入正确的手机号"}
	}

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.InquiryServerType,
		uid, "GetPhoneCode",
		&inquiry.PhoneRequest{Head: &common.Head{Sid: uuid},
			Phone: phone})
	httpserver.CheckRPCErr(rpcerr, "GetPhoneCode")
	res := resp.Interface().(*common.CommReply)

	if res.Head.Retcode != 0 {
		return &util.AppError{Code: httpserver.ErrCode, Msg: "获取验证码失败"}
	}
	w.Write([]byte(`{"errno":0}`))
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func bindPhone(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitInquiry(r)
	phone := req.GetParamString("phone")
	uid := req.GetParamInt("uid")
	code := req.GetParamInt("code")

	if !util.IsIllegalPhone(phone) {
		log.Printf("getPhoneCode illegal phone:%s", phone)
		return &util.AppError{Code: httpserver.ErrIllegalPhone,
			Msg: "请输入正确的手机号"}
	}

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.InquiryServerType,
		uid, "BindPhone",
		&inquiry.PhoneCodeRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Phone: phone, Code: code})
	httpserver.CheckRPCErr(rpcerr, "BindPhone")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "BindPhone")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func inquiryHandler(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	log.Printf("path:%s", r.URL.Path)
	action := extractAction(r.URL.Path)
	switch action {
	case "submit_code":
		submitCode(w, r)
	case "login":
		inquiryLogin(w, r)
	case "get_phone_code":
		return getInquiryPhoneCode(w, r)
	case "bind_phone":
		return bindPhone(w, r)
	default:
		panic(util.AppError{101, "unknown action", ""})
	}
	return nil
}
