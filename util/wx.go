package util

import (
	"fmt"
	"log"
	"net/url"

	simplejson "github.com/bitly/go-simplejson"
)

const (
	wxAppid    = "wx0387308775179bfe"
	wxAppkey   = "829008d0ae26aa03522bc0dbc370d790"
	wxTokenURL = "https://api.weixin.qq.com/sns/oauth2/access_token"
	wxInfoURL  = "https://api.weixin.qq.com/sns/userinfo"
	wxAuthURL  = "https://open.weixin.qq.com/connect/oauth2/authorize"
)

//WxInfo wx login info
type WxInfo struct {
	Openid, Token, NickName, HeadURL, UnionID string
	Sex                                       int
}

//GenRedirectURL generate redirect url
func GenRedirectURL(redirect string) string {
	return fmt.Sprintf("%s?appid=%s&redirect_uri=%s&response_type=code&scope=snsapi_userinfo&state=list#wechat_redirect", wxAuthURL, wxAppid, url.QueryEscape(redirect))
}

//GetCodeToken use code to get wx login info
func GetCodeToken(code string) (wxi WxInfo, err error) {
	url := wxTokenURL + "?appid=" + wxAppid + "&secret=" + wxAppkey + "&code=" + code + "&grant_type=authorization_code"
	log.Printf("url:%s\n", url)
	res, err := HTTPRequest(url, "")
	if err != nil {
		log.Printf("fetch url %s failed:%v", url, err)
		return
	}

	log.Printf("GetCodeToken resp:%s\n", res)
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
	log.Printf("openid:%s token:%s\n", openid, token)

	wxi.Openid = openid
	wxi.Token = token
	return
}

//GetWxInfo get wx user info
func GetWxInfo(wxi *WxInfo) (err error) {
	url := wxInfoURL + "?access_token=" + wxi.Token + "&openid=" + wxi.Openid
	log.Printf("url:%s\n", url)
	res, err := HTTPRequest(url, "")
	if err != nil {
		log.Printf("fetch url %s failed:%v", url, err)
		return
	}

	log.Printf("GetWxInfo resp:%s\n", res)
	js, err := simplejson.NewJson([]byte(res))
	if err != nil {
		log.Printf("parse resp %s failed:%v", res, err)
		return
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
