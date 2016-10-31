package juhe

import (
	"log"
	"strconv"

	"../util"

	simplejson "github.com/bitly/go-simplejson"
)

const (
	youkuPlay = "http://tv.uc.cn/player/youku/1.0.0/?client_id=a0ae6a083d6ac59a"
	letvPlay  = "http://minisite.letv.com/tuiguang/index.shtml?&typeFrom=uc&ref=uc&ark=372"
	youkuURL  = "http://vibll.tv.uc.cn/mobile/page/channel_short/1.1.1?&platform=1&order=5&mode=3&genres=搞笑&source=优酷&sub_genres=&three_genres=&data_type=1&sub_id=520"
)

//YoukuFile youku video file information
type YoukuFile struct {
	ImgURL, PlayURL, Duration, ID, OriginID, Title, Source string
}

//GenYoukuURL generate youku play url
func GenYoukuURL(vid string) string {
	return youkuPlay + "&vid=" + vid
}

//GenLetvURL generate letv play url
func GenLetvURL(vid string) string {
	return letvPlay + "&vid=" + vid
}

//GetYoukuFiles return information of youku video files
func GetYoukuFiles(start, size int) []YoukuFile {
	var files []YoukuFile
	url := youkuURL + "&start=" + strconv.Itoa(start) + "&size=" + strconv.Itoa(size)
	resp, err := util.HTTPRequest(url, "")
	if err != nil {
		log.Printf("HTTPRequest url %s failed:%v", url, err)
		return files
	}

	js, err := simplejson.NewJson([]byte(resp))
	if err != nil {
		log.Printf("parse json failed:%v", err)
		return files
	}

	arr, err := js.Get("data").Get("short_list").Array()
	if err != nil {
		log.Printf("get short_list failed:%v", err)
		return files
	}

	for i := 0; i < len(arr); i++ {
		json := js.Get("data").Get("short_list").GetIndex(i)
		var info YoukuFile
		info.ImgURL, _ = json.Get("img_url").String()
		info.PlayURL, _ = json.Get("play_url").String()
		info.Duration, _ = json.Get("duration").String()
		info.ID, _ = json.Get("id").String()
		info.OriginID, _ = json.Get("origin_id").String()
		info.Title, _ = json.Get("title").String()
		info.Source, _ = json.Get("source").String()
		files = append(files, info)
	}

	return files
}
