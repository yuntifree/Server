package zte

import (
	"errors"
	"log"

	"../util"

	simplejson "github.com/bitly/go-simplejson"
)

const (
	wsmpURL = "http://120.234.130.196:880/wsmp/interface"
	vnoCode = "ROOT_VNO"
)

func genHead(action string) (*simplejson.Json, error) {
	js, err := simplejson.NewJson([]byte(`{}`))
	if err != nil {
		log.Printf("genHead failed:%v", err)
		return nil, err
	}
	js.Set("action", action)
	js.Set("vnoCode", vnoCode)
	return js, nil
}

func genBody(m map[string]string) (*simplejson.Json, error) {
	js, err := simplejson.NewJson([]byte(`{}`))
	if err != nil {
		log.Printf("genBody new json failed:%v", err)
		return nil, err
	}
	for k, v := range m {
		js.Set(k, v)
	}
	return js, nil
}

func genRegisterBody(phone string) (string, error) {
	head, err := genHead("reg")
	if err != nil {
		log.Printf("genHead failed:%v", err)
		return "", err
	}
	body, err := genBody(map[string]string{"custCode": phone})
	if err != nil {
		log.Printf("genBody failed:%v", err)
		return "", err
	}
	js, err := simplejson.NewJson([]byte(`{}`))
	js.Set("head", head)
	js.Set("body", body)
	data, err := js.Encode()
	if err != nil {
		log.Printf("genBody failed:%v", err)
		return "", err
	}
	return string(data), nil
}

//Register return password for new user
func Register(phone string) (string, error) {
	body, err := genRegisterBody(phone)
	if err != nil {
		log.Printf("Register genRegisterBody failed:%v", err)
		return "", err
	}
	resp, err := util.HTTPRequest(wsmpURL, body)
	if err != nil {
		log.Printf("Register HTTPRequest failed:%v", err)
		return "", err
	}
	js, err := simplejson.NewJson([]byte(resp))
	if err != nil {
		log.Printf("Register parse response failed:%v", err)
		return "", err
	}

	ret, err := js.Get("head").Get("retflag").String()
	if err != nil {
		log.Printf("Register get retflag failed:%v", err)
		return "", err
	}
	if ret != "0" {
		log.Printf("Register zte op failed retcode:%d", ret)
		return "", errors.New("zte op failed")
	}

	pass, err := js.Get("body").Get("pwd").String()
	if err != nil {
		log.Printf("Register get pass failed:%v", err)
		return "", err
	}
	return pass, nil
}
