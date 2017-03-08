package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"

	"Server/aliyun"
	"Server/httpserver"
	"Server/proto/advertise"
	"Server/proto/common"
	"Server/proto/config"
	"Server/proto/fetch"
	"Server/proto/modify"
	"Server/proto/monitor"
	"Server/proto/push"
	"Server/proto/verify"
	"Server/util"

	simplejson "github.com/bitly/go-simplejson"
)

var roleConf *simplejson.Json

func initRoleConf() {
	file, err := os.Open("role.json")
	if err != nil {
		log.Fatal("open role.json failed:%v", err)
	}
	roleConf, err = simplejson.NewFromReader(file)
	if err != nil {
		log.Fatal("parse role.json failed:%v", err)
	}
}

func genReqNum(num int64) int64 {
	if num > 100 {
		num = 100
	} else if num < 20 {
		num = 20
	}
	return num
}

func backLogin(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.Init(r)
	username := req.GetParamString("username")
	password := req.GetParamString("password")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.VerifyServerType, 0, "BackLogin",
		&verify.LoginRequest{Head: &common.Head{Sid: uuid},
			Username: username, Password: password})
	httpserver.CheckRPCErr(rpcerr, "BackLogin")
	res := resp.Interface().(*verify.LoginReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "BackLogin")

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{httpserver.ErrInner, "invalid param", ""}
	}
	role := strconv.Itoa(int(res.Role))
	initRoleConf()
	js.SetPath([]string{"data", "uid"}, res.Head.Uid)
	js.SetPath([]string{"data", "token"}, res.Token)
	js.SetPath([]string{"data", "roleconf"}, roleConf.Get(role))

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{httpserver.ErrInner, "marshal json failed", ""}
	}

	w.Write(body)
	return nil
}

func getReviewNews(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	num := req.GetParamInt("num")
	seq := req.GetParamInt("seq")
	ctype := req.GetParamInt("type")
	stype := req.GetParamIntDef("stype", 0)
	search := req.GetParamStringDef("search", "")
	num = genReqNum(num)

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.FetchServerType, uid, "FetchReviewNews",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Seq: seq, Num: num, Type: ctype, Subtype: stype, Search: search})
	httpserver.CheckRPCErr(rpcerr, "FetchReviewNews")
	res := resp.Interface().(*fetch.NewsReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "FetchReviewNews")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	return nil
}

func getTags(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	num := req.GetParamInt("num")
	seq := req.GetParamInt("seq")
	num = genReqNum(num)

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.FetchServerType, uid, "FetchTags",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Seq: seq, Num: num})
	httpserver.CheckRPCErr(rpcerr, "FetchTags")
	res := resp.Interface().(*fetch.TagsReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "FetchTags")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	return nil
}

func getUsers(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	num := req.GetParamInt("num")
	seq := req.GetParamInt("seq")
	num = genReqNum(num)

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.FetchServerType, uid, "FetchUsers",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Seq: seq, Num: num})
	httpserver.CheckRPCErr(rpcerr, "FetchUsers")
	res := resp.Interface().(*fetch.UserReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "FetchUsers")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	return nil
}

func reviewVideo(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	id := req.GetParamInt("id")
	reject := req.GetParamInt("reject")
	mod := req.GetParamIntDef("modify", 0)
	var title string
	if mod != 0 {
		title = req.GetParamStringDef("title", "")
	}

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ModifyServerType, uid, "ReviewVideo",
		&modify.VideoRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Id: id, Reject: reject == 1,
			Modify: mod == 1, Title: title})
	httpserver.CheckRPCErr(rpcerr, "ReviewVideo")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "ReviewVideo")

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func reviewNews(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	id := req.GetParamInt("id")
	reject := req.GetParamInt("reject")
	mod := req.GetParamIntDef("modify", 0)
	var title string
	var tags []int64

	if mod != 0 {
		title = req.GetParamStringDef("title", "")
	}

	arr, err := req.Post.Get("data").Get("tags").Array()
	if err == nil {
		for i := 0; i < len(arr); i++ {
			tid, _ := req.Post.Get("data").Get("tags").GetIndex(i).Int64()
			tags = append(tags, tid)
		}
	}

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ModifyServerType, uid, "ReviewNews",
		&modify.NewsRequest{Head: &common.Head{Sid: uuid, Uid: uid}, Id: id, Reject: reject == 1,
			Modify: mod == 1, Title: title, Tags: tags})
	httpserver.CheckRPCErr(rpcerr, "ReviewNews")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "ReviewNews")

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func getApStat(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	num := req.GetParamInt("num")
	seq := req.GetParamInt("seq")
	num = genReqNum(num)

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.FetchServerType, uid, "FetchApStat",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Seq: seq, Num: num})
	httpserver.CheckRPCErr(rpcerr, "FetchApStat")
	res := resp.Interface().(*fetch.ApStatReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "FetchApStat")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	return nil
}

