package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
	simplejson "github.com/bitly/go-simplejson"
)

const (
	baseUrl    = "http://www.gsdata.cn/rank/wxdetail?wxname="
	contentUrl = "http://www.gsdata.cn/rank/toparc?sort=-1&wxname="
	imgUrl     = "http://img1.gsdata.cn/index.php/rank/getimageurl?callback=_&hash="
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

var ids = []string{
	"mimeng7",
}

var limit chan int

func getWxInfo(id string, ch chan string) WxInfo {
	limit <- 1
	var wxData WxInfo
	client := &http.Client{}
	req, err := http.NewRequest("GET", baseUrl+id, nil)
	req.Header.Add("User-Agent", "Mozilla/5.0 (Linux; Android 6.0; Nexus 5 Build/MRA58N) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/46.0.2490.76 Mobile Safari/537.36")
	resp, err := client.Do(req)
	defer resp.Body.Close()

	d, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Printf("fetch url failed:%v", err)
		return wxData
	}

	div := d.Find(".wxDetail-top")
	wxData.Id = id
	wxData.Name = d.Find(".wxDetail-name label").Text()
	wxData.Desc = div.Find(".info-li p").Last().Text()
	wxData.HeadUrl = "http://open.weixin.qq.com/qr/code/?username=" + wxData.Id

	<-limit
	ch <- fmtContent(wxData, getWxContent(id))

	return wxData
}

func getWxContent(id string) []WxContent {
	client := &http.Client{}
	req, err := http.NewRequest("GET", contentUrl+id, nil)
	req.Header.Add("X-Requested-With", "XMLHttpRequest")

	if err != nil {
		log.Printf("fetch content failed:%v", err)
		return nil
	}

	resp, err := client.Do(req)
	defer resp.Body.Close()

	js, err := simplejson.NewFromReader(resp.Body)
	if err != nil {
		log.Printf("parse rspbody failed:%v", err)
		return nil
	}

	ret, _ := js.Get("error").Int()
	if ret != 0 {
		log.Printf("status not ok: %v", ret)
		return nil
	}
	// get article and images, 取得最近一篇文章
	// TODO: check empty items
	artlist, _ := js.Get("data").Array()
	artlen := len(artlist)
	items := make([]WxContent, artlen)

	for i := 0; i < artlen; i++ {
		article := js.Get("data").GetIndex(i)
		// post infos
		items[i].PostTime, _ = article.Get("posttime").String()
		postUrl, _ := article.Get("url").String()
		items[i].PostUrl = postUrl

		// 获取__biz 得到主页url
		start := strings.Index(postUrl, "__biz=") + 6
		end := strings.Index(postUrl, "&mid=")

		items[i].HomeUrl = "https://mp.weixin.qq.com/mp/profile_ext?action=home&scene=116&__biz=" + string(postUrl[start:end]) + "#wechat_redirect"
		items[i].Title, _ = article.Get("title").String()
		items[i].Desc, _ = article.Get("content").String()

		// fetch aliyun imageurl
		picurl, _ := article.Get("data-hash").String()

		items[i].Image = getImageUrl(url.QueryEscape(picurl))
	}

	return items
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

	str, err := ioutil.ReadAll(resp.Body)
	if checkerr(err) {
		return ""
	}

	js, err := simplejson.NewJson([]byte(str[2 : len(str)-1]))
	if err != nil {
		log.Printf("parse url json failed:%v", err)
		return ""
	}

	ret, _ := js.Get("url").String()

	return ret
}

func checkerr(e error) bool {
	if e != nil {
		panic(e)
		return true
	}
	return false
}

func fmtHeader() string {
	return fmt.Sprintln("wxid,wx_name,wx_desc,wx_head,wx_home,title,desc,img,url,posttime")
}

func fmtContent(wxInfo WxInfo, wxList []WxContent) string {
	ret := ""
	for _, item := range wxList {
		ret += fmt.Sprintf("%v||%v||%v||%v||%v||%v||%v||%v||%v||%v\n", wxInfo.Id, wxInfo.Name, wxInfo.Desc, wxInfo.HeadUrl,
			item.HomeUrl, item.Title, item.Desc, item.Image, item.PostUrl, item.PostTime)
	}
	return ret
}

func main() {

	f, err := os.Create("wxData.csv")
	checkerr(err)
	defer f.Close()

	fmt.Println(fmtHeader())
	w := bufio.NewWriter(f)

	limit = make(chan int, 2)
	chs := make([]chan string, len(ids))
	for i, v := range ids {
		chs[i] = make(chan string)
		go getWxInfo(v, chs[i])
	}
	content := ""
	for _, ch := range chs {
		content = <-ch
		w.WriteString(content)
		fmt.Println(content)
	}
	w.Flush()
}
