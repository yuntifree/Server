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
	case "get_ap_info":
		getApInfo(w, r)
	case "add_ap_info":
		addApInfo(w, r)
	case "mod_ap_info":
		modApInfo(w, r)
	case "get_ad_banner":
		getAdBanner(w, r)
	case "add_ad_banner":
		addAdBanner(w, r)
	case "mod_ad_banner":
		modAdBanner(w, r)
	case "get_travel_ad":
		getTravelAd(w, r)
	case "add_travel_ad":
		addTravelAd(w, r)
	case "mod_travel_ad":
		modTravelAd(w, r)
	default:
		panic(util.AppError{101, "unknown action", ""})
	}
	return nil
}

func getAdBanner(w http.ResponseWriter, r *http.Request) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	seq := req.GetParamInt("seq")
	num := req.GetParamInt("num")
	stype := req.GetParamInt("type")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ConfigServerType, uid, "GetAdBanner",
		&common.CommRequest{
			Head: &common.Head{Sid: uuid}, Seq: seq,
			Num: num, Type: stype})
	httpserver.CheckRPCErr(rpcerr, "GetAdBanner")
	res := resp.Interface().(*config.AdBannerReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "GetAdBanner")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
}

func addAdBanner(w http.ResponseWriter, r *http.Request) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	mtype := req.GetParamInt("type")
	stype := req.GetParamInt("stype")
	img := req.GetParamString("img")
	dst := req.GetParamString("dst")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ConfigServerType, uid, "AddAdBanner",
		&config.AdBannerRequest{
			Head: &common.Head{Sid: uuid},
			Info: &config.AdBannerInfo{Type: mtype, Stype: stype,
				Img: img, Dst: dst}})
	httpserver.CheckRPCErr(rpcerr, "AddAdBanner")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "AddAdBanner")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
}

func modAdBanner(w http.ResponseWriter, r *http.Request) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	id := req.GetParamInt("id")
	stype := req.GetParamInt("stype")
	img := req.GetParamString("img")
	dst := req.GetParamString("dst")
	online := req.GetParamIntDef("online", 0)
	deleted := req.GetParamIntDef("deleted", 0)

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ConfigServerType, uid, "ModAdBanner",
		&config.AdBannerRequest{
			Head: &common.Head{Sid: uuid},
			Info: &config.AdBannerInfo{Id: id, Stype: stype,
				Img: img, Dst: dst, Online: online, Deleted: deleted}})
	httpserver.CheckRPCErr(rpcerr, "ModAdBanner")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "ModAdBanner")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
}

func getApInfo(w http.ResponseWriter, r *http.Request) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	seq := req.GetParamInt("seq")
	num := req.GetParamInt("num")
	search := req.GetParamStringDef("search", "")
	search = strings.ToLower(search)

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ConfigServerType, uid, "GetApInfo",
		&common.CommRequest{
			Head: &common.Head{Sid: uuid}, Search: search, Seq: seq,
			Num: num})
	httpserver.CheckRPCErr(rpcerr, "GetApInfo")
	res := resp.Interface().(*config.ApInfoReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "GetApInfo")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
}

func addApInfo(w http.ResponseWriter, r *http.Request) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	unid := req.GetParamInt("unid")
	longitude := req.GetParamFloat("longitude")
	latitude := req.GetParamFloat("latitude")
	mac := req.GetParamString("mac")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ConfigServerType, uid, "AddApInfo",
		&config.ApInfoRequest{
			Head: &common.Head{Sid: uuid},
			Info: &config.ApInfo{Mac: mac, Longitude: longitude,
				Latitude: latitude, Unid: unid}})
	httpserver.CheckRPCErr(rpcerr, "AddApInfo")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "AddApInfo")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
}

func modApInfo(w http.ResponseWriter, r *http.Request) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	id := req.GetParamInt("id")
	unid := req.GetParamInt("unid")
	longitude := req.GetParamFloat("longitude")
	latitude := req.GetParamFloat("latitude")
	deleted := req.GetParamIntDef("deleted", 0)

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ConfigServerType, uid, "ModApInfo",
		&config.ApInfoRequest{
			Head: &common.Head{Sid: uuid},
			Info: &config.ApInfo{Id: id, Longitude: longitude,
				Latitude: latitude, Unid: unid, Deleted: deleted}})
	httpserver.CheckRPCErr(rpcerr, "ModApInfo")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "ModApInfo")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
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
			Head: &common.Head{Sid: uuid, Uid: uid},
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

func getTravelAd(w http.ResponseWriter, r *http.Request) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	stype := req.GetParamInt("type")
	seq := req.GetParamInt("seq")
	num := req.GetParamInt("num")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ConfigServerType, uid, "FetchTravelAd",
		&common.CommRequest{
			Head: &common.Head{Sid: uuid}, Type: stype, Seq: seq, Num: num})
	httpserver.CheckRPCErr(rpcerr, "FetchTravelAd")
	res := resp.Interface().(*config.TravelAdReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "FetchTravelAd")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
}

func addTravelAd(w http.ResponseWriter, r *http.Request) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	stype := req.GetParamInt("type")
	stime := req.GetParamIntDef("stime", 0)
	etime := req.GetParamIntDef("etime", 0)
	img := req.GetParamString("img")
	title := req.GetParamString("title")
	dst := req.GetParamString("dst")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ConfigServerType,
		uid, "AddTravelAd",
		&config.TravelAdRequest{
			Head: &common.Head{Sid: uuid},
			Info: &config.TravelAdInfo{Type: stype, Title: title,
				Dst: dst, Stime: stime,
				Etime: etime, Img: img}})
	httpserver.CheckRPCErr(rpcerr, "AddTravelAd")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "AddTravelAd")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
}

func modTravelAd(w http.ResponseWriter, r *http.Request) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	id := req.GetParamInt("id")
	stime := req.GetParamIntDef("stime", 0)
	etime := req.GetParamIntDef("etime", 0)
	img := req.GetParamString("img")
	title := req.GetParamString("title")
	dst := req.GetParamString("dst")
	online := req.GetParamIntDef("online", 0)
	deleted := req.GetParamIntDef("deleted", 0)

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ConfigServerType,
		uid, "ModTravelAd",
		&config.TravelAdRequest{
			Head: &common.Head{Sid: uuid, Uid: uid},
			Info: &config.TravelAdInfo{Id: id,
				Title: title, Dst: dst,
				Stime: stime, Etime: etime, Img: img,
				Online: online, Deleted: deleted}})
	httpserver.CheckRPCErr(rpcerr, "ModTravelAd")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "ModTravelAd")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	httpserver.ReportSuccResp(r.RequestURI)
}
