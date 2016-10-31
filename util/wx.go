package util

import (
	"errors"
	"log"

	simplejson "github.com/bitly/go-simplejson"
)

const (
	wxAppid    = ""
	wxAppkey   = ""
	wxTokenURL = "https://api.weixin.qq.com/sns/oauth2/access_token"
	wxInfoURL  = "https://api.weixin.qq.com/sns/userinfo"
)

//WxInfo wx login info
type WxInfo struct {
	Openid, Token, NickName, HeadURL, UnionID string
	Sex                                       int
}

//GetCodeToken use code to get wx login info
func GetCodeToken(code string) (wxi WxInfo, err error) {
	url := wxTokenURL + "?appid=" + appid + "&secret=" + appkey + "&code=" + code + "&grant_type=authorization_code"
	res, err := HTTPRequest(url, "")
	if err != nil {
		log.Printf("fetch url %s failed:%v", url, err)
		return
	}

	js, err := simplejson.NewJson([]byte(res))
	if err != nil {
		log.Printf("parse resp %s failed:%v", res, err)
		return
	}

	openid, err := js.Get("openid").String()
	if err != nil {
		log.Printf("get openid failed:%v", err)
		return
	}

	token, err := js.Get("access_token").String()
	if err != nil {
		log.Printf("get access_token failed:%v", err)
		return
	}

	wxi.Openid = openid
	wxi.Token = token
	return
}

//GetWxInfo get wx user info
func GetWxInfo(wxi *WxInfo) (err error) {
	url := wxInfoURL + "?access_token=" + wxi.Token + "&openid=" + wxi.Openid
	res, err := HTTPRequest(url, "")
	if err != nil {
		log.Printf("fetch url %s failed:%v", url, err)
		return
	}

	js, err := simplejson.NewJson([]byte(res))
	if err != nil {
		log.Printf("parse resp %s failed:%v", res, err)
		return
	}

	errcode, err := js.Get("errcode").Int()
	if err != nil {
		log.Printf("get errcode failed:%v", err)
		return
	}
	if errcode != 0 {
		log.Printf("errcode :%d", errcode)
		err = errors.New("get wx info failed")
	}

	nickname, err := js.Get("nickname").String()
	if err != nil {
		log.Printf("get nickname failed:%v", err)
		return
	}
	unionid, err := js.Get("unionid").String()
	if err != nil {
		log.Printf("get unionid failed:%v", err)
		return
	}
	wxi.NickName = nickname
	wxi.UnionID = unionid
	wxi.HeadURL, _ = js.Get("headimgurl").String()
	wxi.Sex, _ = js.Get("sex").Int()

	return
}
