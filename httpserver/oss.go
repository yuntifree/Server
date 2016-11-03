package httpserver

import (
	"context"
	"net/http"

	common "../proto/common"
	fetch "../proto/fetch"
	verify "../proto/verify"
	util "../util"
	simplejson "github.com/bitly/go-simplejson"
	"google.golang.org/grpc"
)

func backLogin(w http.ResponseWriter, r *http.Request) (apperr *util.AppError) {
	defer func() {
		if r := recover(); r != nil {
			if v, ok := r.(util.ParamError); ok {
				apperr = &util.AppError{util.ParamErr, 2, v.Error()}
			}
		}
	}()
	post, err := simplejson.NewFromReader(r.Body)
	if err != nil {
		return &util.AppError{util.JSONErr, 2, "invalid param"}
	}

	username := util.GetJSONString(post, "username")
	password := util.GetJSONString(post, "password")

	conn, err := grpc.Dial(verifyAddress, grpc.WithInsecure())
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

	js, err := simplejson.NewJson([]byte(`{"errcode":0}`))
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
			if v, ok := r.(util.ParamError); ok {
				apperr = &util.AppError{util.ParamErr, 2, v.Error()}
			}
		}
	}()
	post, err := simplejson.NewFromReader(r.Body)
	if err != nil {
		return &util.AppError{util.JSONErr, 2, "invalid param"}
	}

	uid := util.GetJSONInt(post, "uid")
	token := util.GetJSONString(post, "token")

	flag := checkToken(uid, token, 1)
	if !flag {
		return &util.AppError{util.LogicErr, 101, "token验证失败"}
	}

	num := util.GetJSONInt(post, "num")
	seq := util.GetJSONInt(post, "seq")
	if num > 100 {
		num = 100
	} else if num < 20 {
		num = 20
	}

	conn, err := grpc.Dial(fetchAddress, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	defer conn.Close()
	c := fetch.NewFetchClient(conn)

	uuid := util.GenUUID()
	res, err := c.FetchReviewNews(context.Background(), &fetch.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid}, Seq: seq, Num: int32(num)})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "获取新闻失败"}
	}

	js, err := simplejson.NewJson([]byte(`{"errcode":0}`))
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "invalid param"}
	}
	infos := make([]interface{}, len(res.Infos))
	for i := 0; i < len(res.Infos); i++ {
		json, _ := simplejson.NewJson([]byte(`{}`))
		json.Set("id", res.Infos[i].Id)
		json.Set("seq", res.Infos[i].Id)
		json.Set("title", res.Infos[i].Title)
		infos[i] = json
	}
	js.SetPath([]string{"data", "news"}, infos)
	if len(res.Infos) >= util.MaxListSize {
		js.SetPath([]string{"data", "hasmore"}, 1)
	}

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
			if v, ok := r.(util.ParamError); ok {
				apperr = &util.AppError{util.ParamErr, 2, v.Error()}
			}
		}
	}()
	post, err := simplejson.NewFromReader(r.Body)
	if err != nil {
		return &util.AppError{util.JSONErr, 2, "invalid param"}
	}

	uid := util.GetJSONInt(post, "uid")
	token := util.GetJSONString(post, "token")

	flag := checkToken(uid, token, 1)
	if !flag {
		return &util.AppError{util.LogicErr, 101, "token验证失败"}
	}

	num := util.GetJSONInt(post, "num")
	seq := util.GetJSONInt(post, "seq")
	if num > 100 {
		num = 100
	} else if num < 20 {
		num = 20
	}

	conn, err := grpc.Dial(fetchAddress, grpc.WithInsecure())
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

	js, err := simplejson.NewJson([]byte(`{"errcode":0}`))
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "invalid param"}
	}
	infos := make([]interface{}, len(res.Infos))
	for i := 0; i < len(res.Infos); i++ {
		json, _ := simplejson.NewJson([]byte(`{}`))
		json.Set("id", res.Infos[i].Id)
		json.Set("seq", res.Infos[i].Id)
		json.Set("content", res.Infos[i].Content)
		infos[i] = json
	}
	js.SetPath([]string{"data", "tags"}, infos)
	if len(res.Infos) >= util.MaxListSize {
		js.SetPath([]string{"data", "hasmore"}, 1)
	}

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
			if v, ok := r.(util.ParamError); ok {
				apperr = &util.AppError{util.ParamErr, 2, v.Error()}
			}
		}
	}()
	post, err := simplejson.NewFromReader(r.Body)
	if err != nil {
		return &util.AppError{util.JSONErr, 2, "invalid param"}
	}

	uid := util.GetJSONInt(post, "uid")
	token := util.GetJSONString(post, "token")

	flag := checkToken(uid, token, 1)
	if !flag {
		return &util.AppError{util.LogicErr, 101, "token验证失败"}
	}

	num := util.GetJSONInt(post, "num")
	seq := util.GetJSONInt(post, "seq")
	if num > 100 {
		num = 100
	} else if num < 20 {
		num = 20
	}

	conn, err := grpc.Dial(fetchAddress, grpc.WithInsecure())
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

	js, err := simplejson.NewJson([]byte(`{"errcode":0}`))
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "invalid param"}
	}
	infos := make([]interface{}, len(res.Infos))
	for i := 0; i < len(res.Infos); i++ {
		json, _ := simplejson.NewJson([]byte(`{}`))
		json.Set("id", res.Infos[i].Id)
		json.Set("seq", res.Infos[i].Id)
		json.Set("imei", res.Infos[i].Imei)
		json.Set("phone", res.Infos[i].Phone)
		json.Set("active", res.Infos[i].Active)
		json.Set("remark", res.Infos[i].Remark)
		infos[i] = json
	}
	js.SetPath([]string{"data", "infos"}, infos)
	if len(res.Infos) >= util.MaxListSize {
		js.SetPath([]string{"data", "hasmore"}, 1)
	}

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "marshal json failed"}
	}
	w.Write(body)
	return nil
}

//ServeOss do oss server work
func ServeOss() {
	http.Handle("/login", appHandler(backLogin))
	http.Handle("/get_news", appHandler(getReviewNews))
	http.Handle("/get_tags", appHandler(getTags))
	http.Handle("/get_users", appHandler(getUsers))
	http.Handle("/", http.FileServer(http.Dir("/data/server/oss")))
	http.ListenAndServe(":8080", nil)
}
