package httpserver

import (
	"context"
	"net/http"

	common "../proto/common"
	fetch "../proto/fetch"
	modify "../proto/modify"
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
	defer func() {
		if r := recover(); r != nil {
			apperr = extractError(r)
		}
	}()
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
	defer func() {
		if r := recover(); r != nil {
			apperr = extractError(r)
		}
	}()
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
	res, err := c.FetchReviewNews(context.Background(), &fetch.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid}, Seq: seq, Num: int32(num), Type: int32(ctype)})
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
	infos := make([]interface{}, len(res.Infos))
	for i := 0; i < len(res.Infos); i++ {
		json, _ := simplejson.NewJson([]byte(`{}`))
		json.Set("id", res.Infos[i].Id)
		json.Set("title", res.Infos[i].Title)
		json.Set("ctime", res.Infos[i].Ctime)
		json.Set("source", res.Infos[i].Source)
		json.Set("tag", res.Infos[i].Tag)
		infos[i] = json
	}
	js.SetPath([]string{"data", "news"}, infos)
	js.SetPath([]string{"data", "total"}, res.Total)

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func getTags(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	defer func() {
		if r := recover(); r != nil {
			apperr = extractError(r)
		}
	}()

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
	res, err := c.FetchTags(context.Background(), &fetch.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid}, Seq: seq, Num: int32(num)})
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
	infos := make([]interface{}, len(res.Infos))
	for i := 0; i < len(res.Infos); i++ {
		json, _ := simplejson.NewJson([]byte(`{}`))
		json.Set("id", res.Infos[i].Id)
		json.Set("content", res.Infos[i].Content)
		infos[i] = json
	}
	js.SetPath([]string{"data", "tags"}, infos)
	js.SetPath([]string{"data", "total"}, res.Total)

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func getUsers(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	defer func() {
		if r := recover(); r != nil {
			apperr = extractError(r)
		}
	}()
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
	res, err := c.FetchUsers(context.Background(), &fetch.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid}, Seq: seq, Num: int32(num)})
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
	infos := make([]interface{}, len(res.Infos))
	for i := 0; i < len(res.Infos); i++ {
		json, _ := simplejson.NewJson([]byte(`{}`))
		json.Set("id", res.Infos[i].Id)
		json.Set("imei", res.Infos[i].Imei)
		json.Set("phone", res.Infos[i].Phone)
		json.Set("active", res.Infos[i].Active)
		json.Set("remark", res.Infos[i].Remark)
		infos[i] = json
	}
	js.SetPath([]string{"data", "infos"}, infos)
	js.SetPath([]string{"data", "total"}, res.Total)

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func reviewVideo(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	defer func() {
		if r := recover(); r != nil {
			apperr = extractError(r)
		}
	}()
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
	defer func() {
		if r := recover(); r != nil {
			apperr = extractError(r)
		}
	}()
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
	defer func() {
		if r := recover(); r != nil {
			apperr = extractError(r)
		}
	}()
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
	res, err := c.FetchApStat(context.Background(), &fetch.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid}, Seq: seq, Num: int32(num)})
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
	infos := make([]interface{}, len(res.Infos))
	for i := 0; i < len(res.Infos); i++ {
		json, _ := simplejson.NewJson([]byte(`{}`))
		json.Set("id", res.Infos[i].Id)
		json.Set("address", res.Infos[i].Address)
		json.Set("mac", res.Infos[i].Mac)
		json.Set("online", res.Infos[i].Online)
		json.Set("count", res.Infos[i].Count)
		json.Set("bandwidth", res.Infos[i].Bandwidth)
		infos[i] = json
	}
	js.SetPath([]string{"data", "infos"}, infos)
	js.SetPath([]string{"data", "total"}, res.Total)

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func getVideos(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	defer func() {
		if r := recover(); r != nil {
			apperr = extractError(r)
		}
	}()
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
	res, err := c.FetchVideos(context.Background(), &fetch.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid}, Seq: seq, Num: int32(num), Type: int32(ctype)})
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
	infos := make([]interface{}, len(res.Infos))
	for i := 0; i < len(res.Infos); i++ {
		json, _ := simplejson.NewJson([]byte(`{}`))
		json.Set("id", res.Infos[i].Id)
		json.Set("title", res.Infos[i].Title)
		json.Set("img", res.Infos[i].Img)
		json.Set("dst", res.Infos[i].Dst)
		json.Set("ctime", res.Infos[i].Ctime)
		json.Set("source", res.Infos[i].Source)
		json.Set("duration", res.Infos[i].Duration)
		infos[i] = json
	}
	js.SetPath([]string{"data", "infos"}, infos)
	js.SetPath([]string{"data", "total"}, res.Total)

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func getTemplates(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	defer func() {
		if r := recover(); r != nil {
			apperr = extractError(r)
		}
	}()
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
	res, err := c.FetchTemplates(context.Background(), &fetch.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid}, Seq: seq, Num: int32(num)})
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
	infos := make([]interface{}, len(res.Infos))
	for i := 0; i < len(res.Infos); i++ {
		json, _ := simplejson.NewJson([]byte(`{}`))
		json.Set("id", res.Infos[i].Id)
		json.Set("title", res.Infos[i].Title)
		json.Set("online", res.Infos[i].Online)
		json.Set("content", res.Infos[i].Content)
		infos[i] = json
	}
	js.SetPath([]string{"data", "infos"}, infos)
	js.SetPath([]string{"data", "total"}, res.Total)

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func addTemplate(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	defer func() {
		if r := recover(); r != nil {
			apperr = extractError(r)
		}
	}()
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
	defer func() {
		if r := recover(); r != nil {
			apperr = extractError(r)
		}
	}()
	var req request
	req.initCheckOss(r.Body)
	uid := req.GetParamInt("uid")
	img := req.GetParamString("img")
	dst := req.GetParamString("dst")
	priority := req.GetParamInt("priority")

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
			Info: &common.BannerInfo{Img: img, Dst: dst, Priority: int32(priority)}})
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

func modTemplate(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	defer func() {
		if r := recover(); r != nil {
			apperr = extractError(r)
		}
	}()
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

func getOssAps(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	return getAps(w, r, true)
}

func getBanners(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	defer func() {
		if r := recover(); r != nil {
			apperr = extractError(r)
		}
	}()
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
	res, err := c.FetchBanners(context.Background(), &fetch.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid}, Seq: seq, Num: int32(num)})
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
	infos := make([]interface{}, len(res.Infos))
	for i := 0; i < len(res.Infos); i++ {
		json, _ := simplejson.NewJson([]byte(`{}`))
		json.Set("id", res.Infos[i].Id)
		json.Set("img", res.Infos[i].Img)
		json.Set("dst", res.Infos[i].Dst)
		json.Set("online", res.Infos[i].Online)
		json.Set("priority", res.Infos[i].Priority)
		infos[i] = json
	}
	js.SetPath([]string{"data", "infos"}, infos)
	js.SetPath([]string{"data", "total"}, res.Total)

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
	mux.Handle("/add_template", appHandler(addTemplate))
	mux.Handle("/add_banner", appHandler(addBanner))
	mux.Handle("/mod_template", appHandler(modTemplate))
	mux.Handle("/get_nearby_aps", appHandler(getOssAps))
	mux.Handle("/review_news", appHandler(reviewNews))
	mux.Handle("/review_video", appHandler(reviewVideo))
	mux.Handle("/get_videos", appHandler(getVideos))
	mux.Handle("/get_banners", appHandler(getBanners))
	mux.Handle("/", http.FileServer(http.Dir("/data/server/oss")))
	return mux
}
