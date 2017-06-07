package weixin

import (
	"Server/util"
	"fmt"
	"log"

	simplejson "github.com/bitly/go-simplejson"
)

const (
	wifiAppid        = "wx14a923201458f61b"
	wifiAppsecret    = "673d553f7f55f01a71752c4df1fefda9"
	baseurl          = "https://api.weixin.qq.com/sns/jscode2session"
	inquiryAppid     = "wx22f7ce89ec239c32"
	inquiryAppsecret = "0a126ec36e6b99da43cb1740d52f7d90"
)

//GetSession get wifi session key and openid
func GetSession(code string) (openid, sessionkey string, err error) {
	return getAppSession(code, wifiAppid, wifiAppsecret)
}

//GetInquirySession  get inquiry session key and openid
func GetInquirySession(code string) (openid, sessionkey string, err error) {
	return getAppSession(code, inquiryAppid, inquiryAppsecret)
}

//getAppSession get session key and openid for appid
func getAppSession(code, appid, appsecret string) (openid, sessionkey string, err error) {
	url := fmt.Sprintf("%s?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code",
		baseurl, appid, appsecret, code)
	resp, err := util.HTTPRequest(url, "")
	if err != nil {
		log.Printf("GetSession request failed:%v", err)
		return
	}
	log.Printf("url:%s resp:%s", url, resp)
	js, err := simplejson.NewJson([]byte(resp))
	if err != nil {
		log.Printf("GetSession parse response failed:%v", err)
		return
	}

	openid, err = js.Get("openid").String()
	if err != nil {
		return
	}

	sessionkey, err = js.Get("session_key").String()
	if err != nil {
		return
	}
	return
}
