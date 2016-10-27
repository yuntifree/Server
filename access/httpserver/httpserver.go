package httpserver

import (
	"log"
	"net/http"

	common "../../proto/common"
	discover "../../proto/discover"
	helloworld "../../proto/hello"
	hot "../../proto/hot"
	verify "../../proto/verify"
	util "../../util"
	simplejson "github.com/bitly/go-simplejson"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const (
	helloAddress    = "localhost:50051"
	verifyAddress   = "localhost:50052"
	hotAddress      = "localhost:50053"
	discoverAddress = "localhost:50054"
	defaultName     = "world"
)

type appHandler func(http.ResponseWriter, *http.Request) *util.AppError

func (fn appHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if e := fn(w, r); e != nil {
		log.Printf("error type:%d code:%d msg:%s", e.Type, e.Code, e.Msg)

		js, _ := simplejson.NewJson([]byte(`{}`))
		js.Set("errcode", e.Code)
		js.Set("desc", e.Msg)
		body, err := js.MarshalJSON()
		if err != nil {
			log.Printf("MarshalJSON failed: %v", err)
			w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
			return
		}
		w.Write(body)
	}
}

func getMessage(name string) string {
	conn, err := grpc.Dial(helloAddress, grpc.WithInsecure())
	if err != nil {
		log.Printf("did not connect: %v", err)
		return ""
	}
	defer conn.Close()
	c := helloworld.NewGreeterClient(conn)

	r, err := c.SayHello(context.Background(), &helloworld.HelloRequest{Name: name})
	if err != nil {
		log.Printf("could not greet: %v", err)
		return ""
	}

	return r.Message
}

func welcome(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Welcome to Yunti~"))
}

func hello(w http.ResponseWriter, r *http.Request) {
	name := "hello"
	message := getMessage(name)
	w.Write([]byte(message))
}

func login(w http.ResponseWriter, r *http.Request) *util.AppError {
	post, err := simplejson.NewFromReader(r.Body)
	if err != nil {
		return &util.AppError{util.JSONErr, 2, "", "invalid param"}
	}

	username, err := post.Get("data").Get("username").String()
	if err != nil {
		return &util.AppError{util.ParamErr, 1, "username", "miss param"}
	}

	password, err := post.Get("data").Get("password").String()
	if err != nil {
		return &util.AppError{util.ParamErr, 1, "password", "miss param"}
	}

	conn, err := grpc.Dial(verifyAddress, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, "", err.Error()}
	}
	defer conn.Close()
	c := verify.NewVerifyClient(conn)

	uuid := util.GenUUID()
	res, err := c.Login(context.Background(), &verify.LoginRequest{Head: &common.Head{Sid: uuid}, Username: username, Password: password})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, "", err.Error()}
	}

	if res.Head.Retcode != 0 {
		return &util.AppError{util.LogicErr, int(res.Head.Retcode), "", "登录失败"}
	}

	js, err := simplejson.NewJson([]byte(`{"errcode":0}`))
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "", err.Error()}
	}

	js.SetPath([]string{"data", "uid"}, res.Head.Uid)
	js.SetPath([]string{"data", "token"}, res.Token)
	js.SetPath([]string{"data", "privdata"}, res.Privdata)
	js.SetPath([]string{"data", "expire"}, res.Expire)
	js.SetPath([]string{"data", "wifipass"}, res.Wifipass)
	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "", err.Error()}
	}
	w.Write(body)
	return nil
}

func getCode(phone string, ctype int32) (bool, error) {
	conn, err := grpc.Dial(verifyAddress, grpc.WithInsecure())
	if err != nil {
		log.Printf("did not connect: %v", err)
		return false, err
	}
	defer conn.Close()
	c := verify.NewVerifyClient(conn)

	uuid := util.GenUUID()
	r, err := c.GetPhoneCode(context.Background(), &verify.CodeRequest{Head: &common.Head{Sid: uuid}, Phone: phone, Ctype: ctype})
	if err != nil {
		log.Printf("could not get phone code: %v", err)
		return false, err
	}

	return r.Result, nil
}

func getPhoneCode(w http.ResponseWriter, r *http.Request) *util.AppError {
	post, err := simplejson.NewFromReader(r.Body)
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "", "invalid param"}
	}

	phone, err := post.Get("data").Get("phone").String()
	if err != nil {
		return &util.AppError{util.ParamErr, 2, "phone", "invalid param"}
	}

	ctype, err := post.Get("data").Get("type").Int()
	if err != nil {
		return &util.AppError{util.ParamErr, 2, "type", "invalid param"}
	}
	flag, err := getCode(phone, int32(ctype))
	if err != nil || !flag {
		return &util.AppError{util.LogicErr, 103, "", "获取验证码失败"}
	}
	w.Write([]byte(`{"errno":0}`))
	return nil
}

