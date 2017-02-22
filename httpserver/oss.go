package httpserver

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"

	"Server/aliyun"
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
	var req request
	req.init(r.Body, r.RequestURI)
	username := req.GetParamString("username")
	password := req.GetParamString("password")

	uuid := util.GenUUID()
	resp, rpcerr := callRPC(util.VerifyServerType, 0, "BackLogin",
		&verify.LoginRequest{Head: &common.Head{Sid: uuid},
			Username: username, Password: password})
	checkRPCErr(rpcerr, "BackLogin")
	res := resp.Interface().(*verify.LoginReply)
	checkRPCCode(res.Head.Retcode, "BackLogin")

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{errInner, "invalid param"}
	}
	role := strconv.Itoa(int(res.Role))
	initRoleConf()
	js.SetPath([]string{"data", "uid"}, res.Head.Uid)
	js.SetPath([]string{"data", "token"}, res.Token)
	js.SetPath([]string{"data", "roleconf"}, roleConf.Get(role))

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{errInner, "marshal json failed"}
	}

	w.Write(body)
	return nil
}

func getReviewNews(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body, r.RequestURI)
	uid := req.GetParamInt("uid")
	num := req.GetParamInt("num")
	seq := req.GetParamInt("seq")
	ctype := req.GetParamInt("type")
	stype := req.GetParamIntDef("stype", 0)
	search := req.GetParamStringDef("search", "")
	num = genReqNum(num)

	uuid := util.GenUUID()
	resp, rpcerr := callRPC(util.FetchServerType, uid, "FetchReviewNews",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Seq: seq, Num: num, Type: ctype, Subtype: stype, Search: search})
	checkRPCErr(rpcerr, "FetchReviewNews")
	res := resp.Interface().(*fetch.NewsReply)
	checkRPCCode(res.Head.Retcode, "FetchReviewNews")

	body := genResponseBody(res, false)
	w.Write(body)
	return nil
}

func getTags(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body, r.RequestURI)
	uid := req.GetParamInt("uid")
	num := req.GetParamInt("num")
	seq := req.GetParamInt("seq")
	num = genReqNum(num)

	uuid := util.GenUUID()
	resp, rpcerr := callRPC(util.FetchServerType, uid, "FetchTags",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Seq: seq, Num: num})
	checkRPCErr(rpcerr, "FetchTags")
	res := resp.Interface().(*fetch.TagsReply)
	checkRPCCode(res.Head.Retcode, "FetchTags")

	body := genResponseBody(res, false)
	w.Write(body)
	return nil
}

func getUsers(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body, r.RequestURI)
	uid := req.GetParamInt("uid")
	num := req.GetParamInt("num")
	seq := req.GetParamInt("seq")
	num = genReqNum(num)

	uuid := util.GenUUID()
	resp, rpcerr := callRPC(util.FetchServerType, uid, "FetchUsers",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Seq: seq, Num: num})
	checkRPCErr(rpcerr, "FetchUsers")
	res := resp.Interface().(*fetch.UserReply)
	checkRPCCode(res.Head.Retcode, "FetchUsers")

	body := genResponseBody(res, false)
	w.Write(body)
	return nil
}

func reviewVideo(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body, r.RequestURI)
	uid := req.GetParamInt("uid")
	id := req.GetParamInt("id")
	reject := req.GetParamInt("reject")
	mod := req.GetParamIntDef("modify", 0)
	var title string
	if mod != 0 {
		title = req.GetParamStringDef("title", "")
	}

	uuid := util.GenUUID()
	resp, rpcerr := callRPC(util.ModifyServerType, uid, "ReviewVideo",
		&modify.VideoRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Id: id, Reject: reject == 1,
			Modify: mod == 1, Title: title})
	checkRPCErr(rpcerr, "ReviewVideo")
	res := resp.Interface().(*common.CommReply)
	checkRPCCode(res.Head.Retcode, "ReviewVideo")

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func reviewNews(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body, r.RequestURI)
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
	resp, rpcerr := callRPC(util.ModifyServerType, uid, "ReviewNews",
		&modify.NewsRequest{Head: &common.Head{Sid: uuid, Uid: uid}, Id: id, Reject: reject == 1,
			Modify: mod == 1, Title: title, Tags: tags})
	checkRPCErr(rpcerr, "ReviewNews")
	res := resp.Interface().(*common.CommReply)
	checkRPCCode(res.Head.Retcode, "ReviewNews")

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func getApStat(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body, r.RequestURI)
	uid := req.GetParamInt("uid")
	num := req.GetParamInt("num")
	seq := req.GetParamInt("seq")
	num = genReqNum(num)

	uuid := util.GenUUID()
	resp, rpcerr := callRPC(util.FetchServerType, uid, "FetchApStat",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Seq: seq, Num: num})
	checkRPCErr(rpcerr, "FetchApStat")
	res := resp.Interface().(*fetch.ApStatReply)
	checkRPCCode(res.Head.Retcode, "FetchApStat")

	body := genResponseBody(res, false)
	w.Write(body)
	return nil
}

