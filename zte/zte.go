package zte

import (
	"log"

	util "../util"
	simplejson "github.com/bitly/go-simplejson"
)

const (
	baseurl = "http://120.76.236.185"
)

//APInfo ap base information
type APInfo struct {
	aid, ssid, longitude, latitude, address string
}

//UserInfo online user information
type UserInfo struct {
	username, phone, mac string
}

//RealInfo realtime information
type RealInfo struct {
	bandwidth string
	online    int
	infos     []UserInfo
}

//OnlineRecord user online record
type OnlineRecord struct {
	aid, start, end string
}

func genReqbody(reqinfos map[string]string) (string, error) {
	js, err := simplejson.NewJson([]byte(`{}`))
	if err != nil {
		log.Printf("new json failed:%v", err)
		return "", err
	}
	for k, v := range reqinfos {
		js.Set(k, v)
	}
	data, err := js.Encode()
	if err != nil {
		log.Printf("json encode failed:%v", err)
		return "", err
	}

	return string(data), nil
}

//GetAPInfoList fetch ap info
func GetAPInfoList(seq string) []APInfo {
	infos := make([]APInfo, util.MaxListSize)
	url := baseurl + "/apInfoList"
	data, err := genReqbody(map[string]string{"seq": seq})
	rspbody, err := util.HTTPRequest(url, string(data))
	if err != nil {
		log.Printf("HTTPRequest failed:%v", err)
		return infos[:0]
	}

	js, _ := simplejson.NewJson([]byte(`{}`))
	err = js.UnmarshalJSON([]byte(rspbody))
	retcode, err := js.Get("retcode").Int()
	if err != nil {
		log.Printf("get retcode failed:%v", err)
		return infos[:0]
	}

	if retcode != 0 {
		errmsg, _ := js.Get("errmsg").String()
		log.Printf("get ap info failed:%s", errmsg)
		return infos[:0]
	}

	arr, err := js.Get("data").Get("infos").Array()
	for i := 0; i < len(arr); i++ {
		var info APInfo
		tmp := js.Get("infos").GetIndex(i)
		info.aid, _ = tmp.Get("aid").String()
		info.ssid, _ = tmp.Get("ssid").String()
		info.longitude, _ = tmp.Get("longitude").String()
		info.latitude, _ = tmp.Get("latitude").String()
		info.address, _ = tmp.Get("address").String()
		infos[i] = info
	}

	return infos[:len(arr)]
}

//GetRealTimeInfo get realtime info
func GetRealTimeInfo(aid string) (RealInfo, error) {
	var realinfo RealInfo
	infos := make([]UserInfo, util.MaxListSize)
	data, err := genReqbody(map[string]string{"aid": aid})

	url := baseurl + "/apInfoList"
	rspbody, err := util.HTTPRequest(url, string(data))
	if err != nil {
		log.Printf("HTTPRequest failed:%v", err)
		return realinfo, err
	}

	js, _ := simplejson.NewJson([]byte(`{}`))
	err = js.UnmarshalJSON([]byte(rspbody))
	retcode, err := js.Get("retcode").Int()
	if err != nil {
		log.Printf("get retcode failed:%v", err)
		return realinfo, err
	}

	if retcode != 0 {
		errmsg, err := js.Get("errmsg").String()
		log.Printf("get ap info failed:%s", errmsg)
		return realinfo, err
	}
	realinfo.bandwidth, _ = js.Get("data").Get("bandwidth").String()
	realinfo.online, _ = js.Get("data").Get("online").Int()

	arr, _ := js.Get("data").Get("users").Array()
	i := 0
	for ; i < len(arr); i++ {
		tmp := js.Get("data").Get("users").GetIndex(i)
		var info UserInfo
		info.username, _ = tmp.Get("username").String()
		info.phone, _ = tmp.Get("phone").String()
		info.mac, _ = tmp.Get("mac").String()
		infos[i] = info
	}
	realinfo.infos = infos[:i]

	return realinfo, nil
}

//GetAPStat fetch ap stat info
func GetAPStat(aid, start, end string) (count int, traffic string) {
	data, err := genReqbody(map[string]string{"aid": aid, "start": start, "end": end})

	url := baseurl + "/statistics"
	rspbody, err := util.HTTPRequest(url, string(data))
	if err != nil {
		log.Printf("HTTPRequest failed:%v", err)
		return
	}

	js, _ := simplejson.NewJson([]byte(`{}`))
	err = js.UnmarshalJSON([]byte(rspbody))
	retcode, err := js.Get("retcode").Int()
	if err != nil {
		log.Printf("get retcode failed:%v", err)
		return
	}

	if retcode != 0 {
		errmsg, _ := js.Get("errmsg").String()
		log.Printf("get ap info failed:%s", errmsg)
		return
	}

	count, _ = js.Get("data").Get("count").Int()
	traffic, _ = js.Get("data").Get("traffic").String()
	return
}

//GetOnlineRecords get user online records
func GetOnlineRecords(username, start, end string) []OnlineRecord {
	records := make([]OnlineRecord, util.MaxListSize)
	data, err := genReqbody(map[string]string{"username": username, "start": start, "end": end})

	url := baseurl + "/onlineRecord"
	rspbody, err := util.HTTPRequest(url, string(data))
	if err != nil {
		log.Printf("HTTPRequest failed:%v", err)
		return records[:0]
	}

	js, _ := simplejson.NewJson([]byte(`{}`))
	err = js.UnmarshalJSON([]byte(rspbody))
	retcode, err := js.Get("retcode").Int()
	if err != nil {
		log.Printf("get retcode failed:%v", err)
		return records[:0]
	}

	if retcode != 0 {
		errmsg, _ := js.Get("errmsg").String()
		log.Printf("get ap info failed:%s", errmsg)
		return records[:0]
	}

	arr, _ := js.Get("data").Get("infos").Array()
	i := 0
	for ; i < len(arr); i++ {
		tmp := js.Get("data").Get("infos").GetIndex(i)
		var rec OnlineRecord
		rec.aid, _ = tmp.Get("aid").String()
		rec.start, _ = tmp.Get("start").String()
		rec.end, _ = tmp.Get("end").String()
		records[i] = rec
	}

	return records[:i]
}
