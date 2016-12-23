package httpserver

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	aliyun "../aliyun"
	common "../proto/common"
	fetch "../proto/fetch"
	modify "../proto/modify"
	push "../proto/push"
	verify "../proto/verify"
	util "../util"
	simplejson "github.com/bitly/go-simplejson"
	"google.golang.org/grpc"
)

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
	req.init(r.Body)
	username := req.GetParamString("username")
	password := req.GetParamString("password")

	address := getNameServer(0, util.VerifyServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := verify.NewVerifyClient(conn)

	uuid := util.GenUUID()
	res, err := c.BackLogin(context.Background(), &verify.LoginRequest{Head: &common.Head{Sid: uuid}, Username: username, Password: password})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}

	if res.Head.Retcode != 0 {
		return &util.AppError{util.LogicErr, int(res.Head.Retcode), "登录失败"}
	}

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{util.JSONErr, 4, err.Error()}
	}

	js.SetPath([]string{"data", "uid"}, res.Head.Uid)
	js.SetPath([]string{"data", "token"}, res.Token)
	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{util.JSONErr, 4, err.Error()}
	}
	w.Write(body)
	return nil
}

func getReviewNews(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body)
	uid := req.GetParamInt("uid")
	num := req.GetParamInt("num")
	seq := req.GetParamInt("seq")
	ctype := req.GetParamInt("type")
	num = genReqNum(num)

	address := getNameServer(uid, util.FetchServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := fetch.NewFetchClient(conn)

	uuid := util.GenUUID()
	res, err := c.FetchReviewNews(context.Background(),
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Seq: seq, Num: int32(num), Type: int32(ctype)})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "获取新闻失败"}
	}

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "invalid param"}
	}
	js.SetPath([]string{"data", "news"}, res.Infos)
	js.SetPath([]string{"data", "total"}, res.Total)

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func getTags(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body)
	uid := req.GetParamInt("uid")
	num := req.GetParamInt("num")
	seq := req.GetParamInt("seq")
	num = genReqNum(num)

	address := getNameServer(uid, util.FetchServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := fetch.NewFetchClient(conn)

	uuid := util.GenUUID()
	res, err := c.FetchTags(context.Background(),
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Seq: seq, Num: int32(num)})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "获取标签失败"}
	}

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "invalid param"}
	}
	js.SetPath([]string{"data", "tags"}, res.Infos)
	js.SetPath([]string{"data", "total"}, res.Total)

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func getUsers(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body)
	uid := req.GetParamInt("uid")
	num := req.GetParamInt("num")
	seq := req.GetParamInt("seq")
	num = genReqNum(num)

	address := getNameServer(uid, util.FetchServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := fetch.NewFetchClient(conn)

	uuid := util.GenUUID()
	res, err := c.FetchUsers(context.Background(),
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Seq: seq, Num: int32(num)})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "获取标签失败"}
	}

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "invalid param"}
	}
	js.SetPath([]string{"data", "infos"}, res.Infos)
	js.SetPath([]string{"data", "total"}, res.Total)

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func reviewVideo(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body)
	uid := req.GetParamInt("uid")
	id := req.GetParamInt("id")
	reject := req.GetParamInt("reject")
	mod := req.GetParamIntDef("modify", 0)
	var title string
	if mod != 0 {
		title = req.GetParamStringDef("title", "")
	}

	address := getNameServer(uid, util.ModifyServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := modify.NewModifyClient(conn)
	uuid := util.GenUUID()
	res, err := c.ReviewVideo(context.Background(), &modify.VideoRequest{Head: &common.Head{Sid: uuid, Uid: uid}, Id: id, Reject: reject == 1,
		Modify: mod == 1, Title: title})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "视频审核失败"}
	}

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func reviewNews(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body)
	uid := req.GetParamInt("uid")
	id := req.GetParamInt("id")
	reject := req.GetParamInt("reject")
	mod := req.GetParamIntDef("modify", 0)
	var title string
	var tags []int32

	if mod != 0 {
		title = req.GetParamStringDef("title", "")
	}

	arr, err := req.Post.Get("data").Get("tags").Array()
	if err == nil {
		for i := 0; i < len(arr); i++ {
			tid, _ := req.Post.Get("data").Get("tags").GetIndex(i).Int()
			tags = append(tags, int32(tid))
		}
	}

	address := getNameServer(uid, util.ModifyServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := modify.NewModifyClient(conn)
	uuid := util.GenUUID()
	res, err := c.ReviewNews(context.Background(), &modify.NewsRequest{Head: &common.Head{Sid: uuid, Uid: uid}, Id: id, Reject: reject == 1,
		Modify: mod == 1, Title: title, Tags: tags})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "获取标签失败"}
	}

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func getApStat(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body)
	uid := req.GetParamInt("uid")
	num := req.GetParamInt("num")
	seq := req.GetParamInt("seq")
	num = genReqNum(num)

	address := getNameServer(uid, util.FetchServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := fetch.NewFetchClient(conn)

	uuid := util.GenUUID()
	res, err := c.FetchApStat(context.Background(),
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Seq: seq, Num: int32(num)})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "获取AP监控信息失败"}
	}

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "invalid param"}
	}
	js.SetPath([]string{"data", "infos"}, res.Infos)
	js.SetPath([]string{"data", "total"}, res.Total)

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func getVideos(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body)
	uid := req.GetParamInt("uid")
	num := req.GetParamInt("num")
	seq := req.GetParamInt("seq")
	ctype := req.GetParamInt("type")
	num = genReqNum(num)

	address := getNameServer(uid, util.FetchServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := fetch.NewFetchClient(conn)

	uuid := util.GenUUID()
	res, err := c.FetchVideos(context.Background(),
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Seq: seq, Num: int32(num), Type: int32(ctype)})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "获取视频审核信息失败"}
	}

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "invalid param"}
	}
	js.SetPath([]string{"data", "infos"}, res.Infos)
	js.SetPath([]string{"data", "total"}, res.Total)

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func getTemplates(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body)
	uid := req.GetParamInt("uid")
	num := req.GetParamInt("num")
	seq := req.GetParamInt("seq")
	num = genReqNum(num)

	address := getNameServer(uid, util.FetchServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := fetch.NewFetchClient(conn)

	uuid := util.GenUUID()
	res, err := c.FetchTemplates(context.Background(),
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Seq: seq, Num: int32(num)})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "获取AP监控信息失败"}
	}

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "invalid param"}
	}
	js.SetPath([]string{"data", "infos"}, res.Infos)
	js.SetPath([]string{"data", "total"}, res.Total)

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func getConf(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body)
	uid := req.GetParamInt("uid")

	address := getNameServer(uid, util.FetchServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := fetch.NewFetchClient(conn)

	uuid := util.GenUUID()
	res, err := c.FetchConf(context.Background(),
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid}})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "获取配置信息失败"}
	}

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "invalid param"}
	}
	js.SetPath([]string{"data", "infos"}, res.Infos)

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func getAdBan(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body)
	uid := req.GetParamInt("uid")

	address := getNameServer(uid, util.FetchServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := fetch.NewFetchClient(conn)

	uuid := util.GenUUID()
	res, err := c.FetchAdBan(context.Background(),
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid}})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "获取广告屏蔽信息失败"}
	}

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "invalid param"}
	}
	js.SetPath([]string{"data", "infos"}, res.Infos)

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func addTemplate(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body)
	uid := req.GetParamInt("uid")
	title := req.GetParamString("title")
	content := req.GetParamString("content")

	address := getNameServer(uid, util.ModifyServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := modify.NewModifyClient(conn)

	uuid := util.GenUUID()
	res, err := c.AddTemplate(context.Background(), &modify.AddTempRequest{Head: &common.Head{Sid: uuid, Uid: uid}, Info: &modify.TemplateInfo{Title: title, Content: content}})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "添加模板失败"}
	}

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "invalid param"}
	}
	js.SetPath([]string{"data", "tid"}, res.Id)

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func addBanner(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body)
	uid := req.GetParamInt("uid")
	img := req.GetParamString("img")
	dst := req.GetParamString("dst")
	priority := req.GetParamInt("priority")
	btype := req.GetParamInt("type")
	title := req.GetParamStringDef("title", "")

	address := getNameServer(uid, util.ModifyServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := modify.NewModifyClient(conn)

	uuid := util.GenUUID()
	res, err := c.AddBanner(context.Background(),
		&modify.BannerRequest{
			Head: &common.Head{Sid: uuid, Uid: uid},
			Info: &common.BannerInfo{Img: img, Dst: dst, Priority: int32(priority),
				Title: title, Type: int32(btype)}})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "添加Banner失败"}
	}

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "invalid param"}
	}
	js.SetPath([]string{"data", "id"}, res.Id)

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func addConf(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body)
	uid := req.GetParamInt("uid")
	key := req.GetParamString("key")
	val := req.GetParamString("val")

	address := getNameServer(uid, util.ModifyServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := modify.NewModifyClient(conn)

	uuid := util.GenUUID()
	res, err := c.AddConf(context.Background(),
		&modify.ConfRequest{
			Head: &common.Head{Sid: uuid, Uid: uid},
			Info: &common.KvInfo{Key: key, Val: val}})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "添加配置失败"}
	}

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func addAdBan(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body)
	uid := req.GetParamInt("uid")
	term := req.GetParamInt("term")
	version := req.GetParamInt("version")

	address := getNameServer(uid, util.ModifyServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := modify.NewModifyClient(conn)

	uuid := util.GenUUID()
	res, err := c.AddAdBan(context.Background(),
		&modify.AddBanRequest{
			Head: &common.Head{Sid: uuid, Uid: uid},
			Info: &common.AdBan{Term: term, Version: version}})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "添加广告屏蔽失败"}
	}

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "invalid param"}
	}
	js.SetPath([]string{"data", "id"}, res.Id)

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func delAdBan(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body)
	uid := req.GetParamInt("uid")

	var ids []int64
	arr, err := req.Post.Get("data").Get("ids").Array()
	if err == nil {
		for i := 0; i < len(arr); i++ {
			tid, _ := req.Post.Get("data").Get("ids").GetIndex(i).Int64()
			ids = append(ids, tid)
		}
	}

	address := getNameServer(uid, util.ModifyServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := modify.NewModifyClient(conn)

	uuid := util.GenUUID()
	res, err := c.DelAdBan(context.Background(),
		&modify.DelBanRequest{
			Head: &common.Head{Sid: uuid, Uid: uid},
			Ids:  ids})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "删除广告屏蔽失败"}
	}

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func getWhiteList(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body)
	uid := req.GetParamInt("uid")
	seq := req.GetParamInt("seq")
	num := req.GetParamInt("num")
	wtype := req.GetParamInt("type")

	address := getNameServer(uid, util.FetchServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := fetch.NewFetchClient(conn)

	uuid := util.GenUUID()
	res, err := c.FetchWhiteList(context.Background(),
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Seq: seq, Num: int32(num), Type: int32(wtype)})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "获取白名单失败"}
	}

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "invalid param"}
	}
	js.SetPath([]string{"data", "infos"}, res.Infos)
	js.SetPath([]string{"data", "total"}, res.Total)

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func addWhiteList(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body)
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

	address := getNameServer(uid, util.ModifyServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := modify.NewModifyClient(conn)

	uuid := util.GenUUID()
	res, err := c.AddWhiteList(context.Background(),
		&modify.WhiteRequest{
			Head: &common.Head{Sid: uuid, Uid: uid},
			Type: wtype, Ids: ids})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "添加广告白名单失败"}
	}

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "invalid param"}
	}

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func delWhiteList(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body)
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

	address := getNameServer(uid, util.ModifyServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := modify.NewModifyClient(conn)

	uuid := util.GenUUID()
	res, err := c.DelWhiteList(context.Background(),
		&modify.WhiteRequest{
			Head: &common.Head{Sid: uuid, Uid: uid},
			Type: wtype, Ids: ids})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "删除广告白名单失败"}
	}

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "invalid param"}
	}

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func addTags(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body)
	uid := req.GetParamInt("uid")
	tags, err := req.Post.Get("data").Get("tags").Array()
	if err != nil {
		log.Printf("get tags failed:%v", err)
		return &util.AppError{util.JSONErr, 2, err.Error()}
	}

	var cts []string
	for i := 0; i < len(tags); i++ {
		tag := tags[i].(string)
		cts = append(cts, tag)
	}

	address := getNameServer(uid, util.ModifyServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := modify.NewModifyClient(conn)

	uuid := util.GenUUID()
	res, err := c.AddTags(context.Background(),
		&modify.AddTagRequest{
			Head: &common.Head{Sid: uuid, Uid: uid},
			Tags: cts})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "添加Banner失败"}
	}

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "invalid param"}
	}
	js.SetPath([]string{"data", "ids"}, res.Ids)

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func sendMipush(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body)
	uid := req.GetParamInt("uid")
	desc := req.GetParamString("desc")
	content := req.GetParamString("content")
	target := req.GetParamString("target")
	term := req.GetParamInt("term")
	pushtype := req.GetParamInt("pushtype")

	address := getNameServer(uid, util.PushServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := push.NewPushClient(conn)

	uuid := util.GenUUID()
	res, err := c.Push(context.Background(),
		&push.PushRequest{
			Head: &common.Head{Sid: uuid, Uid: uid},
			Info: &push.PushInfo{PushType: int32(pushtype), Target: target, TermType: int32(term),
				Desc: desc, Content: content}})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "发送push消息失败"}
	}

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func delTags(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body)
	uid := req.GetParamInt("uid")
	tags, err := req.Post.Get("data").Get("ids").Array()
	if err != nil {
		log.Printf("get tags failed:%v", err)
		return &util.AppError{util.JSONErr, 2, err.Error()}
	}

	var cts []int64
	for i := 0; i < len(tags); i++ {
		tag, _ := tags[i].(json.Number).Int64()
		cts = append(cts, tag)
	}

	address := getNameServer(uid, util.ModifyServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := modify.NewModifyClient(conn)

	uuid := util.GenUUID()
	res, err := c.DelTags(context.Background(),
		&modify.DelTagRequest{
			Head: &common.Head{Sid: uuid, Uid: uid},
			Ids:  cts})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "删除标签失败"}
	}

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func delConf(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body)
	uid := req.GetParamInt("uid")
	keys, err := req.Post.Get("data").Get("keys").Array()
	if err != nil {
		log.Printf("get tags failed:%v", err)
		return &util.AppError{util.JSONErr, 2, err.Error()}
	}

	var names []string
	for i := 0; i < len(keys); i++ {
		key, _ := keys[i].(string)
		names = append(names, key)
	}

	address := getNameServer(uid, util.ModifyServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := modify.NewModifyClient(conn)

	uuid := util.GenUUID()
	res, err := c.DelConf(context.Background(),
		&modify.DelConfRequest{
			Head:  &common.Head{Sid: uuid, Uid: uid},
			Names: names})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "删除配置失败"}
	}

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func modTemplate(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body)
	uid := req.GetParamInt("uid")
	id := req.GetParamInt("id")
	title := req.GetParamStringDef("title", "")
	content := req.GetParamStringDef("content", "")
	online := req.GetParamIntDef("online", 0)

	address := getNameServer(uid, util.ModifyServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := modify.NewModifyClient(conn)

	uuid := util.GenUUID()
	res, err := c.ModTemplate(context.Background(), &modify.ModTempRequest{Head: &common.Head{Sid: uuid, Uid: uid}, Info: &modify.TemplateInfo{Id: int32(id), Title: title, Content: content, Online: online != 0}})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "修改模板失败"}
	}

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func modBanner(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body)
	uid := req.GetParamInt("uid")
	id := req.GetParamInt("id")
	img := req.GetParamStringDef("img", "")
	dst := req.GetParamStringDef("dst", "")
	title := req.GetParamStringDef("title", "")
	online := req.GetParamIntDef("online", 0)
	deleted := req.GetParamIntDef("delete", 0)
	priority := req.GetParamIntDef("priority", 0)
	expire := req.GetParamStringDef("expire", "")

	address := getNameServer(uid, util.ModifyServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := modify.NewModifyClient(conn)

	uuid := util.GenUUID()
	res, err := c.ModBanner(context.Background(),
		&modify.BannerRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Info: &common.BannerInfo{Id: id, Img: img, Dst: dst, Priority: int32(priority),
				Online: int32(online), Deleted: int32(deleted), Title: title, Expire: expire}})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "修改Banner失败"}
	}

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func getOssAps(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	return getAps(w, r, true)
}