func getVideos(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body, r.RequestURI)
	uid := req.GetParamInt("uid")
	num := req.GetParamInt("num")
	seq := req.GetParamInt("seq")
	ctype := req.GetParamInt("type")
	num = genReqNum(num)

	uuid := util.GenUUID()
	resp, rpcerr := callRPC(util.FetchServerType, uid, "FetchVideos",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Seq: seq, Num: num, Type: ctype})
	checkRPCErr(rpcerr, "FetchVideos")
	res := resp.Interface().(*fetch.VideoReply)
	checkRPCCode(res.Head.Retcode, "FetchVideos")

	body := genResponseBody(res, false)
	w.Write(body)
	return nil
}

func getTemplates(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body, r.RequestURI)
	uid := req.GetParamInt("uid")
	num := req.GetParamInt("num")
	seq := req.GetParamInt("seq")
	num = genReqNum(num)

	uuid := util.GenUUID()
	resp, rpcerr := callRPC(util.FetchServerType, uid, "FetchTemplates",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Seq: seq, Num: num})
	checkRPCErr(rpcerr, "FetchTemplates")
	res := resp.Interface().(*fetch.TemplateReply)
	checkRPCCode(res.Head.Retcode, "FetchTemplates")

	body := genResponseBody(res, false)
	w.Write(body)
	return nil
}

func getOssConf(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	return getConf(w, r, true)
}

func getAdBan(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body, r.RequestURI)
	uid := req.GetParamInt("uid")

	uuid := util.GenUUID()
	resp, rpcerr := callRPC(util.FetchServerType, uid, "FetchAdBan",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid}})
	checkRPCErr(rpcerr, "FetchAdBan")
	res := resp.Interface().(*fetch.AdBanReply)
	checkRPCCode(res.Head.Retcode, "FetchAdBan")

	body := genResponseBody(res, false)
	w.Write(body)
	return nil
}

func getApi(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body, r.RequestURI)
	uid := req.GetParamInt("uid")

	uuid := util.GenUUID()
	resp, rpcerr := callRPC(util.MonitorServerType, uid, "GetApi",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid}})
	checkRPCErr(rpcerr, "GetApi")
	res := resp.Interface().(*monitor.ApiReply)
	checkRPCCode(res.Head.Retcode, "GetApi")

	body := genResponseBody(res, false)
	w.Write(body)
	return nil
}

func fetchPortalMenu(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body, r.RequestURI)
	uid := req.GetParamInt("uid")
	stype := req.GetParamInt("type")

	uuid := util.GenUUID()
	resp, rpcerr := callRPC(util.ConfigServerType, uid, "FetchPortalMenu",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid}, Type: stype})
	checkRPCErr(rpcerr, "FetchPortalMenu")
	res := resp.Interface().(*config.MenuReply)
	checkRPCCode(res.Head.Retcode, "FetchPortalMenu")

	body := genResponseBody(res, false)
	w.Write(body)
	return nil
}