func getVideos(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	num := req.GetParamInt("num")
	seq := req.GetParamInt("seq")
	ctype := req.GetParamInt("type")
	search := req.GetParamStringDef("search", "")
	num = genReqNum(num)

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.FetchServerType, uid, "FetchVideos",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Seq: seq, Num: num, Type: ctype, Search: search})
	httpserver.CheckRPCErr(rpcerr, "FetchVideos")
	res := resp.Interface().(*fetch.VideoReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "FetchVideos")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	return nil
}

func getTemplates(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	num := req.GetParamInt("num")
	seq := req.GetParamInt("seq")
	num = genReqNum(num)

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.FetchServerType, uid, "FetchTemplates",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Seq: seq, Num: num})
	httpserver.CheckRPCErr(rpcerr, "FetchTemplates")
	res := resp.Interface().(*fetch.TemplateReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "FetchTemplates")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	return nil
}

func getOssConf(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	return httpserver.GetConf(w, r, true)
}

func getAdBan(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.FetchServerType, uid, "FetchAdBan",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid}})
	httpserver.CheckRPCErr(rpcerr, "FetchAdBan")
	res := resp.Interface().(*fetch.AdBanReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "FetchAdBan")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	return nil
}

func getApi(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.MonitorServerType, uid, "GetApi",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid}})
	httpserver.CheckRPCErr(rpcerr, "GetApi")
	res := resp.Interface().(*monitor.ApiReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "GetApi")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	return nil
}

func fetchPortalMenu(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	stype := req.GetParamInt("type")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ConfigServerType, uid, "FetchPortalMenu",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid}, Type: stype})
	httpserver.CheckRPCErr(rpcerr, "FetchPortalMenu")
	res := resp.Interface().(*config.MenuReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "FetchPortalMenu")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	return nil
}

func modPortalMenu(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	id := req.GetParamInt("id")
	icon := req.GetParamStringDef("icon", "")
	text := req.GetParamStringDef("text", "")
	name := req.GetParamStringDef("name", "")
	routername := req.GetParamStringDef("routername", "")
	url := req.GetParamStringDef("url", "")
	priority := req.GetParamIntDef("priority", 0)
	dbg := req.GetParamInt("dbg")
	deleted := req.GetParamInt("deleted")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ConfigServerType, uid, "ModPortalMenu",
		&config.MenuRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Info: &config.PortalMenuInfo{Id: id, Icon: icon, Text: text, Name: name,
				Routername: routername, Url: url, Priority: priority,
				Dbg: dbg, Deleted: deleted}})
	httpserver.CheckRPCErr(rpcerr, "ModPortalMenu")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "ModPortalMenu")

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func addPortalMenu(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	stype := req.GetParamIntDef("type", 0)
	icon := req.GetParamStringDef("icon", "")
	text := req.GetParamStringDef("text", "")
	name := req.GetParamStringDef("name", "")
	routername := req.GetParamStringDef("routername", "")
	url := req.GetParamStringDef("url", "")
	priority := req.GetParamIntDef("priority", 0)
	dbg := req.GetParamIntDef("dbg", 0)
	deleted := req.GetParamIntDef("deleted", 0)

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ConfigServerType, uid, "AddPortalMenu",
		&config.MenuRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Info: &config.PortalMenuInfo{Type: stype, Icon: icon, Text: text,
				Name: name, Routername: routername, Url: url, Priority: priority,
				Dbg: dbg, Deleted: deleted}})
	httpserver.CheckRPCErr(rpcerr, "AddPortalMenu")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "AddPortalMenu")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	return nil
}

