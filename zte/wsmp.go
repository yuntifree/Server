package zte

import (
	"errors"
	"log"

	"../util"

	simplejson "github.com/bitly/go-simplejson"
)

const (
	sshWsmpURL = "http://120.234.130.196:880/wsmp/interface"
	wjjWsmpURL = "http://120.234.130.194:880/wsmp/interface"
	vnoCode    = "ROOT_VNO"
	dgSsid     = "无线东莞DG-FREE"
)
const (
	//SshType 松山湖系统
	SshType = iota
	//WjjType 卫计局系统
	WjjType
)

func genHead(action string) *simplejson.Json {
	js, err := simplejson.NewJson([]byte(`{}`))
	if err != nil {
		log.Printf("genHead failed:%v", err)
		return nil
	}
	js.Set("action", action)
	js.Set("vnoCode", vnoCode)
	return js
}

func genBody(m map[string]string) *simplejson.Json {
	js, err := simplejson.NewJson([]byte(`{}`))
	if err != nil {
		log.Printf("genBody new json failed:%v", err)
		return nil
	}
	for k, v := range m {
		js.Set(k, v)
	}
	return js
}

func genBodyStr(action string, body *simplejson.Json) (string, error) {
	head := genHead(action)
	if head == nil || body == nil {
		return "", errors.New("illegal head or body")
	}
	js, err := simplejson.NewJson([]byte(`{}`))
	if err != nil {
		log.Printf("genBodystr failed:%v", err)
		return "", err
	}
	js.Set("head", head)
	js.Set("body", body)
	data, err := js.Encode()
	if err != nil {
		log.Printf("genBodyStr failed:%v", err)
		return "", err
	}
	return string(data), nil
}

func genRegisterBody(phone string, smsFlag bool) (string, error) {
	var m map[string]string
	if smsFlag {
		m = map[string]string{"custCode": phone, "mobilePhone": phone}
	} else {
		m = map[string]string{"custCode": phone}
	}
	body := genBody(m)
	return genBodyStr("reg", body)
}

func genWsmpURL(stype uint) string {
	switch stype {
	default:
		return sshWsmpURL
	case WjjType:
		return wjjWsmpURL
	}
}

func getResponse(body string, stype uint) (*simplejson.Json, error) {
	url := genWsmpURL(stype)
	resp, err := util.HTTPRequest(url, body)
	if err != nil {
		log.Printf("HTTPRequest failed:%v", err)
		return nil, err
	}
	log.Printf("getResponse resp:%s", resp)
	js, err := simplejson.NewJson([]byte(resp))
	if err != nil {
		log.Printf("parse response failed:%v", err)
		return nil, err
	}

	ret, err := js.Get("head").Get("retflag").String()
	if err != nil {
		log.Printf("get retflag failed:%v", err)
		return nil, err
	}
	if ret != "0" {
		log.Printf("zte op failed retcode:%s resp:%s", ret, resp)
		return nil, errors.New("zte op failed")
	}
	return js, nil
}

//Register return password for new user
//smsFlag send sms or not
func Register(phone string, smsFlag bool, stype uint) (string, error) {
	body, err := genRegisterBody(phone, smsFlag)
	if err != nil {
		log.Printf("Register genRegisterBody failed:%v", err)
		return "", err
	}

	log.Printf("Register request body:%s", body)
	js, err := getResponse(body, stype)
	if err != nil {
		log.Printf("Register get response failed:%v", err)
		return "", err
	}

	retflag, err := js.Get("head").Get("retflag").String()
	if err != nil {
		log.Printf("Register get retflag failed:%v", err)
		return "", err
	}

	if retflag == "1" {
		reason, err := js.Get("head").Get("reason").String()
		if err != nil {
			log.Printf("Register get reason failed:%v", err)
			return "", err
		}
		if reason == "用户已经存在，请勿重复注册" {
			return "", nil
		}
	}

	pass, err := js.Get("body").Get("pwd").String()
	if err != nil {
		log.Printf("Register get pass failed:%v", err)
		return "", err
	}
	return pass, nil
}

func genRemoveBody(phone string) (string, error) {
	body := genBody(map[string]string{"custCode": phone})
	return genBodyStr("remove", body)
}

//Remove delete user
func Remove(phone string, stype uint) bool {
	body, err := genRemoveBody(phone)
	if err != nil {
		log.Printf("Remove genRemoveBody failed:%v", err)
		return false
	}

	_, err = getResponse(body, stype)
	if err != nil {
		log.Printf("Remove get response failed:%v", err)
		return false
	}

	return true
}

func genLoginBody(phone, pass, userip, usermac, acip, acname string) (string, error) {
	body := genBody(map[string]string{"custCode": phone,
		"pass": pass, "ssid": dgSsid, "mac": usermac, "ip": userip, "acip": acip, "acname": acname})
	return genBodyStr("login", body)
}

//Login user login
func Login(phone, pass, userip, usermac, acip, acname string, stype uint) bool {
	body, err := genLoginBody(phone, pass, userip, usermac, acip, acname)
	if err != nil {
		log.Printf("Login genLoginBody failed:%v", err)
		return false
	}

	log.Printf("Login request body:%s", body)
	_, err = getResponse(body, stype)
	if err != nil {
		log.Printf("Register getResponse failed:%v", err)
		return false
	}

	return true
}

func genLoginnopassBody(phone, userip, usermac, acip, acname string) (string, error) {
	body := genBody(map[string]string{"custCode": phone,
		"ssid": dgSsid, "mac": usermac, "ip": userip, "acip": acip, "acname": acname})
	return genBodyStr("loginnopass", body)
}

//Loginnopass user login without password
func Loginnopass(phone, userip, usermac, acip, acname string, stype uint) bool {
	body, err := genLoginnopassBody(phone, userip, usermac, acip, acname)
	if err != nil {
		log.Printf("Login genLoginBody failed:%v", err)
		return false
	}

	log.Printf("Loginnopass reqbody:%s", body)
	_, err = getResponse(body, stype)
	if err != nil {
		log.Printf("Loginnopass getResponse failed:%v", err)
		return false
	}

	return true
}

func genLogoutBody(phone, mac, userip, acip string) (string, error) {
	body := genBody(map[string]string{"custCode": phone,
		"mac": mac, "ip": userip, "acip": acip})
	return genBodyStr("logout", body)
}

//Logout user quit
func Logout(phone, mac, userip, acip string, stype uint) bool {
	body, err := genLogoutBody(phone, mac, userip, acip)
	if err != nil {
		log.Printf("Logout genLoginBody failed:%v", err)
		return false
	}

	_, err = getResponse(body, stype)
	if err != nil {
		log.Printf("Logout getResponse failed:%v", err)
		return false
	}

	return true
}