func logout(w http.ResponseWriter, r *http.Request) *util.AppError {
	post, err := simplejson.NewFromReader(r.Body)
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "", "invalid param"}
	}

	token, err := post.Get("token").String()
	if err != nil {
		return &util.AppError{util.ParamErr, 2, "", "invalid param"}
	}

	uid, err := post.Get("uid").Int64()
	if err != nil {
		return &util.AppError{util.ParamErr, 2, "", "invalid param"}
	}

	conn, err := grpc.Dial(verifyAddress, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, "", err.Error()}
	}
	defer conn.Close()
	c := verify.NewVerifyClient(conn)

	uuid := util.GenUUID()
	res, err := c.Logout(context.Background(), &verify.LogoutRequest{Head: &common.Head{Sid: uuid, Uid: uid}, Token: token})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, "", err.Error()}
	}

	if res.Head.Retcode != 0 {
		return &util.AppError{util.LogicErr, 4, "", "logout failed"}
	}

	w.Write([]byte(`{"errno":0}`))
	return nil
}

func checkToken(uid int64, token string) bool {
	conn, err := grpc.Dial(verifyAddress, grpc.WithInsecure())
	if err != nil {
		log.Printf("did not connect: %v", err)
		return false
	}
	defer conn.Close()
	c := verify.NewVerifyClient(conn)

	uuid := util.GenUUID()
	res, err := c.CheckToken(context.Background(), &verify.TokenRequest{Head: &common.Head{Sid: uuid, Uid: uid}, Token: token})
	if err != nil {
		log.Printf("failed: %v", err)
		return false
	}

	if res.Head.Retcode != 0 {
		log.Printf("check token failed")
		return false
	}

	return true
}

func getHot(w http.ResponseWriter, r *http.Request) *util.AppError {
	post, err := simplejson.NewFromReader(r.Body)
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "", "invalid param"}
	}

	uid, err := post.Get("uid").Int64()
	if err != nil {
		return &util.AppError{util.ParamErr, 2, "uid", ""}
	}

	token, err := post.Get("token").String()
	if err != nil {
		return &util.AppError{util.ParamErr, 2, "token", ""}
	}

	flag := checkToken(uid, token)
	if !flag {
		return &util.AppError{util.LogicErr, 101, "", "token验证失败"}
	}

	ctype, err := post.Get("data").Get("type").Int()
	if err != nil {
		return &util.AppError{util.ParamErr, 2, "type", ""}
	}

	seq, err := post.Get("data").Get("seq").Int()
	if err != nil {
		return &util.AppError{util.ParamErr, 2, "seq", ""}
	}

	conn, err := grpc.Dial(hotAddress, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, "", err.Error()}
	}
	defer conn.Close()
	c := hot.NewHotClient(conn)

	uuid := util.GenUUID()
	res, err := c.GetHots(context.Background(), &hot.HotsRequest{Head: &common.Head{Sid: uuid, Uid: uid}, Type: int32(ctype), Seq: int32(seq)})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, "", err.Error()}
	}
	if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "", "获取新闻失败"}
	}

	js, err := simplejson.NewJson([]byte(`{"errcode":0}`))
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "", "invalid param"}
	}
	infos := make([]interface{}, len(res.Infos))
	for i := 0; i < len(res.Infos); i++ {
		json, _ := simplejson.NewJson([]byte(`{}`))
		json.Set("seq", res.Infos[i].Seq)
		json.Set("title", res.Infos[i].Title)
		if len(res.Infos[i].Images) > 0 {
			json.Set("images", res.Infos[i].Images)
		}
		json.Set("source", res.Infos[i].Source)
		json.Set("dst", res.Infos[i].Dst)
		json.Set("ctime", res.Infos[i].Ctime)
		json.Set("video", res.Infos[i].Video)
		infos[i] = json
	}
	js.SetPath([]string{"data", "infos"}, infos)
	if len(res.Infos) >= util.MaxListSize {
		js.SetPath([]string{"data", "hasmore"}, 1)
	}

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "", "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func autoLogin(w http.ResponseWriter, r *http.Request) *util.AppError {
	post, err := simplejson.NewFromReader(r.Body)
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "", "parse input json failed"}
	}

	uid, err := post.Get("uid").Int64()
	if err != nil {
		return &util.AppError{util.ParamErr, 2, "uid", ""}
	}

	token, err := post.Get("token").String()
	if err != nil {
		return &util.AppError{util.ParamErr, 2, "token", ""}
	}

	privdata, err := post.Get("data").Get("privdata").String()
	if err != nil {
		return &util.AppError{util.ParamErr, 2, "privdata", ""}
	}

	conn, err := grpc.Dial(verifyAddress, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, "", err.Error()}
	}
	defer conn.Close()
	c := verify.NewVerifyClient(conn)

	uuid := util.GenUUID()
	res, err := c.AutoLogin(context.Background(), &verify.AutoRequest{Head: &common.Head{Uid: uid, Sid: uuid}, Token: token, Privdata: privdata})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, "", err.Error()}
	}

	if res.Head.Retcode == common.ErrCode_INVALID_TOKEN {
		return &util.AppError{util.LogicErr, 4, "", "token验证失败"}
	} else if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "", "服务器又傲娇了"}
	}

	js, err := simplejson.NewJson([]byte(`{"errcode":0}`))
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "", "init json failed"}
	}

	js.SetPath([]string{"data", "token"}, res.Token)
	js.SetPath([]string{"data", "privdata"}, res.Privdata)
	js.SetPath([]string{"data", "expire"}, res.Expire)
	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "", "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func register(w http.ResponseWriter, r *http.Request) *util.AppError {
	post, err := simplejson.NewFromReader(r.Body)
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "", "parse input json failed"}
	}

	username, err := post.Get("data").Get("username").String()
	if err != nil {
		return &util.AppError{util.ParamErr, 2, "username", ""}
	}

	password, err := post.Get("data").Get("password").String()
	if err != nil {
		return &util.AppError{util.ParamErr, 2, "password", ""}
	}

	code, err := post.Get("data").Get("code").Int()
	if err != nil {
		return &util.AppError{util.ParamErr, 2, "password", ""}
	}

	conn, err := grpc.Dial(verifyAddress, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{util.RPCErr, 4, "", err.Error()}
	}
	defer conn.Close()
	c := verify.NewVerifyClient(conn)

	uuid := util.GenUUID()
	res, err := c.Register(context.Background(), &verify.RegisterRequest{Head: &common.Head{Sid: uuid}, Username: username, Password: password, Code: int32(code)})
	if err != nil {
		return &util.AppError{util.RPCErr, 4, "", err.Error()}
	}

	if res.Head.Retcode == common.ErrCode_USED_PHONE {
		return &util.AppError{util.LogicErr, 104, "", "该手机号已注册，请直接登录"}
	} else if res.Head.Retcode != 0 {
		return &util.AppError{util.DataErr, 4, "", "服务器又傲娇了"}
	}

	js, err := simplejson.NewJson([]byte(`{"errcode":0}`))
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "", "init json failed"}
	}

	js.SetPath([]string{"data", "uid"}, res.Head.Uid)
	js.SetPath([]string{"data", "token"}, res.Token)
	js.SetPath([]string{"data", "privdata"}, res.Privdata)
	js.SetPath([]string{"data", "expire"}, res.Expire)
	js.SetPath([]string{"data", "wifipass"}, res.Wifipass)
	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{util.JSONErr, 4, "", "marshal json failed"}
	}
	w.Write(body)
	return nil
}

