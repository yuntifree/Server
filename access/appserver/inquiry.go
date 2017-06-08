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

	simplejson "github.com/bitly/go-simplejson"
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
	res := resp.Interface().(*inquiry.RoleReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "BindPhone")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func writeInfoResp(w http.ResponseWriter, info interface{}) {
	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		log.Printf("writeInfoResp NewJson failed: %v", err)
		w.Write([]byte(`{"errno":103,"desc":"inner failed"}`))
		return
	}
	js.Set("data", info)

	resp, err := js.Encode()
	if err != nil {
		log.Printf("writeInfoResp NewJson search failed: %v", err)
		w.Write([]byte(`{"errno":103,"desc":"inner failed"}`))
		return
	}
	w.Write(resp)
}

func getDoctorInfo(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitInquiry(r)
	uid := req.GetParamInt("uid")
	tuid := req.GetParamInt("tuid")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.InquiryServerType,
		uid, "GetDoctorInfo",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Id: tuid})
	httpserver.CheckRPCErr(rpcerr, "GetDoctorInfo")
	res := resp.Interface().(*inquiry.DoctorInfoReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "GetDoctorInfo")

	writeInfoResp(w, res.Info)
	return nil
}

func getPatientInfo(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitInquiry(r)
	uid := req.GetParamInt("uid")
	tuid := req.GetParamInt("tuid")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.InquiryServerType,
		uid, "GetPatientInfo",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Id: tuid})
	httpserver.CheckRPCErr(rpcerr, "GetPatientInfo")
	res := resp.Interface().(*inquiry.PatientInfoReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "GetPatientInfo")

	writeInfoResp(w, res.Info)
	return nil
}

func setFee(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitInquiry(r)
	uid := req.GetParamInt("uid")
	fee := req.GetParamInt("fee")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.InquiryServerType,
		uid, "SetFee",
		&inquiry.FeeRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Fee: fee})
	httpserver.CheckRPCErr(rpcerr, "SetFee")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "SetFee")

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func bindOp(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitInquiry(r)
	otype := req.GetParamInt("type")
	tuid := req.GetParamInt("tuid")
	uid := req.GetParamInt("uid")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.InquiryServerType,
		uid, "BindOp",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Type: otype, Id: tuid})
	httpserver.CheckRPCErr(rpcerr, "BindOp")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "BindOp")

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func getPatients(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitInquiry(r)
	uid := req.GetParamInt("uid")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.InquiryServerType,
		uid, "GetPatients",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid}})
	httpserver.CheckRPCErr(rpcerr, "GetPatients")
	res := resp.Interface().(*inquiry.PatientsReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "GetPatients")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func addPatientInfo(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitInquiry(r)
	uid := req.GetParamInt("uid")
	name := req.GetParamString("name")
	phone := req.GetParamString("phone")
	mcard := req.GetParamString("mcard")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.InquiryServerType,
		uid, "AddPatient",
		&inquiry.PatientRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Info: &inquiry.PatientInfo{Name: name, Phone: phone,
				Mcard: mcard}})
	httpserver.CheckRPCErr(rpcerr, "AddPatient")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "AddPatient")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func modPatientInfo(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitInquiry(r)
	uid := req.GetParamInt("uid")
	id := req.GetParamInt("id")
	deleted := req.GetParamInt("deleted")
	name := req.GetParamString("name")
	phone := req.GetParamString("phone")
	mcard := req.GetParamString("mcard")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.InquiryServerType,
		uid, "ModPatient",
		&inquiry.PatientRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Info: &inquiry.PatientInfo{Name: name, Phone: phone,
				Mcard: mcard, Id: id, Deleted: deleted}})
	httpserver.CheckRPCErr(rpcerr, "ModPatient")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "MoPatient")

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
	case "get_doctor_info":
		return getDoctorInfo(w, r)
	case "get_patient_info":
		return getPatientInfo(w, r)
	case "set_fee":
		return setFee(w, r)
	case "bind_op":
		return bindOp(w, r)
	case "get_patients":
		return getPatients(w, r)
	case "add_patient_info":
		return addPatientInfo(w, r)
	case "mod_patient_info":
		return modPatientInfo(w, r)
	default:
		panic(util.AppError{101, "unknown action", ""})
	}
	return nil
}
