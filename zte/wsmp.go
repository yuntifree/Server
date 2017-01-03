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
	dgSsid  = "无线东莞DG-FREE"
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

func getResponse(body string) (*simplejson.Json, error) {
	resp, err := util.HTTPRequest(wsmpURL, body)
	if err != nil {
		log.Printf("Register HTTPRequest failed:%v", err)
		return nil, err
	}
	js, err := simplejson.NewJson([]byte(resp))
	if err != nil {
		log.Printf("Register parse response failed:%v", err)
		return nil, err
	}

	ret, err := js.Get("head").Get("retflag").String()
	if err != nil {
		log.Printf("Register get retflag failed:%v", err)
		return nil, err
	}
	if ret != "0" {
		log.Printf("Register zte op failed retcode:%s resp:%s", ret, resp)
		return nil, errors.New("zte op failed")
	}
	return js, nil
}

//Register return password for new user
//smsFlag send sms or not
func Register(phone string, smsFlag bool) (string, error) {
	body, err := genRegisterBody(phone, smsFlag)
	if err != nil {
		log.Printf("Register genRegisterBody failed:%v", err)
		return "", err
	}

	js, err := getResponse(body)
	if err != nil {
		log.Printf("Register get response failed:%v", err)
		return "", err
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
func Remove(phone string) bool {
	body, err := genRemoveBody(phone)
	if err != nil {
		log.Printf("Remove genRemoveBody failed:%v", err)
		return false
	}

	_, err = getResponse(body)
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
func Login(phone, pass, userip, usermac, acip, acname string) bool {
	body, err := genLoginBody(phone, pass, userip, usermac, acip, acname)
	if err != nil {
		log.Printf("Login genLoginBody failed:%v", err)
		return false
	}

	_, err = getResponse(body)
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
func Loginnopass(phone, userip, usermac, acip, acname string) bool {
	body, err := genLoginnopassBody(phone, userip, usermac, acip, acname)
	if err != nil {
		log.Printf("Login genLoginBody failed:%v", err)
		return false
	}

	_, err = getResponse(body)
	if err != nil {
		log.Printf("Register getResponse failed:%v", err)
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
func Logout(phone, mac, userip, acip string) bool {
	body, err := genLogoutBody(phone, mac, userip, acip)
	if err != nil {
		log.Printf("Logout genLoginBody failed:%v", err)
		return false
	}

	_, err = getResponse(body)
	if err != nil {
		log.Printf("Logout getResponse failed:%v", err)
		return false
	}

	return true
}
