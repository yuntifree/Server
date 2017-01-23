package aliyun

import (
	"fmt"
	"log"
	"net/url"
	"sort"
	"time"

	"Server/util"
)

const (
	stsAccessKeyID     = "LTAIBKxRwMQqYvLH"
	stsAccessKeySecret = "mUw3s7Ego87Tcr8UXruZIstoERv9Of"
	stsRoleArn         = "acs:ram::1366075934953540:role/aliyunosstokengeneratorrole"
	stsRoleSessionName = "external-username"
	stsDurationSeconds = 3600
	stsHost            = "https://sts.aliyuncs.com"
)

func getUTCISO8601Time(ts time.Time) string {
	return ts.UTC().Format("2006-01-02T15:04:05Z")
}

func genMapStr(m map[string]string) string {
	var cts sort.StringSlice
	for k, v := range m {
		str := fmt.Sprintf("%s=%s&", k, url.QueryEscape(v))
		cts = append(cts, str)
	}
	sort.Sort(cts)
	var res string
	for _, v := range cts {
		res += v
	}

	return res[:len(res)-1]
}

func genStsSign(content string) string {
	key := stsAccessKeySecret + "&"
	val := "GET&%2F&" + url.QueryEscape(content)
	sign := genSign(val, key)
	return sign
}

//FetchStsCredentials fetch sts credentials
func FetchStsCredentials() string {
	nonce := util.GenUUID()
	timestamp := getUTCISO8601Time(time.Now())
	m := map[string]string{
		"Format":           "JSON",
		"Version":          "2015-04-01",
		"AccessKeyId":      stsAccessKeyID,
		"SignatureMethod":  "HMAC-SHA1",
		"SignatureVersion": "1.0",
		"SignatureNonce":   nonce,
		"Timestamp":        timestamp,
		"Action":           "AssumeRole",
		"RoleArn":          stsRoleArn,
		"RoleSessionName":  stsRoleSessionName,
		"Duration":         "900",
	}
	str := genMapStr(m)
	sign := genStsSign(str)
	log.Printf("str:%s signature:%s", str, sign)
	url := stsHost + "?" + str + "&Signature=" + url.QueryEscape(sign)
	log.Printf("url:%s", url)
	res, err := util.HTTPRequest(url, "")
	if err != nil {
		log.Printf("FetchStsCredentials HTTPRequest failed:%v", err)
		return ""
	}

	return res
}