func getBatchApiStat(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	num := req.GetParamInt("num")

	var names []string
	arr, err := req.Post.Get("data").Get("names").Array()
	if err == nil {
		for i := 0; i < len(arr); i++ {
			name, _ := req.Post.Get("data").Get("names").GetIndex(i).String()
			names = append(names, name)
		}
	}

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.MonitorServerType, uid, "GetBatchApiStat",
		&monitor.BatchApiStatRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Names: names, Num: num})
	httpserver.CheckRPCErr(rpcerr, "GetBatchApiStat")
	res := resp.Interface().(*monitor.BatchApiStatReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "GetBatchApiStat")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	return nil
}

func addTemplate(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	title := req.GetParamString("title")
	content := req.GetParamString("content")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ModifyServerType, uid, "AddTemplate",
		&modify.AddTempRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Info: &modify.TemplateInfo{Title: title, Content: content}})
	httpserver.CheckRPCErr(rpcerr, "AddTemplate")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "AddTemplate")

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{httpserver.ErrInner, "invalid param", ""}
	}
	js.SetPath([]string{"data", "tid"}, res.Id)

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{httpserver.ErrInner, "marshal json failed", ""}
	}
	w.Write(body)
	return nil
}

func addBanner(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	img := req.GetParamString("img")
	dst := req.GetParamString("dst")
	priority := req.GetParamInt("priority")
	btype := req.GetParamInt("type")
	title := req.GetParamStringDef("title", "")
	expire := req.GetParamStringDef("expire", "")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ModifyServerType, uid, "AddBanner",
		&modify.BannerRequest{
			Head: &common.Head{Sid: uuid, Uid: uid},
			Info: &common.BannerInfo{Img: img, Dst: dst, Priority: priority,
				Title: title, Type: btype, Expire: expire}})
	httpserver.CheckRPCErr(rpcerr, "AddBanner")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "AddBanner")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	return nil
}

func addConf(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	key := req.GetParamString("key")
	val := req.GetParamString("val")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ModifyServerType, uid, "AddConf",
		&modify.ConfRequest{
			Head: &common.Head{Sid: uuid, Uid: uid},
			Info: &common.KvInfo{Key: key, Val: val}})
	httpserver.CheckRPCErr(rpcerr, "AddConf")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "AddConf")

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func addAdBan(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	term := req.GetParamInt("term")
	version := req.GetParamInt("version")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ModifyServerType, uid, "AddAdBan",
		&modify.AddBanRequest{
			Head: &common.Head{Sid: uuid, Uid: uid},
			Info: &common.AdBan{Term: term, Version: version}})
	httpserver.CheckRPCErr(rpcerr, "AddAdBan")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "AddAdBan")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	return nil
}

func delAdBan(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")

	var ids []int64
	arr, err := req.Post.Get("data").Get("ids").Array()
	if err == nil {
		for i := 0; i < len(arr); i++ {
			tid, _ := req.Post.Get("data").Get("ids").GetIndex(i).Int64()
			ids = append(ids, tid)
		}
	}

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ModifyServerType, uid, "DelAdBan",
		&modify.DelBanRequest{
			Head: &common.Head{Sid: uuid, Uid: uid},
			Ids:  ids})
	httpserver.CheckRPCErr(rpcerr, "DelAdBan")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "DelAdBan")

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func getWhiteList(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	seq := req.GetParamInt("seq")
	num := req.GetParamInt("num")
	wtype := req.GetParamInt("type")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.FetchServerType, uid, "FetchWhiteList",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Seq: seq, Num: num, Type: wtype})
	httpserver.CheckRPCErr(rpcerr, "FetchWhiteList")
	res := resp.Interface().(*fetch.WhiteReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "FetchWhiteList")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	return nil
}

