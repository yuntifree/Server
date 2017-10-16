package juhe

import (
	"Server/aliyun"
	"Server/util"
	"bytes"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const (
	baseURL = "http://cpc.people.com.cn"
)

var targets = []string{"http://cpc.people.com.cn/GB/67481/412690/414402/index.html",
	"http://cpc.people.com.cn/GB/67481/412690/414114/index.html",
	"http://cpc.people.com.cn/GB/67481/412690/412747/index.html",
	"http://cpc.people.com.cn/GB/67481/412690/413271/index.html",
	"http://cpc.people.com.cn/GB/67481/412690/413204/index.html",
	"http://cpc.people.com.cn/GB/67481/412690/412964/index.html",
	"http://cpc.people.com.cn/GB/67481/412690/413654/index.html",
	"http://cpc.people.com.cn/GB/67481/412690/413308/index.html",
	"http://cpc.people.com.cn/GB/67481/412690/413943/index.html",
	"http://cpc.people.com.cn/GB/67481/412690/414240/index.html"}

//GetTopicNews get 19th topic news
func GetTopicNews(url string) []News {
	var news []News
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Add("User-Agent", "Mozilla/5.0 (Linux; Android 6.0; Nexus 5 Build/MRA58N) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/46.0.2490.76 Mobile Safari/537.36")
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("GetHBNews request failed:%v", err)
		return news
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("read response body failed:%v", err)
	}

	d, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		log.Printf("NewDocumentFromReader failed:%v", err)
		return news
	}

	d.Find(".p2j_con02 .fl li").Each(func(i int, n *goquery.Selection) {
		if href, ok := n.Find("a").Attr("href"); ok {
			log.Printf("href:%s", href)
			url := baseURL + href
			ns, err := getTopicNewsInfo(url)
			if err != nil {
				log.Printf("getTopicNewsInfo failed:%v %s", err, url)
			} else {
				log.Printf("news info:%+v", ns)
				news = append(news, ns)
			}
		}
	})

	return news
}

func getTopicNewsInfo(url string) (news News, err error) {
	news.Origin = url
	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("User-Agent", "Mozilla/5.0 (Linux; Android 6.0; Nexus 5 Build/MRA58N) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/46.0.2490.76 Mobile Safari/537.36")
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("get url failed:%v", err)
		return news, err
	}
	defer resp.Body.Close()

	d, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Printf("NewDocumentFromReader failed:%v", err)
		return news, err
	}

	title := d.Find(".text_con01 .text_c h1").Text()
	title = string(gbToUtf8([]byte(title)))
	news.Title = title
	news.Md5 = util.GetMD5Hash(news.Title)
	d.Find("meta").Each(func(i int, s *goquery.Selection) {
		name, _ := s.Attr("name")
		content, _ := s.Attr("content")
		if name == "publishdate" {
			log.Printf("publish date:%s", content)
			news.Date = content
		} else if name == "source" {
			content = string(gbToUtf8([]byte(content)))
			arr := strings.Split(content, "：")
			if len(arr) >= 2 {
				news.Author = arr[1]
			}
		}
	})

	var images []string
	var content []Content
	d.Find(".show_text .pic_c").Each(func(i int, n *goquery.Selection) {
		if img, ok := n.Find("img").Attr("src"); ok {
			log.Printf("img:%s", img)
			images = append(images, img)
			var cont Content
			cont.Type = typeImg
			cont.Src = img
			content = append(content, cont)
		}
	})
	d.Find(".show_text p").Each(func(i int, n *goquery.Selection) {
		if img, ok := n.Find("img").Attr("src"); ok {
			image := img
			images = append(images, image)
			var cont Content
			cont.Type = typeImg
			cont.Src = image
			content = append(content, cont)
		} else {
			txt := n.Text()
			var cont Content
			cont.Type = typeText
			txt = string(gbToUtf8([]byte(txt)))
			cont.Src = txt
			content = append(content, cont)
		}
	})

	tpl, err := template.ParseFiles("/data/server/templates/content.html")
	if err != nil {
		log.Printf("parse template failed")
		return news, err
	}
	var buf bytes.Buffer
	w := io.Writer(&buf)
	err = tpl.Execute(w, &DgPage{Title: news.Title, Source: news.Author, Ctime: news.Date, Infos: content})
	filename := util.GenSalt() + ".html"
	if flag := aliyun.UploadOssFile(filename, buf.String()); !flag {
		log.Printf("UploadOssFile failed %s:%v", filename, err)
		return news, err
	}
	log.Printf("UploadOssFile success: %s", filename)
	news.URL = aliyun.GenOssNewsURL(filename)

	for i := 0; i < 3 && i < len(images); i++ {
		news.Pics[i] = images[i]
	}

	news.Date = extractDate(news.Date)
	return news, nil
}
