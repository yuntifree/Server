package main

import (
	"Server/aliyun"
	"Server/httpserver"
	"Server/proto/common"
	"Server/proto/inquiry"
	"Server/proto/pay"
	"Server/util"
	"Server/weixin"
	"encoding/xml"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	simplejson "github.com/bitly/go-simplejson"
)

const (
	succRsp = "<xml><return_code><![CDATA[SUCCESS]]></return_code><return_msg><![CDATA[OK]]></return_msg></xml>"
	failRsp = "<xml><return_code><![CDATA[FAIL]]></return_code><return_msg><![CDATA[SERVER ERROR]]></return_msg></xml>"
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
		&inquiry.LoginRequest{
			Head: &common.Head{Sid: uuid}, Sid: sid,
			Rawdata: rawData, Signature: signature,
			Encrypteddata: encryptedData, Iv: iv})
	httpserver.CheckRPCErr(rpcerr, "Login")
	res := resp.Interface().(*inquiry.LoginReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "Login")

	log.Printf("res:%+v", res)
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

func getMyDoctors(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitInquiry(r)
	uid := req.GetParamInt("uid")
	seq := req.GetParamInt("seq")
	num := req.GetParamInt("num")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.InquiryServerType,
		uid, "GetDoctors",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Seq: seq, Num: num})
	httpserver.CheckRPCErr(rpcerr, "GetDoctors")
	res := resp.Interface().(*inquiry.DoctorsReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "GetDoctors")

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
	gender := req.GetParamInt("gender")
	age := req.GetParamInt("age")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.InquiryServerType,
		uid, "AddPatient",
		&inquiry.PatientRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Info: &inquiry.PatientInfo{Name: name, Phone: phone,
				Mcard: mcard, Gender: gender, Age: age}})
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
	deleted := req.GetParamIntDef("deleted", 0)
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

type image struct {
	filename string `json:"filename"`
}

func uploadImg(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	r.ParseMultipartForm(10 * 1024 * 1024)
	files := r.MultipartForm.File["file"]
	log.Printf("form:%+v", r.MultipartForm)
	var buf []byte
	if len(files) > 0 {
		f, err := files[0].Open()
		if err != nil {
			log.Printf("open file failed:%v", err)
			return &util.AppError{Code: httpserver.ErrInner, Msg: err.Error()}
		}
		buf, err = ioutil.ReadAll(f)
		if err != nil {
			log.Printf("read file failed:%v", err)
			return &util.AppError{Code: httpserver.ErrInner, Msg: err.Error()}
		}
	} else {
		log.Printf("empty file")
		return &util.AppError{Code: httpserver.ErrInner, Msg: "empty file"}
	}
	filename := util.GenUUID() + ".jpg"
	log.Printf("filename :%s", filename)
	flag := aliyun.UploadOssImg(filename, string(buf))
	if !flag {
		log.Printf("UploadOssImg failed")
		return &util.AppError{Code: httpserver.ErrInner, Msg: "upload image failed"}
	}
	filename = aliyun.GenOssImgURL(filename)
	log.Printf("filename upload succ:%s", filename)
	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		log.Printf("writeInfoResp NewJson failed: %v", err)
		w.Write([]byte(`{"errno":103,"desc":"inner failed"}`))
		return
	}
	js.SetPath([]string{"data", "filename"}, filename)

	resp, err := js.Encode()
	if err != nil {
		log.Printf("writeInfoResp NewJson search failed: %v", err)
		w.Write([]byte(`{"errno":103,"desc":"inner failed"}`))
		return
	}
	w.Write(resp)
	return nil
}