func addWhiteList(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	wtype := req.GetParamInt("type")
	var ids []int64
	arr, err := req.Post.Get("data").Get("uids").Array()
	if err == nil {
		for i := 0; i < len(arr); i++ {
			tid, _ := req.Post.Get("data").Get("uids").GetIndex(i).Int64()
			ids = append(ids, tid)
		}
	}

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ModifyServerType, uid, "AddWhiteList",
		&modify.WhiteRequest{
			Head: &common.Head{Sid: uuid, Uid: uid},
			Type: wtype, Ids: ids})
	httpserver.CheckRPCErr(rpcerr, "AddWhiteList")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "AddWhiteList")

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func addPortalDir(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	ptype := req.GetParamInt("type")
	desc := req.GetParamStringDef("description", "")
	dir := req.GetParamString("dir")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ModifyServerType, uid, "AddPortalDir",
		&modify.PortalDirRequest{
			Head: &common.Head{Sid: uuid, Uid: uid},
			Info: &common.PortalDirInfo{Type: ptype, Dir: dir, Description: desc}})
	httpserver.CheckRPCErr(rpcerr, "AddPortalDir")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "AddPortalDir")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	return nil
}

func onlinePortalDir(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	id := req.GetParamInt("id")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ModifyServerType, uid, "OnlinePortalDir",
		&common.CommRequest{
			Head: &common.Head{Sid: uuid, Uid: uid},
			Id:   id})
	httpserver.CheckRPCErr(rpcerr, "OnlinePortalDir")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "OnlinePortalDir")

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func delWhiteList(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	wtype := req.GetParamInt("type")
	var ids []int64
	arr, err := req.Post.Get("data").Get("uids").Array()
	if err == nil {
		for i := 0; i < len(arr); i++ {
			tid, _ := req.Post.Get("data").Get("uids").GetIndex(i).Int64()
			ids = append(ids, tid)
		}
	}

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ModifyServerType, uid, "DelWhiteList",
		&modify.WhiteRequest{
			Head: &common.Head{Sid: uuid, Uid: uid},
			Type: wtype, Ids: ids})
	httpserver.CheckRPCErr(rpcerr, "DelWhiteList")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "DelWhiteList")

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func addTags(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	tags, err := req.Post.Get("data").Get("tags").Array()
	if err != nil {
		log.Printf("get tags failed:%v", err)
		return &util.AppError{httpserver.ErrInvalidParam, err.Error(), ""}
	}

	var cts []string
	for i := 0; i < len(tags); i++ {
		tag := tags[i].(string)
		cts = append(cts, tag)
	}

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ModifyServerType, uid, "AddTags",
		&modify.AddTagRequest{
			Head: &common.Head{Sid: uuid, Uid: uid},
			Tags: cts})
	httpserver.CheckRPCErr(rpcerr, "AddTags")
	res := resp.Interface().(*modify.AddTagReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "AddTags")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	return nil
}

func addCustomer(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	name := req.GetParamString("name")
	contact := req.GetParamString("contact")
	phone := req.GetParamString("phone")
	address := req.GetParamString("address")
	atime := req.GetParamString("atime")
	etime := req.GetParamString("etime")
	remark := req.GetParamStringDef("remark", "")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.AdvertiseServerType, uid, "AddCustomer",
		&advertise.CustomerRequest{
			Head: &common.Head{Sid: uuid, Uid: uid},
			Info: &advertise.CustomerInfo{Name: name, Contact: contact,
				Phone: phone, Address: address,
				Atime: atime, Etime: etime, Remark: remark}})
	httpserver.CheckRPCErr(rpcerr, "AddCustomer")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "AddCustomer")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	return nil
}

func addUnit(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	name := req.GetParamString("name")
	address := req.GetParamString("address")
	longitude := req.GetParamFloat("longitude")
	latitude := req.GetParamFloat("latitude")
	cnt := req.GetParamInt("cnt")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.AdvertiseServerType, uid, "AddUnit",
		&advertise.UnitRequest{
			Head: &common.Head{Sid: uuid, Uid: uid},
			Info: &advertise.UnitInfo{Name: name, Address: address,
				Longitude: longitude, Latitude: latitude, Cnt: cnt}})
	httpserver.CheckRPCErr(rpcerr, "AddUnit")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "AddUnit")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	return nil
}