func modPortalMenu(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body, r.RequestURI)
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
	resp, rpcerr := callRPC(util.ConfigServerType, uid, "ModPortalMenu",
		&config.MenuRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Info: &config.PortalMenuInfo{Id: id, Icon: icon, Text: text, Name: name,
				Routername: routername, Url: url, Priority: priority,
				Dbg: dbg, Deleted: deleted}})
	checkRPCErr(rpcerr, "ModPortalMenu")
	res := resp.Interface().(*common.CommReply)
	checkRPCCode(res.Head.Retcode, "ModPortalMenu")

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func getBatchApiStat(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body, r.RequestURI)
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
	resp, rpcerr := callRPC(util.MonitorServerType, uid, "GetBatchApiStat",
		&monitor.BatchApiStatRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Names: names, Num: num})
	checkRPCErr(rpcerr, "GetBatchApiStat")
	res := resp.Interface().(*monitor.BatchApiStatReply)
	checkRPCCode(res.Head.Retcode, "GetBatchApiStat")

	body := genResponseBody(res, false)
	w.Write(body)
	return nil
}

func addTemplate(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body, r.RequestURI)
	uid := req.GetParamInt("uid")
	title := req.GetParamString("title")
	content := req.GetParamString("content")

	uuid := util.GenUUID()
	resp, rpcerr := callRPC(util.ModifyServerType, uid, "AddTemplate",
		&modify.AddTempRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Info: &modify.TemplateInfo{Title: title, Content: content}})
	checkRPCErr(rpcerr, "AddTemplate")
	res := resp.Interface().(*common.CommReply)
	checkRPCCode(res.Head.Retcode, "AddTemplate")

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{errInner, "invalid param"}
	}
	js.SetPath([]string{"data", "tid"}, res.Id)

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{errInner, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func addBanner(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body, r.RequestURI)
	uid := req.GetParamInt("uid")
	img := req.GetParamString("img")
	dst := req.GetParamString("dst")
	priority := req.GetParamInt("priority")
	btype := req.GetParamInt("type")
	title := req.GetParamStringDef("title", "")
	expire := req.GetParamStringDef("expire", "")

	uuid := util.GenUUID()
	resp, rpcerr := callRPC(util.ModifyServerType, uid, "AddBanner",
		&modify.BannerRequest{
			Head: &common.Head{Sid: uuid, Uid: uid},
			Info: &common.BannerInfo{Img: img, Dst: dst, Priority: priority,
				Title: title, Type: btype, Expire: expire}})
	checkRPCErr(rpcerr, "AddBanner")
	res := resp.Interface().(*common.CommReply)
	checkRPCCode(res.Head.Retcode, "AddBanner")

	body := genResponseBody(res, false)
	w.Write(body)
	return nil
}

func addConf(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body, r.RequestURI)
	uid := req.GetParamInt("uid")
	key := req.GetParamString("key")
	val := req.GetParamString("val")

	uuid := util.GenUUID()
	resp, rpcerr := callRPC(util.ModifyServerType, uid, "AddConf",
		&modify.ConfRequest{
			Head: &common.Head{Sid: uuid, Uid: uid},
			Info: &common.KvInfo{Key: key, Val: val}})
	checkRPCErr(rpcerr, "AddConf")
	res := resp.Interface().(*common.CommReply)
	checkRPCCode(res.Head.Retcode, "AddConf")

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func addAdBan(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body, r.RequestURI)
	uid := req.GetParamInt("uid")
	term := req.GetParamInt("term")
	version := req.GetParamInt("version")

	uuid := util.GenUUID()
	resp, rpcerr := callRPC(util.ModifyServerType, uid, "AddAdBan",
		&modify.AddBanRequest{
			Head: &common.Head{Sid: uuid, Uid: uid},
			Info: &common.AdBan{Term: term, Version: version}})
	checkRPCErr(rpcerr, "AddAdBan")
	res := resp.Interface().(*common.CommReply)
	checkRPCCode(res.Head.Retcode, "AddAdBan")

	body := genResponseBody(res, false)
	w.Write(body)
	return nil
}

