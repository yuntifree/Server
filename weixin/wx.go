package weixin

import (
	"Server/util"
	"encoding/base64"
	"fmt"
	"log"

	simplejson "github.com/bitly/go-simplejson"
)

const (
	wifiAppid        = "wx14a923201458f61b"
	wifiAppsecret    = "673d553f7f55f01a71752c4df1fefda9"
	baseurl          = "https://api.weixin.qq.com/sns/jscode2session"
	InquiryAppid     = "wx22f7ce89ec239c32"
	InquiryAppsecret = "0a126ec36e6b99da43cb1740d52f7d90"
	DgyAppid         = "wxb5c7692e667731fc"
	DgyAppsecret     = "e74c0945c26c5fd5f7cd51f116a4b525"
)

//WaterMark watermark
type WaterMark struct {
	Appid     string `json:"appid"`
	Timestamp int    `json:"timestamp"`
}

//UserInfo user info
type UserInfo struct {
	OpenId     string    `json:"openId"`
	NickName   string    `json:"nickName"`
	Gender     int64     `json:"gender"`
	City       string    `json:"city"`
	Province   string    `json:"province"`
	Country    string    `json:"country"`
	AvartarUrl string    `json:"avatarUrl"`
	UnionId    string    `json:"unionId"`
	Watermark  WaterMark `json:"watermark"`
}

//GetSession get wifi session key and openid
func GetSession(code string) (openid, sessionkey string, err error) {
	return getAppSession(code, wifiAppid, wifiAppsecret)
}

//GetInquirySession  get inquiry session key and openid
func GetInquirySession(code string) (openid, sessionkey string, err error) {
	return getAppSession(code, InquiryAppid, InquiryAppsecret)
}

//GetDgySession get dgy session key and openid
func GetDgySession(code string) (openid, sessionkey string, err error) {
	return getAppSession(code, DgyAppid, DgyAppsecret)
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

//DecryptData decrypt user raw data
func DecryptData(skey, encrypted, iv string) ([]byte, error) {
	src, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return []byte(""), err
	}
	key, err := base64.StdEncoding.DecodeString(skey)
	if err != nil {
		return []byte(""), err
	}
	vec, err := base64.StdEncoding.DecodeString(iv)
	if err != nil {
		return []byte(""), err
	}
	dst, err := util.AesDecrypt(src, key, vec)
	if err != nil {
		return []byte(""), err
	}
	return dst, nil
}
