package main

import (
	"Server/httpserver"
	"Server/proto/common"
	"Server/proto/inquiry"
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

func inquiryHandler(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	log.Printf("path:%s", r.URL.Path)
	action := extractAction(r.URL.Path)
	switch action {
	case "submit_code":
		submitCode(w, r)
	default:
		panic(util.AppError{101, "unknown action", ""})
	}
	return nil
}
