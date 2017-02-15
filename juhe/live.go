package juhe

import (
	"Server/util"
	"fmt"
	"log"
	"strings"

	simplejson "github.com/bitly/go-simplejson"
)

const (
	liveURL = "http://web.free.wifi.360.cn/internet/huajiao"
)

//LiveInfo for 360 live info
type LiveInfo struct {
	Uid      int64
	Avatar   string
	Nickname string
	LiveId   int64
	Img      string
	PTime    string
	Location string
	Watches  int64
	Live     int64
}

//GetLiveInfo get live info from 360
func GetLiveInfo(offset int64) ([]*LiveInfo, int64) {
	var infos []*LiveInfo
	url := fmt.Sprintf("%s?offset=%d", liveURL, offset)
	resp, err := util.HTTPRequest(url, "")
	if err != nil {
		log.Printf("GetLiveInfo HTTPRequest failed:%v", err)
		return infos, 0
	}
	if len(resp) <= 2 {
		log.Printf("GetLiveInfo illegal resp:%s %v", resp, err)
		return infos, 0
	}
	lpos := strings.Index(resp, "(")
	rpos := strings.LastIndex(resp, ")")
	data := string(resp[lpos+1 : rpos])
	js, err := simplejson.NewJson([]byte(data))
	if err != nil {
		log.Printf("GetLiveInfo parse failed:%s %v", data, err)
		return infos, 0
	}

	arr, err := js.Get("data").Get("list").Array()
	if err != nil {
		log.Printf("GetLiveInfo get list failed:%v", err)
		return infos, 0
	}

	for i := 0; i < len(arr); i++ {
		json := js.Get("data").Get("list").GetIndex(i)
		var info LiveInfo
		info.Uid, _ = json.Get("uid").Int64()
		info.Avatar, _ = json.Get("avatar").String()
		info.Nickname, _ = json.Get("nickname").String()
		if info.Nickname == "" {
			info.Nickname = "主播"
		}
		info.LiveId, _ = json.Get("live_id").Int64()
		info.PTime, _ = json.Get("p_time").String()
		info.Location, _ = json.Get("location").String()
		if info.Location == "" {
			info.Location = "难道在火星？"
		}
		info.Watches, _ = json.Get("watches").Int64()
		info.Live, _ = json.Get("live").Int64()
		info.Img, _ = json.Get("img").String()
		infos = append(infos, &info)
	}
	more, _ := js.Get("data").Get("more").Int64()
	var seq int64
	if more != 0 {
		seq, _ = js.Get("data").Get("offset").Int64()
	}

	log.Printf("seq:%d", seq)
	return infos, seq
}
