package weixin

import (
	"Server/util"
	"database/sql"
	"fmt"
	"log"

	simplejson "github.com/bitly/go-simplejson"
)

const (
	tokenURL = "https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential"
)

//RefreshAccessToken refresh access token
func RefreshAccessToken(db *sql.DB, appid string) (string, error) {
	var secret, token string
	var flag int
	err := db.QueryRow("SELECT secret, access_token, IF(expire_time > NOW(), 1, 0) FROM wx_token WHERE appid = ?", appid).
		Scan(&secret, &token, &flag)
	if err != nil {
		log.Printf("getAccessToken query failed:%v", err)
		return "", err
	}
	if flag == 1 && token != "" {
		return token, nil
	}
	token, err = GetAccessToken(appid, secret)
	if err != nil {
		log.Printf("getAccessToken  GetAccessToken failed:%s %v", appid, err)
		return "", err
	}
	_, err = db.Exec("UPDATE wx_token SET access_token = ?, expire_time = DATE_ADD(NOW(), INTERVAL 2 HOUR) WHERE appid = ?", token, appid)
	if err != nil {
		log.Printf("getAccessToken update token failed:%s %s %v",
			appid, token, err)
	}

	return token, nil
}

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
