package main

import (
	"Server/httpserver"
	"Server/proto/common"
	"Server/proto/config"
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

func configHandler(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	log.Printf("path:%s", r.URL.Path)
	action := extractAction(r.URL.Path)
	switch action {
	case "get_login_img":
		getLoginImg(w, r)
	case "add_login_img":
		addLoginImg(w, r)
	case "mod_login_img":
		modLoginImg(w, r)
	default:
		panic(util.AppError{101, "unknown action", ""})
	}
	return nil
}

func getLoginImg(w http.ResponseWriter, r *http.Request) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	stype := req.GetParamInt("type")
	seq := req.GetParamInt("seq")
	num := req.GetParamInt("num")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ConfigServerType, uid, "GetLoginImg",
		&common.CommRequest{
			Head: &common.Head{Sid: uuid}, Type: stype, Seq: seq, Num: num})
	httpserver.CheckRPCErr(rpcerr, "GetLoginImg")
	res := resp.Interface().(*config.LoginImgReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "GetLoginImg")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
}

func addLoginImg(w http.ResponseWriter, r *http.Request) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	stype := req.GetParamInt("type")
	stime := req.GetParamIntDef("stime", 0)
	etime := req.GetParamIntDef("etime", 0)
	img := req.GetParamString("img")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ConfigServerType,
		uid, "AddLoginImg",
		&config.LoginImgRequest{
			Head: &common.Head{Sid: uuid},
			Info: &config.LoginImgInfo{Type: stype, Stime: stime,
				Etime: etime, Img: img}})
	httpserver.CheckRPCErr(rpcerr, "AddLoginImg")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "AddLoginImg")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
}

func modLoginImg(w http.ResponseWriter, r *http.Request) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	id := req.GetParamInt("id")
	stime := req.GetParamIntDef("stime", 0)
	etime := req.GetParamIntDef("etime", 0)
	img := req.GetParamString("img")
	online := req.GetParamIntDef("online", 0)
	deleted := req.GetParamIntDef("deleted", 0)

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ConfigServerType,
		uid, "ModLoginImg",
		&config.LoginImgRequest{
			Head: &common.Head{Sid: uuid},
			Info: &config.LoginImgInfo{Id: id,
				Stime: stime, Etime: etime, Img: img,
				Online: online, Deleted: deleted}})
	httpserver.CheckRPCErr(rpcerr, "ModLoginImg")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "ModLoginImg")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
}
