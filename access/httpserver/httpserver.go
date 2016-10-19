package httpserver

import (
	"log"
	"net/http"

	common "../../proto/common"
	helloworld "../../proto/hello"
	verify "../../proto/verify"
	util "../../util"
	simplejson "github.com/bitly/go-simplejson"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const (
	helloAddress  = "localhost:50051"
	verifyAddress = "localhost:50052"
	defaultName   = "world"
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
	log.Printf("request:%v", r.Body)
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
	log.Printf("request:%v", r.Body)
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

func register(w http.ResponseWriter, r *http.Request) {
	log.Printf("request:%v", r.Body)
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
	body, err := js.MarshalJSON()
	if err != nil {
		log.Printf("MarshalJSON failed: %v", err)
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}
	w.Write(body)
}

func Serve() {
	http.HandleFunc("/", welcome)
	http.HandleFunc("/hello", hello)
	http.HandleFunc("/login", login)
	http.HandleFunc("/get_phone_code", getPhoneCode)
	http.HandleFunc("/register", register)
	http.ListenAndServe(":80", nil)
}