func modUnit(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	name := req.GetParamString("name")
	address := req.GetParamString("address")
	longitude := req.GetParamFloat("longitude")
	latitude := req.GetParamFloat("latitude")
	cnt := req.GetParamInt("cnt")
	id := req.GetParamInt("id")
	deleted := req.GetParamIntDef("deleted", 0)

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.AdvertiseServerType, uid, "ModUnit",
		&advertise.UnitRequest{
			Head: &common.Head{Sid: uuid, Uid: uid},
			Info: &advertise.UnitInfo{ID: id, Name: name, Address: address,
				Longitude: longitude, Latitude: latitude,
				Cnt: cnt, Deleted: deleted}})
	httpserver.CheckRPCErr(rpcerr, "ModUnit")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "ModUnit")

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func getUnit(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	seq := req.GetParamInt("seq")
	num := req.GetParamInt("num")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.AdvertiseServerType, uid, "FetchUnit",
		&common.CommRequest{
			Head: &common.Head{Uid: uid, Sid: uuid},
			Seq:  seq,
			Num:  num,
		})
	httpserver.CheckRPCErr(rpcerr, "FetchUnit")
	res := resp.Interface().(*advertise.UnitReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "FetchUnit")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	return nil
}

func addAdvertise(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	name := req.GetParamString("name")
	version := req.GetParamString("version")
	adid := req.GetParamInt("adid")
	areaid := req.GetParamInt("areaid")
	tsid := req.GetParamInt("tsid")
	abstract := req.GetParamString("abstract")
	content := req.GetParamString("content")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.AdvertiseServerType, uid, "AddAdvertise",
		&advertise.AdvertiseRequest{
			Head: &common.Head{Sid: uuid, Uid: uid},
			Info: &advertise.AdvertiseInfo{Name: name, Version: version,
				Adid: adid, Areaid: areaid, Tsid: tsid, Abstract: abstract,
				Content: content}})
	httpserver.CheckRPCErr(rpcerr, "AddAdvertise")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "AddAdvertise")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	return nil
}

func modAdvertise(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	id := req.GetParamInt("id")
	name := req.GetParamString("name")
	version := req.GetParamString("version")
	adid := req.GetParamInt("adid")
	areaid := req.GetParamInt("areaid")
	tsid := req.GetParamInt("tsid")
	abstract := req.GetParamString("abstract")
	content := req.GetParamString("content")
	deleted := req.GetParamIntDef("deleted", 0)

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.AdvertiseServerType, uid, "ModAdvertise",
		&advertise.AdvertiseRequest{
			Head: &common.Head{Sid: uuid, Uid: uid},
			Info: &advertise.AdvertiseInfo{ID: id, Name: name, Version: version,
				Adid: adid, Areaid: areaid, Tsid: tsid, Abstract: abstract,
				Content: content, Deleted: deleted}})
	httpserver.CheckRPCErr(rpcerr, "ModAdvertise")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "ModAdvertise")

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func getAdvertise(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	seq := req.GetParamInt("seq")
	num := req.GetParamInt("num")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.AdvertiseServerType, uid, "FetchAdvertise",
		&common.CommRequest{
			Head: &common.Head{Uid: uid, Sid: uuid},
			Seq:  seq,
			Num:  num,
		})
	httpserver.CheckRPCErr(rpcerr, "FetchAdvertise")
	res := resp.Interface().(*advertise.AdvertiseReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "FetchAdvertise")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	return nil
}

func getCustomer(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	seq := req.GetParamInt("seq")
	num := req.GetParamInt("num")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.AdvertiseServerType, uid, "FetchCustomer",
		&common.CommRequest{
			Head: &common.Head{Uid: uid, Sid: uuid},
			Seq:  seq,
			Num:  num,
		})
	httpserver.CheckRPCErr(rpcerr, "FetchCustomer")
	res := resp.Interface().(*advertise.CustomerReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "FetchCustomer")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	return nil
}

