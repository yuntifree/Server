package weixin

import (
	"Server/util"
	"fmt"
	"log"

	simplejson "github.com/bitly/go-simplejson"
)

const (
	tokenURL = "https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential"
)

//GetAccessToken get wx access token
func GetAccessToken(appid, appsec string) (token string, err error) {
	url := fmt.Sprintf("%s&appid=%s&secret=%s", tokenURL, appid, appsec)
	res, err := util.HTTPRequest(url, "")
	if err != nil {
		log.Printf("fetch url %s failed:%v", url, err)
		return
	}

	log.Printf("GetAccessToken resp:%s\n", res)
	js, err := simplejson.NewJson([]byte(res))
	if err != nil {
		log.Printf("parse resp %s failed:%v", res, err)
		return
	}

	token, err = js.Get("access_token").String()
	if err != nil {
		log.Printf("json get access_token failed:%v", err)
	}
	return
}
