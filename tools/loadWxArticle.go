package main

import (
	"Server/util"
	"bufio"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
	simplejson "github.com/bitly/go-simplejson"
)

const (
	baseUrl    = "http://www.gsdata.cn/rank/single?id="
	contentUrl = "http://www.gsdata.cn/rank/recommendArticles?type=hot&id="
	imgUrl     = "http://img1.gsdata.cn/index.php/rank/getImageUrl?hash="
)

//WxData information for wxData
type WxInfo struct {
	Id, // wxid
	Name, // 微信名称
	Desc, // 公众号描述
	HeadUrl string // 公众号头像
}

type WxContent struct {
	Title, // 文章标题
	Desc, // 文章描述
	Image, // 文章图片：aliyun
	PostTime, // 发布时间
	PostUrl, // 文章地址
	HomeUrl string // 公众号主页，点击进入公众号用
}

var ids = [][2]string{
	{"MxTuEex5N2T1Mtxa", "111531"},                  // OurDongguan
	{"MxTuQe25N2D1ctyaNnggO0O0O#O0O0O4", "1464726"}, // 东莞大喇叭
	{"MxTuYew5M2z1Qt5aNnwgO0O0O#O0O0O4", "1603497"}, // 东莞美食先锋队
	{"MxTuAe35O2T1ktxa", "107991"},                  // 东莞时间网
	{"MxTuUez5N2T1YtxaNnAgO0O0O#O0O0O4", "1535614"}, // 东莞生活君
	{"MxTuMe35M2z1ctza", "137373"},                  // 东莞美食大搜罗
	{"MxTuAe35O2T1gtxa", "107981"},                  // 东莞阳光网
	{"MxTuIe45M2j1gtO0O0Oa", "12828"},               // 东莞日报
	{"MxTuEex5N2T1Mtwa", "111530"},                  // 流行东莞
	{"MxTuUez5M2j1EtzaNnQgO0O0O#O0O0O4", "1532135"}, // 差评
	{"Mxjuce55O2A1O0O0OtO0O0Oa", "2798"},            // 虎嗅网
	{"MxTuUe05O2D1QtyaMnwgO0O0O#O0O0O4", "1548423"}, // 超高能E姐
	{"MxjuceO0O0O5", "27"},                          // Vista看天下
	{"MxzuYe35O2D1Ut5a", "367859"},                  // 同道大叔
	{"MxTuAe35N2T1Ytya", "107562"},                  // 人民日报
	{"MxTuce25M2z1gtO0O0Oa", "17638"},               // 深夜发媸
	{"NxTuQez5N2T1Itwa", "543520"},                  // 二更食堂
	{"MxTuQez5", "143"},                             // 十点读书
}

func getWxInfo(id [2]string, w *bufio.Writer) WxInfo {
	var wxData WxInfo
	client := &http.Client{}
	req, err := http.NewRequest("GET", baseUrl+id[0], nil)
	req.Header.Add("User-Agent", "Mozilla/5.0 (Linux; Android 6.0; Nexus 5 Build/MRA58N) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/46.0.2490.76 Mobile Safari/537.36")
	resp, err := client.Do(req)
	defer resp.Body.Close()

	d, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Printf("fetch url failed:%v", err)
		return wxData
	}

	div := d.Find(".number-txt")
	wxData.Id = div.Find("#wx_name").Text()
	wxData.Name = d.Find(".number-title #wx_nickname").Text()
	wxData.Desc = div.Find(".wx-sp").Eq(1).Find(".sp-txt").Text()
	wxData.HeadUrl = "http://open.weixin.qq.com/qr/code/?username=" + wxData.Id

	w.WriteString(fmtContent(wxData, getWxContent(id[1])))
	return wxData
}

func getWxContent(id string) WxContent {
	var wxContent WxContent
	client := &http.Client{}
	req, err := http.NewRequest("GET", contentUrl+id, nil)
	req.Header.Add("X-Requested-With", "XMLHttpRequest")

	if err != nil {
		log.Printf("fetch content failed:%v", err)
		return wxContent
	}

	resp, err := client.Do(req)
	defer resp.Body.Close()

	js, err := simplejson.NewFromReader(resp.Body)
	if err != nil {
		log.Printf("parse rspbody failed:%v", err)
		return wxContent
	}

	ret, _ := js.Get("status").String()
	if ret != "OK" {
		log.Printf("status not ok: %v", ret)
	}
	// get article and images, 取得最近一篇文章
	// TODO: check empty items
	article := js.Get("result").Get("items").GetIndex(0)

	// post infos
	wxContent.PostTime, _ = article.Get("posttime").String()
	postUrl, _ := article.Get("url").String()
	wxContent.PostUrl = postUrl

	// 获取__biz 得到主页url
	start := strings.Index(postUrl, "__biz=") + 6
	end := strings.Index(postUrl, "&mid=")

	wxContent.HomeUrl = "https://mp.weixin.qq.com/mp/profile_ext?action=home&scene=116&__biz=" + string(postUrl[start:end]) + "#wechat_redirect"
	wxContent.Title, _ = article.Get("title").String()
	wxContent.Desc, _ = article.Get("content").String()

	// fetch aliyun imageurl
	picurl, _ := article.Get("picurl").String()

	wxContent.Image = getImageUrl(picurl)

	return wxContent
}

func getImageUrl(qurl string) string {
	client := &http.Client{}
	req, err := http.NewRequest("GET", imgUrl+qurl, nil)

	if err != nil {
		log.Printf("fetch image url failed:%v", err)
		return ""
	}

	resp, err := client.Do(req)
	defer resp.Body.Close()

	js, err := simplejson.NewFromReader(resp.Body)
	if err != nil {
		log.Printf("parse url json failed:%v", err)
		return ""
	}

	ret, _ := js.Get("url").String()

	return ret
}

func checkerr(e error) {
	if e != nil {
		panic(e)
	}
}

func fmtHeader() string {
	return fmt.Sprintln("wxid,wx_name,wx_desc,wx_head,wx_home,title,desc,img,url,posttime")
}

func fmtContent(wxInfo WxInfo, wxContent WxContent) string {
	return fmt.Sprintf("%v||%v||%v||%v||%v||%v||%v||%v||%v||%v\n", wxInfo.Id, wxInfo.Name, wxInfo.Desc, wxInfo.HeadUrl,
		wxContent.HomeUrl, wxContent.Title, wxContent.Desc, wxContent.Image, wxContent.PostUrl, wxContent.PostTime)
}

func main() {
	db, err := util.InitDB(false)
	if err != nil {
		panic(err)
	}

}