func modCustomer(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	id := req.GetParamInt("id")
	name := req.GetParamString("name")
	contact := req.GetParamString("contact")
	phone := req.GetParamString("phone")
	address := req.GetParamString("address")
	atime := req.GetParamString("atime")
	etime := req.GetParamString("etime")
	remark := req.GetParamStringDef("remark", "")
	deleted := req.GetParamIntDef("deleted", 0)

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.AdvertiseServerType, uid, "ModCustomer",
		&advertise.CustomerRequest{
			Head: &common.Head{Sid: uuid, Uid: uid},
			Info: &advertise.CustomerInfo{ID: id, Name: name, Contact: contact,
				Phone: phone, Address: address,
				Atime: atime, Etime: etime, Remark: remark,
				Deleted: deleted}})
	httpserver.CheckRPCErr(rpcerr, "ModCustomer")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "ModCustomer")

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func sendMipush(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	desc := req.GetParamString("desc")
	content := req.GetParamString("content")
	target := req.GetParamString("target")
	term := req.GetParamInt("term")
	pushtype := req.GetParamInt("pushtype")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ModifyServerType, uid, "Push",
		&push.PushRequest{
			Head: &common.Head{Sid: uuid, Uid: uid},
			Info: &push.PushInfo{PushType: pushtype, Target: target, TermType: term,
				Desc: desc, Content: content}})
	httpserver.CheckRPCErr(rpcerr, "Push")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "Push")

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func delTags(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	tags, err := req.Post.Get("data").Get("ids").Array()
	if err != nil {
		log.Printf("get tags failed:%v", err)
		return &util.AppError{httpserver.ErrInvalidParam, err.Error(), ""}
	}

	var cts []int64
	for i := 0; i < len(tags); i++ {
		tag, _ := tags[i].(json.Number).Int64()
		cts = append(cts, tag)
	}

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ModifyServerType, uid, "DelTags",
		&modify.DelTagRequest{
			Head: &common.Head{Sid: uuid, Uid: uid},
			Ids:  cts})
	httpserver.CheckRPCErr(rpcerr, "DelTags")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "DelTags")

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func delConf(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	keys, err := req.Post.Get("data").Get("keys").Array()
	if err != nil {
		log.Printf("get tags failed:%v", err)
		return &util.AppError{httpserver.ErrInvalidParam, err.Error(), ""}
	}

	var names []string
	for i := 0; i < len(keys); i++ {
		key, _ := keys[i].(string)
		names = append(names, key)
	}

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ModifyServerType, uid, "DelConf",
		&modify.DelConfRequest{
			Head:  &common.Head{Sid: uuid, Uid: uid},
			Names: names})
	httpserver.CheckRPCErr(rpcerr, "DelConf")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "DelConf")

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func delZteAccount(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	phone := req.GetParamString("phone")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ModifyServerType, uid, "DelZteAccount",
		&modify.ZteRequest{
			Head:  &common.Head{Sid: uuid, Uid: uid},
			Phone: phone})
	httpserver.CheckRPCErr(rpcerr, "DelZteAccount")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "DelZteAccount")

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func modTemplate(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	id := req.GetParamInt("id")
	title := req.GetParamStringDef("title", "")
	content := req.GetParamStringDef("content", "")
	online := req.GetParamIntDef("online", 0)

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ModifyServerType, uid, "ModTemplate",
		&modify.ModTempRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Info: &modify.TemplateInfo{Id: id, Title: title, Content: content, Online: online != 0}})
	httpserver.CheckRPCErr(rpcerr, "ModTemplate")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "ModTemplate")

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func modBanner(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	id := req.GetParamInt("id")
	img := req.GetParamStringDef("img", "")
	dst := req.GetParamStringDef("dst", "")
	title := req.GetParamStringDef("title", "")
	online := req.GetParamIntDef("online", 0)
	deleted := req.GetParamIntDef("delete", 0)
	priority := req.GetParamIntDef("priority", 0)
	expire := req.GetParamStringDef("expire", "")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ModifyServerType, uid, "ModBanner",
		&modify.BannerRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Info: &common.BannerInfo{Id: id, Img: img, Dst: dst, Priority: priority,
				Online: online, Deleted: deleted, Title: title, Expire: expire}})
	httpserver.CheckRPCErr(rpcerr, "ModBanner")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "ModBanner")

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func getOssAps(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	return httpserver.GetAps(w, r, true)
}

func getBanners(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	num := req.GetParamInt("num")
	seq := req.GetParamInt("seq")
	btype := req.GetParamInt("type")
	num = genReqNum(num)

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.FetchServerType, uid, "FetchBanners",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Seq: seq, Type: btype, Num: num})
	httpserver.CheckRPCErr(rpcerr, "FetchBanners")
	res := resp.Interface().(*fetch.BannerReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "FetchBanners")

	body := httpserver.GenResponseBody(res, false)
	log.Printf("getBanners body:%s\n", body)
	w.Write(body)
	return nil
}