func getBanners(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body)
	uid := req.GetParamInt("uid")
	num := req.GetParamInt("num")
	seq := req.GetParamInt("seq")
	btype := req.GetParamInt("type")
	num = genReqNum(num)

	address := getNameServer(uid, util.FetchServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := fetch.NewFetchClient(conn)

	uuid := util.GenUUID()
	res, err := c.FetchBanners(context.Background(),
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Seq: seq, Type: int32(btype), Num: int32(num)})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "获取Banner信息失败"}
	}

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "invalid param"}
	}
	js.SetPath([]string{"data", "infos"}, res.Infos)
	js.SetPath([]string{"data", "total"}, res.Total)

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	log.Printf("getBanners body:%s\n", body)
	w.Write(body)
	return nil
}

func getFeedback(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body)
	uid := req.GetParamInt("uid")
	num := req.GetParamInt("num")
	seq := req.GetParamInt("seq")
	num = genReqNum(num)

	address := getNameServer(uid, util.FetchServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := fetch.NewFetchClient(conn)

	uuid := util.GenUUID()
	res, err := c.FetchFeedback(context.Background(),
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Seq: seq, Num: int32(num)})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "获取用户反馈信息失败"}
	}

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "invalid param"}
	}
	js.SetPath([]string{"data", "infos"}, res.Infos)
	js.SetPath([]string{"data", "total"}, res.Total)

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func getOssImagePolicy(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	var req request
	req.initCheckOss(r.Body)
	uid := req.GetParamInt("uid")
	formats, err := req.Post.Get("data").Get("formats").Array()
	if err != nil {
		log.Printf("get format failed:%v", err)
		return &util.AppError{util.RPCErr, 2, err.Error()}
	}

	var names []string
	for i := 0; i < len(formats); i++ {
		format, _ := formats[i].(string)
		fname := util.GenUUID() + "." + format
		names = append(names, fname)
	}
	err = addImages(uid, names)
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "invalid param"}
	}
	data, _ := simplejson.NewJson([]byte(`{}`))
	aliyun.FillPolicyResp(data)
	data.Set("names", names)
	js.Set("errno", 0)
	js.Set("data", data)

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
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
	mux.Handle("/get_conf", appHandler(getConf))
	mux.Handle("/get_adban", appHandler(getAdBan))
	mux.Handle("/add_adban", appHandler(addAdBan))
	mux.Handle("/del_adban", appHandler(delAdBan))
	mux.Handle("/get_white_list", appHandler(getWhiteList))
	mux.Handle("/add_white_list", appHandler(addWhiteList))
	mux.Handle("/del_white_list", appHandler(delWhiteList))
	mux.Handle("/add_template", appHandler(addTemplate))
	mux.Handle("/add_banner", appHandler(addBanner))
	mux.Handle("/set_conf", appHandler(addConf))
	mux.Handle("/add_tags", appHandler(addTags))
	mux.Handle("/send_mipush", appHandler(sendMipush))
	mux.Handle("/del_tags", appHandler(delTags))
	mux.Handle("/del_conf", appHandler(delConf))
	mux.Handle("/mod_template", appHandler(modTemplate))
	mux.Handle("/mod_banner", appHandler(modBanner))
	mux.Handle("/get_nearby_aps", appHandler(getOssAps))
	mux.Handle("/review_news", appHandler(reviewNews))
	mux.Handle("/review_video", appHandler(reviewVideo))
	mux.Handle("/get_videos", appHandler(getVideos))
	mux.Handle("/get_banners", appHandler(getBanners))
	mux.Handle("/get_feedback", appHandler(getFeedback))
	mux.Handle("/get_oss_image_policy", appHandler(getOssImagePolicy))
	mux.Handle("/", http.FileServer(http.Dir("/data/server/oss")))
	return mux
}
