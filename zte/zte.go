package zte

import (
	"log"

	"Server/util"

	simplejson "github.com/bitly/go-simplejson"
)

const (
	baseurl = "http://120.234.130.195:19000"
)

//APInfo ap base information
type APInfo struct {
	Aid                 int
	Address, Mac        string
	Longitude, Latitude float64
}

//UserInfo online user information
type UserInfo struct {
	Username, Phone, Mac string
}

//RealInfo realtime information
type RealInfo struct {
	Bandwidth string
	Online    int
	Infos     []UserInfo
}

//OnlineRecord user online record
type OnlineRecord struct {
	Aid                 int
	Start, End, Traffic string
}

func genReqbody(reqinfos map[string]interface{}) (string, error) {
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
func GetAPInfoList(seq int) []APInfo {
	infos := make([]APInfo, util.MaxListSize)
	url := baseurl + "/apInfoList"
	data, err := genReqbody(map[string]interface{}{"seq": seq})
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
		tmp := js.Get("data").Get("infos").GetIndex(i)
		info.Aid, _ = tmp.Get("aid").Int()
		info.Longitude, _ = tmp.Get("longitude").Float64()
		info.Latitude, _ = tmp.Get("latitude").Float64()
		info.Address, _ = tmp.Get("address").String()
		info.Mac, _ = tmp.Get("apmac").String()
		log.Printf("get %d %f %f %s %s", info.Aid, info.Longitude, info.Latitude, info.Address, info.Mac)
		infos[i] = info
	}

	return infos[:len(arr)]
}

//GetRealTimeInfo get realtime info
func GetRealTimeInfo(aid int) (RealInfo, error) {
	var realinfo RealInfo
	infos := make([]UserInfo, util.MaxListSize)
	data, err := genReqbody(map[string]interface{}{"aid": aid})

	url := baseurl + "/realTime"
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
	realinfo.Bandwidth, _ = js.Get("data").Get("bandwidth").String()
	realinfo.Online, _ = js.Get("data").Get("online").Int()

	arr, _ := js.Get("data").Get("users").Array()
	i := 0
	for ; i < len(arr); i++ {
		tmp := js.Get("data").Get("users").GetIndex(i)
		var info UserInfo
		info.Username, _ = tmp.Get("username").String()
		info.Phone, _ = tmp.Get("phone").String()
		info.Mac, _ = tmp.Get("mac").String()
		infos[i] = info
	}
	realinfo.Infos = infos[:i]

	return realinfo, nil
}

//GetAPStat fetch ap stat info
func GetAPStat(aid int, start string, end string) (count int, traffic string) {
	data, err := genReqbody(map[string]interface{}{"aid": aid, "start": start, "end": end})

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
	data, err := genReqbody(map[string]interface{}{"username": username, "start": start, "end": end})

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
		rec.Aid, _ = tmp.Get("aid").Int()
		rec.Start, _ = tmp.Get("start").String()
		rec.End, _ = tmp.Get("end").String()
		rec.Traffic, _ = tmp.Get("traffic").String()
		records[i] = rec
	}

	return records[:i]
}