func delAdBan(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body, r.RequestURI)
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
	resp, rpcerr := callRPC(util.ModifyServerType, uid, "DelAdBan",
		&modify.DelBanRequest{
			Head: &common.Head{Sid: uuid, Uid: uid},
			Ids:  ids})
	checkRPCErr(rpcerr, "DelAdBan")
	res := resp.Interface().(*common.CommReply)
	checkRPCCode(res.Head.Retcode, "DelAdBan")

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func getWhiteList(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body, r.RequestURI)
	uid := req.GetParamInt("uid")
	seq := req.GetParamInt("seq")
	num := req.GetParamInt("num")
	wtype := req.GetParamInt("type")

	uuid := util.GenUUID()
	resp, rpcerr := callRPC(util.FetchServerType, uid, "FetchWhiteList",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Seq: seq, Num: num, Type: wtype})
	checkRPCErr(rpcerr, "FetchWhiteList")
	res := resp.Interface().(*fetch.WhiteReply)
	checkRPCCode(res.Head.Retcode, "FetchWhiteList")

	body := genResponseBody(res, false)
	w.Write(body)
	return nil
}

func addWhiteList(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body, r.RequestURI)
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
	resp, rpcerr := callRPC(util.ModifyServerType, uid, "AddWhiteList",
		&modify.WhiteRequest{
			Head: &common.Head{Sid: uuid, Uid: uid},
			Type: wtype, Ids: ids})
	checkRPCErr(rpcerr, "AddWhiteList")
	res := resp.Interface().(*common.CommReply)
	checkRPCCode(res.Head.Retcode, "AddWhiteList")

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func addPortalDir(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body, r.RequestURI)
	uid := req.GetParamInt("uid")
	ptype := req.GetParamInt("type")
	desc := req.GetParamStringDef("description", "")
	dir := req.GetParamString("dir")

	uuid := util.GenUUID()
	resp, rpcerr := callRPC(util.ModifyServerType, uid, "AddPortalDir",
		&modify.PortalDirRequest{
			Head: &common.Head{Sid: uuid, Uid: uid},
			Info: &common.PortalDirInfo{Type: ptype, Dir: dir, Description: desc}})
	checkRPCErr(rpcerr, "AddPortalDir")
	res := resp.Interface().(*common.CommReply)
	checkRPCCode(res.Head.Retcode, "AddPortalDir")

	body := genResponseBody(res, false)
	w.Write(body)
	return nil
}

func onlinePortalDir(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body, r.RequestURI)
	uid := req.GetParamInt("uid")
	id := req.GetParamInt("id")

	uuid := util.GenUUID()
	resp, rpcerr := callRPC(util.ModifyServerType, uid, "OnlinePortalDir",
		&common.CommRequest{
			Head: &common.Head{Sid: uuid, Uid: uid},
			Id:   id})
	checkRPCErr(rpcerr, "OnlinePortalDir")
	res := resp.Interface().(*common.CommReply)
	checkRPCCode(res.Head.Retcode, "OnlinePortalDir")

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func delWhiteList(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body, r.RequestURI)
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
	resp, rpcerr := callRPC(util.ModifyServerType, uid, "DelWhiteList",
		&modify.WhiteRequest{
			Head: &common.Head{Sid: uuid, Uid: uid},
			Type: wtype, Ids: ids})
	checkRPCErr(rpcerr, "DelWhiteList")
	res := resp.Interface().(*common.CommReply)
	checkRPCCode(res.Head.Retcode, "DelWhiteList")

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func addTags(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body, r.RequestURI)
	uid := req.GetParamInt("uid")
	tags, err := req.Post.Get("data").Get("tags").Array()
	if err != nil {
		log.Printf("get tags failed:%v", err)
		return &util.AppError{2, err.Error()}
	}

	var cts []string
	for i := 0; i < len(tags); i++ {
		tag := tags[i].(string)
		cts = append(cts, tag)
	}

	uuid := util.GenUUID()
	resp, rpcerr := callRPC(util.ModifyServerType, uid, "AddTags",
		&modify.AddTagRequest{
			Head: &common.Head{Sid: uuid, Uid: uid},
			Tags: cts})
	checkRPCErr(rpcerr, "AddTags")
	res := resp.Interface().(*common.CommReply)
	checkRPCCode(res.Head.Retcode, "AddTags")

	body := genResponseBody(res, false)
	w.Write(body)
	return nil
}

