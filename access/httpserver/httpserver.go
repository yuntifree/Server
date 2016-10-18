package httpserver

import (
	"log"
	"net/http"
	"strconv"

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

func checkPhoneCode(phone string, code int32) (bool, error) {
	conn, err := grpc.Dial(verifyAddress, grpc.WithInsecure())
	if err != nil {
		log.Printf("did not connect: %v", err)
		return false, err
	}
	defer conn.Close()
	c := verify.NewVerifyClient(conn)

	uuid := util.GenUUID()
	r, err := c.VerifyPhoneCode(context.Background(), &verify.PhoneRequest{Head: &common.Head{Sid: uuid}, Phone: phone, Code: code})
	if err != nil {
		log.Printf("could not verify phone code: %v", err)
		return false, err
	}

	return r.Result, nil
}

func login(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

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
	err := r.ParseForm()
	if err != nil {
		w.Write([]byte(`{"errno":2,"desc":"invalid param"}`))
		return
	}
	phone := r.FormValue("phone")
	ctype, _ := strconv.Atoi(r.FormValue("type"))
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

func Serve() {
	http.HandleFunc("/", welcome)
	http.HandleFunc("/hello", hello)
	http.HandleFunc("/login", login)
	http.HandleFunc("/phonecode", getPhoneCode)
	http.ListenAndServe(":80", nil)
}
