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

func login(w http.ResponseWriter, r *http.Request) {
	post, err := simplejson.NewFromReader(r.Body)
	if err != nil {
		log.Printf("parse request body failed:%v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}

	username, err := post.Get("data").Get("username").String()
	if err != nil {
		log.Printf("get username failed:%v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}

	password, err := post.Get("data").Get("password").String()
	if err != nil {
		log.Printf("get password failed:%v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}

	conn, err := grpc.Dial(verifyAddress, grpc.WithInsecure())
	if err != nil {
		log.Printf("did not connect: %v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}
	defer conn.Close()
	c := verify.NewVerifyClient(conn)

	uuid := util.GenUUID()
	res, err := c.Login(context.Background(), &verify.LoginRequest{Head: &common.Head{Sid: uuid}, Username: username, Password: password})
	if err != nil {
		log.Printf("Login failed: %v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}

	js, err := simplejson.NewJson([]byte(`{"errcode":0}`))
	if err != nil {
		log.Printf("NewJson failed: %v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}

	js.SetPath([]string{"data", "uid"}, res.Head.Uid)
	js.SetPath([]string{"data", "token"}, res.Token)
	js.SetPath([]string{"data", "privdata"}, res.Privdata)
	js.SetPath([]string{"data", "expire"}, res.Expire)
	js.SetPath([]string{"data", "wifipass"}, res.Wifipass)
	body, err := js.MarshalJSON()
	if err != nil {
		log.Printf("MarshalJSON failed: %v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}
	w.Write(body)

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

func getPhoneCode(w http.ResponseWriter, r *http.Request) {
	post, err := simplejson.NewFromReader(r.Body)
	if err != nil {
		log.Printf("parse request body failed:%v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}

	phone, err := post.Get("data").Get("phone").String()
	if err != nil {
		log.Printf("get phone failed:%v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}

	ctype, err := post.Get("data").Get("type").Int()
	if err != nil {
		log.Printf("get password failed:%v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}
	flag, err := getCode(phone, int32(ctype))
	if err != nil {
		log.Printf("get code failed: %v", err)
		w.Write([]byte(`{"errno":103,"desc":"get code failed"}`))
		return
	}
	if !flag {
		log.Printf("get code failed")
		w.Write([]byte(`{"errno":103,"desc":"get code failed"}`))
		return
	}
	w.Write([]byte(`{"errno":0}`))
}

func logout(w http.ResponseWriter, r *http.Request) {
	post, err := simplejson.NewFromReader(r.Body)
	if err != nil {
		log.Printf("parse request body failed:%v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}

	token, err := post.Get("token").String()
	if err != nil {
		log.Printf("get token failed:%v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}

	uid, err := post.Get("uid").Int64()
	if err != nil {
		log.Printf("get password failed:%v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}

	conn, err := grpc.Dial(verifyAddress, grpc.WithInsecure())
	if err != nil {
		log.Printf("did not connect: %v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}
	defer conn.Close()
	c := verify.NewVerifyClient(conn)

	uuid := util.GenUUID()
	res, err := c.Logout(context.Background(), &verify.LogoutRequest{Head: &common.Head{Sid: uuid, Uid: uid}, Token: token})
	if err != nil {
		log.Printf("Login failed: %v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}

	if res.Head.Retcode != 0 {
		w.Write([]byte(`{"errno":4,"desc:"logout failed"}`))
		return
	}

	w.Write([]byte(`{"errno":0}`))

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

func getHot(w http.ResponseWriter, r *http.Request) {
	post, err := simplejson.NewFromReader(r.Body)
	if err != nil {
		log.Printf("parse request body failed:%v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}

	uid, err := post.Get("uid").Int64()
	if err != nil {
		log.Printf("get uid failed:%v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}

	token, err := post.Get("token").String()
	if err != nil {
		log.Printf("get uid failed:%v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}

	flag := checkToken(uid, token)
	if !flag {
		log.Printf("get uid failed:%v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}

	ctype, err := post.Get("data").Get("type").Int()
	if err != nil {
		log.Printf("get type failed:%v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}

	seq, err := post.Get("data").Get("seq").Int()
	if err != nil {
		log.Printf("get seq failed:%v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}

	conn, err := grpc.Dial(hotAddress, grpc.WithInsecure())
	if err != nil {
		log.Printf("did not connect: %v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
	}
	defer conn.Close()
	c := hot.NewHotClient(conn)

	uuid := util.GenUUID()
	res, err := c.GetHots(context.Background(), &hot.HotsRequest{Head: &common.Head{Sid: uuid, Uid: uid}, Type: int32(ctype), Seq: int32(seq)})
	if err != nil {
		log.Printf("failed: %v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}
	if res.Head.Retcode != 0 {
		log.Printf("get hots failed errcode:%d", res.Head.Retcode)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}

	js, err := simplejson.NewJson([]byte(`{"errcode":0}`))
	if err != nil {
		log.Printf("NewJson failed: %v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
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
		log.Printf("MarshalJSON failed: %v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}
	w.Write(body)
}
func autoLogin(w http.ResponseWriter, r *http.Request) {
	post, err := simplejson.NewFromReader(r.Body)
	if err != nil {
		log.Printf("parse request body failed:%v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}

	uid, err := post.Get("uid").Int64()
	if err != nil {
		log.Printf("get uid failed:%v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}

	token, err := post.Get("token").String()
	if err != nil {
		log.Printf("get token failed:%v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}

	privdata, err := post.Get("data").Get("privdata").String()
	if err != nil {
		log.Printf("get privdata failed:%v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}

	conn, err := grpc.Dial(verifyAddress, grpc.WithInsecure())
	if err != nil {
		log.Printf("did not connect: %v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}
	defer conn.Close()
	c := verify.NewVerifyClient(conn)

	uuid := util.GenUUID()
	res, err := c.AutoLogin(context.Background(), &verify.AutoRequest{Head: &common.Head{Uid: uid, Sid: uuid}, Token: token, Privdata: privdata})
	if err != nil {
		log.Printf("Login failed: %v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}

	if res.Head.Retcode == common.ErrCode_INVALID_TOKEN {
		w.Write([]byte(`{"errno":101,"desc":"token验证失败"}`))
		return
	}

	js, err := simplejson.NewJson([]byte(`{"errcode":0}`))
	if err != nil {
		log.Printf("NewJson failed: %v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}

	js.SetPath([]string{"data", "token"}, res.Token)
	js.SetPath([]string{"data", "privdata"}, res.Privdata)
	js.SetPath([]string{"data", "expire"}, res.Expire)
	body, err := js.MarshalJSON()
	if err != nil {
		log.Printf("MarshalJSON failed: %v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}
	w.Write(body)
}

func register(w http.ResponseWriter, r *http.Request) {
	post, err := simplejson.NewFromReader(r.Body)
	if err != nil {
		log.Printf("parse request body failed:%v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}

	username, err := post.Get("data").Get("username").String()
	if err != nil {
		log.Printf("get username failed:%v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}

	password, err := post.Get("data").Get("password").String()
	if err != nil {
		log.Printf("get password failed:%v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}

	code, err := post.Get("data").Get("code").Int()
	if err != nil {
		log.Printf("get code failed:%v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}

	conn, err := grpc.Dial(verifyAddress, grpc.WithInsecure())
	if err != nil {
		log.Printf("did not connect: %v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}
	defer conn.Close()
	c := verify.NewVerifyClient(conn)

	uuid := util.GenUUID()
	res, err := c.Register(context.Background(), &verify.RegisterRequest{Head: &common.Head{Sid: uuid}, Username: username, Password: password, Code: int32(code)})
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

	js.SetPath([]string{"data", "uid"}, res.Head.Uid)
	js.SetPath([]string{"data", "token"}, res.Token)
	js.SetPath([]string{"data", "privdata"}, res.Privdata)
	js.SetPath([]string{"data", "expire"}, res.Expire)
	js.SetPath([]string{"data", "wifipass"}, res.Wifipass)
	body, err := js.MarshalJSON()
	if err != nil {
		log.Printf("MarshalJSON failed: %v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}
	w.Write(body)
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
	http.HandleFunc("/login", login)
	http.HandleFunc("/get_phone_code", getPhoneCode)
	http.HandleFunc("/register", register)
	http.HandleFunc("/logout", logout)
	http.HandleFunc("/hot", getHot)
	http.HandleFunc("/auto_login", autoLogin)
	http.HandleFunc("/discover", discoverServer)
	http.ListenAndServe(":80", nil)
}
