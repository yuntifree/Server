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