func sendMipush(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body, r.RequestURI)
	uid := req.GetParamInt("uid")
	desc := req.GetParamString("desc")
	content := req.GetParamString("content")
	target := req.GetParamString("target")
	term := req.GetParamInt("term")
	pushtype := req.GetParamInt("pushtype")

	uuid := util.GenUUID()
	resp, rpcerr := callRPC(util.ModifyServerType, uid, "Push",
		&push.PushRequest{
			Head: &common.Head{Sid: uuid, Uid: uid},
			Info: &push.PushInfo{PushType: pushtype, Target: target, TermType: term,
				Desc: desc, Content: content}})
	checkRPCErr(rpcerr, "Push")
	res := resp.Interface().(*common.CommReply)
	checkRPCCode(res.Head.Retcode, "Push")

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func delTags(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body, r.RequestURI)
	uid := req.GetParamInt("uid")
	tags, err := req.Post.Get("data").Get("ids").Array()
	if err != nil {
		log.Printf("get tags failed:%v", err)
		return &util.AppError{2, err.Error()}
	}

	var cts []int64
	for i := 0; i < len(tags); i++ {
		tag, _ := tags[i].(json.Number).Int64()
		cts = append(cts, tag)
	}

	uuid := util.GenUUID()
	resp, rpcerr := callRPC(util.ModifyServerType, uid, "DelTags",
		&modify.DelTagRequest{
			Head: &common.Head{Sid: uuid, Uid: uid},
			Ids:  cts})
	checkRPCErr(rpcerr, "DelTags")
	res := resp.Interface().(*common.CommReply)
	checkRPCCode(res.Head.Retcode, "DelTags")

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func delConf(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body, r.RequestURI)
	uid := req.GetParamInt("uid")
	keys, err := req.Post.Get("data").Get("keys").Array()
	if err != nil {
		log.Printf("get tags failed:%v", err)
		return &util.AppError{2, err.Error()}
	}

	var names []string
	for i := 0; i < len(keys); i++ {
		key, _ := keys[i].(string)
		names = append(names, key)
	}

	uuid := util.GenUUID()
	resp, rpcerr := callRPC(util.ModifyServerType, uid, "DelConf",
		&modify.DelConfRequest{
			Head:  &common.Head{Sid: uuid, Uid: uid},
			Names: names})
	checkRPCErr(rpcerr, "DelConf")
	res := resp.Interface().(*common.CommReply)
	checkRPCCode(res.Head.Retcode, "DelConf")

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func delZteAccount(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body, r.RequestURI)
	uid := req.GetParamInt("uid")
	phone := req.GetParamString("phone")

	uuid := util.GenUUID()
	resp, rpcerr := callRPC(util.ModifyServerType, uid, "DelZteAccount",
		&modify.ZteRequest{
			Head:  &common.Head{Sid: uuid, Uid: uid},
			Phone: phone})
	checkRPCErr(rpcerr, "DelZteAccount")
	res := resp.Interface().(*common.CommReply)
	checkRPCCode(res.Head.Retcode, "DelZteAccount")

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func modTemplate(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body, r.RequestURI)
	uid := req.GetParamInt("uid")
	id := req.GetParamInt("id")
	title := req.GetParamStringDef("title", "")
	content := req.GetParamStringDef("content", "")
	online := req.GetParamIntDef("online", 0)

	uuid := util.GenUUID()
	resp, rpcerr := callRPC(util.ModifyServerType, uid, "ModTemplate",
		&modify.ModTempRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Info: &modify.TemplateInfo{Id: id, Title: title, Content: content, Online: online != 0}})
	checkRPCErr(rpcerr, "ModTemplate")
	res := resp.Interface().(*common.CommReply)
	checkRPCCode(res.Head.Retcode, "ModTemplate")

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func modBanner(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body, r.RequestURI)
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
	resp, rpcerr := callRPC(util.ModifyServerType, uid, "ModBanner",
		&modify.BannerRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Info: &common.BannerInfo{Id: id, Img: img, Dst: dst, Priority: priority,
				Online: online, Deleted: deleted, Title: title, Expire: expire}})
	checkRPCErr(rpcerr, "ModBanner")
	res := resp.Interface().(*common.CommReply)
	checkRPCCode(res.Head.Retcode, "ModBanner")

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func getOssAps(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	return getAps(w, r, true)
}

