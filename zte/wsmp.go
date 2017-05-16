package zte

import (
	"errors"
	"log"
	"time"

	"Server/util"

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

var errStatus = errors.New("zte op failed")
var ErrForbid = errors.New("login forbid")

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
		log.Printf("zte op failed body:%s retcode:%s resp:%s", body, ret, resp)
		return js, errStatus
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
		if err == errStatus {
			reason, err := js.Get("head").Get("reason").String()
			if err != nil {
				log.Printf("Register get reason failed:%v", err)
				return "", err
			}
			if reason == "用户已经存在，请勿重复注册" {
				return "", nil
			}
		}
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
func Loginnopass(phone, userip, usermac, acip, acname string, stype uint) (bool, error) {
	body, err := genLoginnopassBody(phone, userip, usermac, acip, acname)
	if err != nil {
		log.Printf("Login genLoginBody failed:%v", err)
		return false, nil
	}

	log.Printf("Loginnopass reqbody:%s", body)
	js, err := getResponse(body, stype)
	if err != nil {
		log.Printf("Loginnopass getResponse failed:%v", err)
		if err == errStatus {
			reason, err := js.Get("head").Get("reason").String()
			if err != nil {
				log.Printf("Loginnopass get reason failed:%v", err)
				return false, nil
			}
			log.Printf("reason:%s", reason)
			if reason == "无线接入控制失败或限制接入" {
				if QueryOnline(phone, stype) {
					log.Printf("Loginnopass queryonline succ:%s", phone)
					return true, nil
				}
				log.Printf("Loginnopass QueryOnline failed phone:%s", phone)
			} else if reason == "您的帐号或者设备被禁止访问网络" {
				return false, ErrForbid
			}
		}
		return false, nil
	}

	return true, nil
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

func genQueryOnlineBody(phone string) (string, error) {
	end := time.Now().Format("2006-01-02")
	body := genBody(map[string]string{"custCode": phone,
		"opervnocode": "ROOT_VNO", "status": "30A", "enddate": end,
		"isbysubvno": "1", "pageno": "1", "pagesize": "10"})
	return genBodyStr("qryonlineinfo", body)
}

//QueryOnline query phone online status
func QueryOnline(phone string, stype uint) bool {
	body, err := genQueryOnlineBody(phone)
	if err != nil {
		log.Printf("QueryOnline genQueryOnlineBody failed:%v", err)
		return false
	}

	res, err := getResponse(body, stype)
	if err != nil {
		log.Printf("QueryOnline getResponse failed:%v", err)
		return false
	}

	log.Printf("QueryOnline req:%s resp:%v", body, res)
	rspbody, err := res.Get("body").Array()
	if err != nil {
		log.Printf("QueryOnline get body failed:%v", err)
		return false
	}

	if len(rspbody) == 0 {
		return false
	}
	return true
}