func getFeedback(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	num := req.GetParamInt("num")
	seq := req.GetParamInt("seq")
	num = genReqNum(num)

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.FetchServerType, uid, "FetchFeedback",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Seq: seq, Num: num})
	httpserver.CheckRPCErr(rpcerr, "FetchFeedback")
	res := resp.Interface().(*fetch.FeedbackReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "FetchFeedback")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	return nil
}

func getPortalDirList(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	num := req.GetParamInt("num")
	seq := req.GetParamInt("seq")
	ptype := req.GetParamIntDef("type", 0)
	num = genReqNum(num)

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.FetchServerType, uid, "FetchPortalDir",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Seq: seq, Num: num, Type: ptype})
	httpserver.CheckRPCErr(rpcerr, "FetchPortalDir")
	res := resp.Interface().(*fetch.PortalDirReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "FetchPortalDir")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	return nil
}

func getChannelVersion(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	num := req.GetParamInt("num")
	seq := req.GetParamInt("seq")
	num = genReqNum(num)

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.FetchServerType, uid, "FetchChannelVersion",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Seq: seq, Num: num})
	httpserver.CheckRPCErr(rpcerr, "FetchChannelVersion")
	res := resp.Interface().(*fetch.ChannelVersionReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "FetchChannelVersion")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	return nil
}

func addChannelVersion(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	channel := req.GetParamString("channel")
	cname := req.GetParamString("cname")
	vname := req.GetParamString("vname")
	version := req.GetParamInt("version")
	downurl := req.GetParamString("downurl")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ModifyServerType, uid, "AddChannelVersion",
		&modify.ChannelVersionRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Info: &common.ChannelVersionInfo{Channel: channel, Cname: cname,
				Version: version, Vname: vname, Downurl: downurl}})
	httpserver.CheckRPCErr(rpcerr, "AddChannelVersion")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "AddChannelVersion")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	return nil
}

func modChannelVersion(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	id := req.GetParamInt("id")
	channel := req.GetParamStringDef("channel", "")
	cname := req.GetParamStringDef("cname", "")
	vname := req.GetParamStringDef("vname", "")
	version := req.GetParamInt("version")
	downurl := req.GetParamStringDef("downurl", "")

	uuid := util.GenUUID()
	resp, rpcerr := httpserver.CallRPC(util.ModifyServerType, uid, "ModChannelVersion",
		&modify.ChannelVersionRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Info: &common.ChannelVersionInfo{Id: id, Channel: channel, Cname: cname,
				Version: version, Vname: vname, Downurl: downurl}})
	httpserver.CheckRPCErr(rpcerr, "ModChannelVersion")
	res := resp.Interface().(*common.CommReply)
	httpserver.CheckRPCCode(res.Head.Retcode, "ModChannelVersion")

	body := httpserver.GenResponseBody(res, false)
	w.Write(body)
	return nil
}

func getOssImagePolicy(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req httpserver.Request
	req.InitCheckOss(r)
	uid := req.GetParamInt("uid")
	formats, err := req.Post.Get("data").Get("formats").Array()
	if err != nil {
		log.Printf("get format failed:%v", err)
		return &util.AppError{httpserver.ErrInvalidParam, err.Error(), ""}
	}

	var names []string
	for i := 0; i < len(formats); i++ {
		format, _ := formats[i].(string)
		fname := util.GenUUID() + "." + format
		names = append(names, fname)
	}
	err = httpserver.AddImages(uid, names)
	if err != nil {
		return &util.AppError{httpserver.ErrInner, err.Error(), ""}
	}

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{httpserver.ErrInner, "invalid param", ""}
	}
	data, _ := simplejson.NewJson([]byte(`{}`))
	aliyun.FillPolicyResp(data)
	data.Set("names", names)
	js.Set("errno", 0)
	js.Set("data", data)

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{httpserver.ErrInner, "marshal json failed", ""}
	}
	w.Write(body)
	return nil
}