func getBanners(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body, r.RequestURI)
	uid := req.GetParamInt("uid")
	num := req.GetParamInt("num")
	seq := req.GetParamInt("seq")
	btype := req.GetParamInt("type")
	num = genReqNum(num)

	uuid := util.GenUUID()
	resp, rpcerr := callRPC(util.FetchServerType, uid, "FetchBanners",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Seq: seq, Type: btype, Num: num})
	checkRPCErr(rpcerr, "FetchBanners")
	res := resp.Interface().(*fetch.BannerReply)
	checkRPCCode(res.Head.Retcode, "FetchBanners")

	body := genResponseBody(res, false)
	log.Printf("getBanners body:%s\n", body)
	w.Write(body)
	return nil
}

func getFeedback(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body, r.RequestURI)
	uid := req.GetParamInt("uid")
	num := req.GetParamInt("num")
	seq := req.GetParamInt("seq")
	num = genReqNum(num)

	uuid := util.GenUUID()
	resp, rpcerr := callRPC(util.FetchServerType, uid, "FetchFeedback",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Seq: seq, Num: num})
	checkRPCErr(rpcerr, "FetchFeedback")
	res := resp.Interface().(*fetch.FeedbackReply)
	checkRPCCode(res.Head.Retcode, "FetchFeedback")

	body := genResponseBody(res, false)
	w.Write(body)
	return nil
}

func getPortalDirList(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body, r.RequestURI)
	uid := req.GetParamInt("uid")
	num := req.GetParamInt("num")
	seq := req.GetParamInt("seq")
	ptype := req.GetParamIntDef("type", 0)
	num = genReqNum(num)

	uuid := util.GenUUID()
	resp, rpcerr := callRPC(util.FetchServerType, uid, "FetchPortalDir",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Seq: seq, Num: num, Type: ptype})
	checkRPCErr(rpcerr, "FetchPortalDir")
	res := resp.Interface().(*fetch.PortalDirReply)
	checkRPCCode(res.Head.Retcode, "FetchPortalDir")

	body := genResponseBody(res, false)
	w.Write(body)
	return nil
}

func getChannelVersion(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body, r.RequestURI)
	uid := req.GetParamInt("uid")
	num := req.GetParamInt("num")
	seq := req.GetParamInt("seq")
	num = genReqNum(num)

	uuid := util.GenUUID()
	resp, rpcerr := callRPC(util.FetchServerType, uid, "FetchChannelVersion",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Seq: seq, Num: num})
	checkRPCErr(rpcerr, "FetchChannelVersion")
	res := resp.Interface().(*fetch.ChannelVersionReply)
	checkRPCCode(res.Head.Retcode, "FetchChannelVersion")

	body := genResponseBody(res, false)
	w.Write(body)
	return nil
}

func addChannelVersion(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body, r.RequestURI)
	uid := req.GetParamInt("uid")
	channel := req.GetParamString("channel")
	cname := req.GetParamString("cname")
	vname := req.GetParamString("vname")
	version := req.GetParamInt("version")
	downurl := req.GetParamString("downurl")

	uuid := util.GenUUID()
	resp, rpcerr := callRPC(util.ModifyServerType, uid, "AddChannelVersion",
		&modify.ChannelVersionRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Info: &common.ChannelVersionInfo{Channel: channel, Cname: cname,
				Version: version, Vname: vname, Downurl: downurl}})
	checkRPCErr(rpcerr, "AddChannelVersion")
	res := resp.Interface().(*common.CommReply)
	checkRPCCode(res.Head.Retcode, "AddChannelVersion")

	body := genResponseBody(res, false)
	w.Write(body)
	return nil
}

func modChannelVersion(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body, r.RequestURI)
	uid := req.GetParamInt("uid")
	id := req.GetParamInt("id")
	channel := req.GetParamStringDef("channel", "")
	cname := req.GetParamStringDef("cname", "")
	vname := req.GetParamStringDef("vname", "")
	version := req.GetParamInt("version")
	downurl := req.GetParamStringDef("downurl", "")

	uuid := util.GenUUID()
	resp, rpcerr := callRPC(util.ModifyServerType, uid, "ModChannelVersion",
		&modify.ChannelVersionRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Info: &common.ChannelVersionInfo{Id: id, Channel: channel, Cname: cname,
				Version: version, Vname: vname, Downurl: downurl}})
	checkRPCErr(rpcerr, "ModChannelVersion")
	res := resp.Interface().(*common.CommReply)
	checkRPCCode(res.Head.Retcode, "ModChannelVersion")

	body := genResponseBody(res, false)
	w.Write(body)
	return nil
}