func discoverServer(w http.ResponseWriter, r *http.Request) {
	post, err := simplejson.NewFromReader(r.Body)
	if err != nil {
		log.Printf("parse request body failed:%v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}

	name, err := post.Get("name").String()
	if err != nil {
		log.Printf("get name failed:%v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}

	conn, err := grpc.Dial(discoverAddress, grpc.WithInsecure())
	if err != nil {
		log.Printf("did not connect: %v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}
	defer conn.Close()
	c := discover.NewDiscoverClient(conn)

	uuid := util.GenUUID()
	res, err := c.Resolve(context.Background(), &discover.ServerRequest{Head: &common.Head{Sid: uuid}, Sname: name})
	if err != nil {
		log.Printf("Login failed: %v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}

	if res.Head.Retcode == common.ErrCode_USED_PHONE {
		w.Write([]byte(`{"errno":104,"desc":"该手机号已注册，请直接登录"}`))
		return
	}

	js, err := simplejson.NewJson([]byte(`{"errcode":0}`))
	if err != nil {
		log.Printf("NewJson failed: %v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}

	js.SetPath([]string{"data", "host"}, res.Host)
	js.SetPath([]string{"data", "port"}, res.Port)
	body, err := js.MarshalJSON()
	if err != nil {
		log.Printf("MarshalJSON failed: %v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}
	w.Write(body)

}

//Serve do server work
func Serve() {
	http.HandleFunc("/", welcome)
	http.HandleFunc("/hello", hello)
	http.Handle("/login", appHandler(login))
	http.Handle("/get_phone_code", appHandler(getPhoneCode))
	http.Handle("/register", appHandler(register))
	http.Handle("/logout", appHandler(logout))
	http.Handle("/hot", appHandler(getHot))
	http.Handle("/auto_login", appHandler(autoLogin))
	http.HandleFunc("/discover", discoverServer)
	http.ListenAndServe(":80", nil)
}
