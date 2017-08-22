package main

import (
	"Server/httpserver"
	"Server/proto/common"
	"Server/proto/userinfo"
	"Server/util"
	"net/http"
)

func getUserScore(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckApp(r)
	uid := req.GetParamInt("uid")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.UserinfoServerType, uid, "GetUserScore",
		&common.CommRequest{
			Head: &common.Head{Sid: uuid, Uid: uid}})
	httpserver.CheckRPCErr(rpcerr, "GetUserScore")
	res := resp.Interface().(*userinfo.ScoreReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "GetUserScore")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
	return nil
}