func getOssImagePolicy(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body, r.RequestURI)
	uid := req.GetParamInt("uid")
	formats, err := req.Post.Get("data").Get("formats").Array()
	if err != nil {
		log.Printf("get format failed:%v", err)
		return &util.AppError{2, err.Error()}
	}

	var names []string
	for i := 0; i < len(formats); i++ {
		format, _ := formats[i].(string)
		fname := util.GenUUID() + "." + format
		names = append(names, fname)
	}
	err = addImages(uid, names)
	if err != nil {
		return &util.AppError{errInner, err.Error()}
	}

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{errInner, "invalid param"}
	}
	data, _ := simplejson.NewJson([]byte(`{}`))
	aliyun.FillPolicyResp(data)
	data.Set("names", names)
	js.Set("errno", 0)
	js.Set("data", data)

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{errInner, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

//NewOssServer return oss http handler
func NewOssServer() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/login", appHandler(backLogin))
	mux.Handle("/get_news", appHandler(getReviewNews))
	mux.Handle("/get_tags", appHandler(getTags))
	mux.Handle("/get_ap_stat", appHandler(getApStat))
	mux.Handle("/get_users", appHandler(getUsers))
	mux.Handle("/get_templates", appHandler(getTemplates))
	mux.Handle("/get_conf", appHandler(getOssConf))
	mux.Handle("/get_api", appHandler(getApi))
	mux.Handle("/get_batch_api_stat", appHandler(getBatchApiStat))
	mux.Handle("/get_portal_menu", appHandler(fetchPortalMenu))
	mux.Handle("/mod_portal_menu", appHandler(modPortalMenu))
	mux.Handle("/get_adban", appHandler(getAdBan))
	mux.Handle("/add_adban", appHandler(addAdBan))
	mux.Handle("/del_adban", appHandler(delAdBan))
	mux.Handle("/get_white_list", appHandler(getWhiteList))
	mux.Handle("/add_white_list", appHandler(addWhiteList))
	mux.Handle("/add_portal_dir", appHandler(addPortalDir))
	mux.Handle("/online_portal_dir", appHandler(onlinePortalDir))
	mux.Handle("/del_white_list", appHandler(delWhiteList))
	mux.Handle("/add_template", appHandler(addTemplate))
	mux.Handle("/add_banner", appHandler(addBanner))
	mux.Handle("/set_conf", appHandler(addConf))
	mux.Handle("/add_tags", appHandler(addTags))
	mux.Handle("/send_mipush", appHandler(sendMipush))
	mux.Handle("/del_tags", appHandler(delTags))
	mux.Handle("/del_conf", appHandler(delConf))
	mux.Handle("/del_zte_account", appHandler(delZteAccount))
	mux.Handle("/mod_template", appHandler(modTemplate))
	mux.Handle("/mod_banner", appHandler(modBanner))
	mux.Handle("/get_nearby_aps", appHandler(getOssAps))
	mux.Handle("/review_news", appHandler(reviewNews))
	mux.Handle("/review_video", appHandler(reviewVideo))
	mux.Handle("/get_videos", appHandler(getVideos))
	mux.Handle("/get_banners", appHandler(getBanners))
	mux.Handle("/get_feedback", appHandler(getFeedback))
	mux.Handle("/get_portal_dir", appHandler(getPortalDirList))
	mux.Handle("/get_channel_version", appHandler(getChannelVersion))
	mux.Handle("/add_channel_version", appHandler(addChannelVersion))
	mux.Handle("/mod_channel_version", appHandler(modChannelVersion))
	mux.Handle("/get_oss_image_policy", appHandler(getOssImagePolicy))
	mux.Handle("/", http.FileServer(http.Dir("/data/server/oss")))
	return mux
}