//NewOssServer return oss http handler
func NewOssServer() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/login", httpserver.AppHandler(backLogin))
	mux.Handle("/get_news", httpserver.AppHandler(getReviewNews))
	mux.Handle("/get_tags", httpserver.AppHandler(getTags))
	mux.Handle("/get_ap_stat", httpserver.AppHandler(getApStat))
	mux.Handle("/get_users", httpserver.AppHandler(getUsers))
	mux.Handle("/get_templates", httpserver.AppHandler(getTemplates))
	mux.Handle("/get_conf", httpserver.AppHandler(getOssConf))
	mux.Handle("/get_api", httpserver.AppHandler(getApi))
	mux.Handle("/get_batch_api_stat", httpserver.AppHandler(getBatchApiStat))
	mux.Handle("/get_portal_menu", httpserver.AppHandler(fetchPortalMenu))
	mux.Handle("/mod_portal_menu", httpserver.AppHandler(modPortalMenu))
	mux.Handle("/add_portal_menu", httpserver.AppHandler(addPortalMenu))
	mux.Handle("/get_adban", httpserver.AppHandler(getAdBan))
	mux.Handle("/add_adban", httpserver.AppHandler(addAdBan))
	mux.Handle("/del_adban", httpserver.AppHandler(delAdBan))
	mux.Handle("/get_white_list", httpserver.AppHandler(getWhiteList))
	mux.Handle("/add_white_list", httpserver.AppHandler(addWhiteList))
	mux.Handle("/add_portal_dir", httpserver.AppHandler(addPortalDir))
	mux.Handle("/online_portal_dir", httpserver.AppHandler(onlinePortalDir))
	mux.Handle("/del_white_list", httpserver.AppHandler(delWhiteList))
	mux.Handle("/add_template", httpserver.AppHandler(addTemplate))
	mux.Handle("/add_banner", httpserver.AppHandler(addBanner))
	mux.Handle("/set_conf", httpserver.AppHandler(addConf))
	mux.Handle("/add_tags", httpserver.AppHandler(addTags))
	mux.Handle("/add_advertise", httpserver.AppHandler(addAdvertise))
	mux.Handle("/mod_advertise", httpserver.AppHandler(modAdvertise))
	mux.Handle("/get_advertise", httpserver.AppHandler(getAdvertise))
	mux.Handle("/add_unit", httpserver.AppHandler(addUnit))
	mux.Handle("/mod_unit", httpserver.AppHandler(modUnit))
	mux.Handle("/get_unit", httpserver.AppHandler(getUnit))
	mux.Handle("/add_customer", httpserver.AppHandler(addCustomer))
	mux.Handle("/mod_customer", httpserver.AppHandler(modCustomer))
	mux.Handle("/get_customer", httpserver.AppHandler(getCustomer))
	mux.Handle("/send_mipush", httpserver.AppHandler(sendMipush))
	mux.Handle("/del_tags", httpserver.AppHandler(delTags))
	mux.Handle("/del_conf", httpserver.AppHandler(delConf))
	mux.Handle("/del_zte_account", httpserver.AppHandler(delZteAccount))
	mux.Handle("/mod_template", httpserver.AppHandler(modTemplate))
	mux.Handle("/mod_banner", httpserver.AppHandler(modBanner))
	mux.Handle("/get_nearby_aps", httpserver.AppHandler(getOssAps))
	mux.Handle("/review_news", httpserver.AppHandler(reviewNews))
	mux.Handle("/review_video", httpserver.AppHandler(reviewVideo))
	mux.Handle("/get_videos", httpserver.AppHandler(getVideos))
	mux.Handle("/get_banners", httpserver.AppHandler(getBanners))
	mux.Handle("/get_feedback", httpserver.AppHandler(getFeedback))
	mux.Handle("/get_portal_dir", httpserver.AppHandler(getPortalDirList))
	mux.Handle("/get_channel_version", httpserver.AppHandler(getChannelVersion))
	mux.Handle("/add_channel_version", httpserver.AppHandler(addChannelVersion))
	mux.Handle("/mod_channel_version", httpserver.AppHandler(modChannelVersion))
	mux.Handle("/get_oss_image_policy", httpserver.AppHandler(getOssImagePolicy))
	mux.Handle("/", http.FileServer(http.Dir("/data/server/oss")))
	return mux
}