func addInquiry(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitInquiry(r)
	uid := req.GetParamInt("uid")
	pid := req.GetParamInt("pid")
	doctor := req.GetParamInt("doctor")
	fee := req.GetParamInt("fee")
	formid := req.GetParamString("formid")
	log.Printf("addInquiry formid:%s", formid)

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.InquiryServerType,
		uid, "AddInquiry",
		&inquiry.InquiryRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Doctor: doctor, Pid: pid, Fee: fee, Formid: formid})
	httpserver.CheckRPCErr(rpcerr, "AddInquiry")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "AddInquiry")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func finInquiry(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitInquiry(r)
	uid := req.GetParamInt("uid")
	tuid := req.GetParamInt("tuid")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.InquiryServerType,
		uid, "FinInquiry",
		&inquiry.FinInquiryRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Tuid: tuid})
	httpserver.CheckRPCErr(rpcerr, "FinInquiry")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "FinInquiry")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func sendChat(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitInquiry(r)
	uid := req.GetParamInt("uid")
	tuid := req.GetParamInt("tuid")
	ctype := req.GetParamInt("type")
	content := req.GetParamString("content")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.InquiryServerType,
		uid, "SendChat",
		&inquiry.ChatRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Tuid: tuid, Type: ctype, Content: content})
	httpserver.CheckRPCErr(rpcerr, "SendChat")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "SendChat")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func getChat(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitInquiry(r)
	uid := req.GetParamInt("uid")
	tuid := req.GetParamInt("tuid")
	seq := req.GetParamInt("seq")
	num := req.GetParamInt("num")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.InquiryServerType,
		uid, "GetChat",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Id: tuid, Seq: seq, Num: num})
	httpserver.CheckRPCErr(rpcerr, "GetChat")
	res := resp.Interface().(*inquiry.ChatReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "GetChat")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func getChatSession(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitInquiry(r)
	uid := req.GetParamInt("uid")
	seq := req.GetParamInt("seq")
	num := req.GetParamInt("num")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.InquiryServerType,
		uid, "GetChatSession",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Seq: seq, Num: num})
	httpserver.CheckRPCErr(rpcerr, "GetChatSession")
	res := resp.Interface().(*inquiry.ChatSessionReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "GetChatSession")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func getWallet(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitInquiry(r)
	uid := req.GetParamInt("uid")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.InquiryServerType,
		uid, "GetWallet",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid}})
	httpserver.CheckRPCErr(rpcerr, "GetWallet")
	res := resp.Interface().(*inquiry.WalletReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "GetWallet")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func applyDraw(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitInquiry(r)
	uid := req.GetParamInt("uid")
	fee := req.GetParamInt("fee")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.InquiryServerType,
		uid, "ApplyDraw",
		&inquiry.DrawRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Fee: fee})
	httpserver.CheckRPCErr(rpcerr, "ApplyDraw")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "ApplyDraw")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func getQRCode(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitInquiry(r)
	uid := req.GetParamInt("uid")
	width := req.GetParamInt("width")
	path := req.GetParamString("path")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.InquiryServerType,
		uid, "GetQRCode",
		&inquiry.QRCodeRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Path: path, Width: width})
	httpserver.CheckRPCErr(rpcerr, "GetQRCode")
	res := resp.Interface().(*inquiry.QRCodeReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "GetQRCode")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func wxPay(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitInquiry(r)
	uid := req.GetParamInt("uid")
	id := req.GetParamInt("id")
	doctor := req.GetParamInt("doctor")
	fee := req.GetParamInt("fee")
	callback := strings.Replace(r.RequestURI, "wx_pay", "wx_pay_callback", -1)
	arr := strings.Split(r.RemoteAddr, ":")
	var clientip string
	if len(arr) > 0 {
		clientip = arr[0]
	}

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.PayServerType,
		uid, "WxPay",
		&pay.WxPayRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Type: 0, Item: id, Tuid: doctor, Fee: fee,
			Clientip: clientip, Callback: callback})
	httpserver.CheckRPCErr(rpcerr, "WxPay")
	res := resp.Interface().(*pay.WxPayReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "WxPay")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}

func wxPayCallback(w http.ResponseWriter, r *http.Request) {
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("wxPayCallback read body failed:%v", err)
		w.Write([]byte(failRsp))
		return
	}
	log.Printf("wxPayCallback request:%s", string(buf))
	var notify weixin.NotifyRequest
	err = xml.Unmarshal(buf, &notify)
	if err != nil {
		log.Printf("wxPayCallback Unmarshal xml failed:%s %v", string(buf), err)
		w.Write([]byte(failRsp))
		return
	}
	if notify.ReturnCode != "SUCCESS" || notify.ResultCode != "SUCCESS" {
		log.Printf("wxPayCallback failed response:%+v", notify)
		w.Write([]byte(succRsp))
		return
	}

	if !weixin.VerifyNotify(notify) {
		log.Printf("wxPayCallback VerifyNotify failed:%+v", notify)
		w.Write([]byte(failRsp))
		return
	}
	w.Write([]byte(succRsp))

	uuid := util.GenUUID()
	rsp, rpcerr := httpserver.CallRPC(util.PayServerType,
		0, "WxPayCB",
		&pay.WxPayCBRequest{Head: &common.Head{Sid: uuid},
			Oid: notify.OutTradeNO, Fee: notify.TotalFee})
	if rpcerr.Interface() != nil {
		log.Printf("WxPayCallback CallRPC failed:%v", err)
		return
	}
	res, ok := rsp.Interface().(*common.CommReply)
	if !ok {
		log.Printf("WxPayCallback assert reply failed:%v", err)
		return
	}
	if res.Head.Retcode != 0 {
		log.Printf("WxPayCallback retcode:%d", res.Head.Retcode)
		return
	}
	return
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
	case "get_my_patients":
		return getPatients(w, r)
	case "add_patient_info":
		return addPatientInfo(w, r)
	case "mod_patient_info":
		return modPatientInfo(w, r)
	case "upload_img":
		return uploadImg(w, r)
	case "add_inquiry":
		return addInquiry(w, r)
	case "fin_inquiry":
		return finInquiry(w, r)
	case "wx_pay":
		return wxPay(w, r)
	case "wx_pay_callback":
		wxPayCallback(w, r)
	case "send_chat":
		sendChat(w, r)
	case "get_chat":
		getChat(w, r)
	case "get_chat_session":
		getChatSession(w, r)
	case "get_my_doctors":
		getMyDoctors(w, r)
	case "get_my_wallet":
		getWallet(w, r)
	case "apply_draw":
		applyDraw(w, r)
	case "get_qrcode":
		getQRCode(w, r)
	default:
		panic(util.AppError{101, "unknown action", ""})
	}
	return nil
}
