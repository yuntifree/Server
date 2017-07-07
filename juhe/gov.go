package juhe

import (
	"Server/aliyun"
	"Server/util"
	"bytes"
	"encoding/xml"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	iconv "github.com/djimenez/iconv-go"
)

const (
	gdURL     = "http://wgx.dg.gov.cn/publicfiles/business/htmlfiles/dgwhj/s35233/list.htm"
	gdBase    = "http://wgx.dg.gov.cn/publicfiles/business/htmlfiles/"
	gdImgBase = "http://wgx.dg.gov.cn"
	fgURL     = "http://dgdp.dg.gov.cn/business/htmlfiles/dgfg/pxwzx/list.htm"
	fgBase    = "http://dgdp.dg.gov.cn"
	hbURL     = "http://dgepb.dg.gov.cn/business/htmlfiles/dgepb/dgdt/list.htm"
	hbBase    = "http://dgepb.dg.gov.cn/business/htmlfiles/"
	wjURL     = "http://www.dgwsj.gov.cn/304219773/0802/wjjlist2.shtml"
	wjBase    = "http://www.dgwsj.gov.cn"
	jyURL     = "http://www.dgjy.net/moreInfo.aspx?menuId=679"
	jyBase    = "http://www.dgjy.net"
	jcURL     = "http://www.gddg110.gov.cn/publicfiles/business/htmlfiles/dgjch/s14345/list.htm"
)

type gdInfo struct {
	XMLName       xml.Name `xml:"INFO"`
	Title         string   `xml:"Title"`
	PublishedTime string   `xml:"PublishedTime"`
	InfoURL       string   `xml:"InfoURL"`
}

type gdRecs struct {
	XMLName   xml.Name `xml:"RECS"`
	TotalRecs string   `xml:"totalRecs,attr"`
	Infos     []gdInfo `xml:"INFO"`
}

type gdContent struct {
	XMLName xml.Name `xml:"xml"`
	Id      string   `xml:"id,attr"`
	Recs    gdRecs   `xml:"RECS"`
}

func isExpire(date string) bool {
	if date == "" {
		return true
	}
	arr := strings.Split(date, "-")
	if len(arr) > 1 {
		year, err := strconv.Atoi(arr[0])
		if err != nil {
			return true
		}
		if year < 2017 {
			return true
		}
	}
	return false
}

//GetHBNews get dongguan huanbaoju news
func GetHBNews() []News {
	var news []News
	client := &http.Client{}
	req, err := http.NewRequest("GET", hbURL, nil)
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

	str := string(body)
	start := strings.Index(str, `<?xml`)
	end := strings.Index(str, `</xml>`)
	var gd gdContent
	if start != -1 && end != -1 {
		content := str[start : end+6]
		err := xml.Unmarshal([]byte(content), &gd)
		if err != nil {
			log.Printf("xml Unmarshal failed:%v", err)
		}
	}
	infos := gd.Recs.Infos
	for i := 0; i < len(infos); i++ {
		url := hbBase + infos[i].InfoURL
		log.Printf("url:%s", url)
		info, err := getHBNewsInfo(url)
		if err != nil {
			log.Printf("getHBNewsInfo failed:%s %v", url, err)
			continue
		}
		if isExpire(info.Date) {
			break
		}
		news = append(news, info)
	}

	return news
}

func getHBNewsInfo(url string) (news News, err error) {
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

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("getFgNewsInfo ReadAll failed:%v", err)
		return news, err
	}

	d, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		log.Printf("NewDocumentFromReader failed:%v", err)
		return news, err
	}
	str := string(body)
	date := extractTime(str)
	news.Date = date

	title := d.Find(".conert_tt").Text()
	news.Title = title
	news.Md5 = util.GetMD5Hash(news.Title)
	log.Printf("title:%s date:%s", news.Title, news.Date)

	var images []string
	var content []Content
	d.Find(".conert_nr  P").Each(func(i int, n *goquery.Selection) {
		if img, ok := n.Find("IMG").Attr("src"); ok {
			image := fgBase + img
			images = append(images, image)
			var cont Content
			cont.Type = typeImg
			cont.Src = image
			content = append(content, cont)
		} else {
			txt := n.Text()
			var cont Content
			cont.Type = typeText
			cont.Src = txt
			content = append(content, cont)
		}
	})

	log.Printf("images:%+v content:%+v", images, content)
	tpl, err := template.ParseFiles("/data/server/templates/content.html")
	if err != nil {
		log.Printf("parse template failed")
		return news, err
	}
	var buf bytes.Buffer
	w := io.Writer(&buf)
	err = tpl.Execute(w, &DgPage{Title: news.Title, Source: "东莞市环境保护局", Ctime: news.Date, Infos: content})
	filename := util.GenSalt() + ".html"
	if flag := aliyun.UploadOssFile(filename, buf.String()); !flag {
		log.Printf("UploadOssFile failed %s:%v", filename, err)
		return news, err
	}
	log.Printf("UploadOssFile success: %s", filename)
	news.URL = aliyun.GenOssNewsURL(filename)
	news.Author = "东莞市环境保护局"

	for i := 0; i < 3 && i < len(images); i++ {
		news.Pics[i] = images[i]
	}

	news.Date = extractDate(news.Date)
	return news, nil
	return
	return
}

//GetFGNews get dongguan fagaiwei news
func GetFGNews() []News {
	var news []News
	client := &http.Client{}
	req, err := http.NewRequest("GET", fgURL, nil)
	req.Header.Add("User-Agent", "Mozilla/5.0 (Linux; Android 6.0; Nexus 5 Build/MRA58N) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/46.0.2490.76 Mobile Safari/537.36")
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("GetFGNews request failed:%v", err)
		return news
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("read response body failed:%v", err)
	}

	str := string(body)
	start := strings.Index(str, `<?xml`)
	end := strings.Index(str, `</xml>`)
	var gd gdContent
	if start != -1 && end != -1 {
		content := str[start : end+6]
		err := xml.Unmarshal([]byte(content), &gd)
		if err != nil {
			log.Printf("xml Unmarshal failed:%v", err)
		}
	}
	infos := gd.Recs.Infos
	for i := 0; i < len(infos); i++ {
		url := gdBase + infos[i].InfoURL
		log.Printf("url:%s", url)
		info, err := getFgNewsInfo(url)
		if err != nil {
			log.Printf("getFgNewsInfo failed:%s %v", url, err)
			continue
		}
		if isExpire(info.Date) {
			break
		}
		news = append(news, info)
	}

	return news
}

func extractTime(str string) string {
	pos := strings.Index(str, "发布日期：")
	if pos != -1 {
		log.Printf("pos:%d", pos)
		substr := str[pos:]
		end := strings.Index(substr, "<")
		if end != -1 {
			tstr := substr[:end]
			log.Printf("tstr:%s", tstr)
			start := strings.Index(tstr, "：")
			if start != -1 {
				content := tstr[start+3:]
				content = cleanText(content)
				return content
			}
		}
	}
	return ""
}

func getFgNewsInfo(url string) (news News, err error) {
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

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("getFgNewsInfo ReadAll failed:%v", err)
		return news, err
	}

	d, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		log.Printf("NewDocumentFromReader failed:%v", err)
		return news, err
	}
	str := string(body)
	date := extractTime(str)
	news.Date = date

	title := d.Find(".ttbg").Text()
	news.Title = title
	news.Md5 = util.GetMD5Hash(news.Title)
	log.Printf("title:%s date:%s", news.Title, news.Date)

	var images []string
	var content []Content
	d.Find(".tzxq  P").Each(func(i int, n *goquery.Selection) {
		if img, ok := n.Find("IMG").Attr("src"); ok {
			image := fgBase + img
			images = append(images, image)
			var cont Content
			cont.Type = typeImg
			cont.Src = image
			content = append(content, cont)
		} else {
			txt := n.Text()
			var cont Content
			cont.Type = typeText
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
	err = tpl.Execute(w, &DgPage{Title: news.Title, Source: "东莞市发展和改革局", Ctime: news.Date, Infos: content})
	filename := util.GenSalt() + ".html"
	if flag := aliyun.UploadOssFile(filename, buf.String()); !flag {
		log.Printf("UploadOssFile failed %s:%v", filename, err)
		return news, err
	}
	log.Printf("UploadOssFile success: %s", filename)
	news.URL = aliyun.GenOssNewsURL(filename)
	news.Author = "东莞市发展和改革局"

	for i := 0; i < 3 && i < len(images); i++ {
		news.Pics[i] = images[i]
	}

	news.Date = extractDate(news.Date)
	return news, nil
	return
}

//GetGDNews get donguan guangdianju news
func GetGDNews() []News {
	var news []News
	client := &http.Client{}
	req, err := http.NewRequest("GET", gdURL, nil)
	req.Header.Add("User-Agent", "Mozilla/5.0 (Linux; Android 6.0; Nexus 5 Build/MRA58N) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/46.0.2490.76 Mobile Safari/537.36")
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("GetGDNews request failed:%v", err)
		return news
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("read response body failed:%v", err)
	}
	str := string(body)
	start := strings.Index(str, `<?xml`)
	end := strings.Index(str, `</xml>`)
	var gd gdContent
	if start != -1 && end != -1 {
		content := str[start : end+6]
		err := xml.Unmarshal([]byte(content), &gd)
		if err != nil {
			log.Printf("xml Unmarshal failed:%v", err)
		}
	}
	infos := gd.Recs.Infos
	for i := 0; i < len(infos); i++ {
		url := gdBase + infos[i].InfoURL
		log.Printf("url:%s", url)
		info, err := getGdNewsInfo(url)
		if err != nil {
			log.Printf("getGdNewsInfo failed:%s %v", url, err)
			continue
		}
		if isExpire(info.Date) {
			break
		}
		news = append(news, info)
	}

	return news
}

func getGdNewsInfo(url string) (news News, err error) {
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

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("getFgNewsInfo ReadAll failed:%v", err)
		return news, err
	}

	d, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		log.Printf("NewDocumentFromReader failed:%v", err)
		return news, err
	}
	str := string(body)
	date := extractTime(str)
	news.Date = date

	title := d.Find("title").Text()
	news.Title = title
	news.Md5 = util.GetMD5Hash(news.Title)
	log.Printf("title:%s date:%s", news.Title, news.Date)

	var images []string
	var content []Content
	d.Find(".concen_04 font P").Each(func(i int, n *goquery.Selection) {
		if img, ok := n.Find("img").Attr("src"); ok {
			image := gdImgBase + img
			images = append(images, image)
			var cont Content
			cont.Type = typeImg
			cont.Src = image
			content = append(content, cont)
		} else {
			txt := n.Text()
			var cont Content
			cont.Type = typeText
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
	err = tpl.Execute(w, &DgPage{Title: news.Title, Source: "东莞市文化广电新闻出版局", Ctime: news.Date, Infos: content})
	filename := util.GenSalt() + ".html"
	if flag := aliyun.UploadOssFile(filename, buf.String()); !flag {
		log.Printf("UploadOssFile failed %s:%v", filename, err)
		return news, err
	}
	log.Printf("UploadOssFile success: %s", filename)
	news.URL = aliyun.GenOssNewsURL(filename)
	news.Author = "东莞市文化广电新闻出版局"

	for i := 0; i < 3 && i < len(images); i++ {
		news.Pics[i] = images[i]
	}

	news.Date = extractDate(news.Date)
	return news, nil
	return
}

//GetJCNews get dongguan gongannju news
func GetJCNews() []News {
	var news []News
	client := &http.Client{}
	req, err := http.NewRequest("GET", jcURL, nil)
	req.Header.Add("User-Agent", "Mozilla/5.0 (Linux; Android 6.0; Nexus 5 Build/MRA58N) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/46.0.2490.76 Mobile Safari/537.36")
	resp, err := client.Do(req)
	defer resp.Body.Close()

	d, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Printf("fetch url failed:%v", err)
		return news
	}

	d.Find("tbody tr").Each(func(i int, n *goquery.Selection) {
		log.Printf("index:%d, node name:%+v", i, goquery.NodeName(n))
		src, _ := n.Find("a").Attr("href")
		log.Printf("src:%s", src)
		if src != "" {
			arr := strings.Split(src, "..")
			if len(arr) > 0 {
				src = "http://www.gddg110.gov.cn/publicfiles" + arr[len(arr)-1]
			}
			if src == jcURL {
				return
			}
			log.Printf("get src:%s", src)
			info, err := getJcNewsInfo(src)
			if err != nil {
				log.Printf("getJyNewsInfo failed:%s %v", src, err)
				return
			}
			if isExpire(info.Date) {
				return
			}
			news = append(news, info)
		}
	})

	return news
}

func getJcTime(str string) string {
	re := regexp.MustCompile(`\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}`)
	date := re.FindString(str)
	return date
}

func getJcNewsInfo(url string) (news News, err error) {
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

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("getFgNewsInfo ReadAll failed:%v", err)
		return news, err
	}

	d, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		log.Printf("NewDocumentFromReader failed:%v", err)
		return news, err
	}
	str := string(body)
	date := getJcTime(str)
	news.Date = date

	base := getURLBase(url)
	title := d.Find("tr td .title").Text()
	news.Title = title
	news.Md5 = util.GetMD5Hash(news.Title)
	log.Printf("title:%s date:%s", news.Title, news.Date)

	var images []string
	var content []Content
	d.Find(".content p").Each(func(i int, n *goquery.Selection) {
		if img, ok := n.Find("img").Attr("src"); ok {
			image := base + img
			images = append(images, image)
			var cont Content
			cont.Type = typeImg
			cont.Src = image
			content = append(content, cont)
		} else {
			txt := n.Text()
			var cont Content
			cont.Type = typeText
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
	err = tpl.Execute(w, &DgPage{Title: news.Title, Source: "东莞公安局", Ctime: news.Date, Infos: content})
	filename := util.GenSalt() + ".html"
	if flag := aliyun.UploadOssFile(filename, buf.String()); !flag {
		log.Printf("UploadOssFile failed %s:%v", filename, err)
		return news, err
	}
	log.Printf("UploadOssFile success: %s", filename)
	news.URL = aliyun.GenOssNewsURL(filename)
	news.Author = "东莞公安局"

	for i := 0; i < 3 && i < len(images); i++ {
		news.Pics[i] = images[i]
	}

	news.Date = extractDate(news.Date)
	return news, nil
	return
}

//GetJYNews get dongguan jiaoyuju news
func GetJYNews() []News {
	var news []News
	client := &http.Client{}
	req, err := http.NewRequest("GET", jyURL, nil)
	req.Header.Add("User-Agent", "Mozilla/5.0 (Linux; Android 6.0; Nexus 5 Build/MRA58N) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/46.0.2490.76 Mobile Safari/537.36")
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("GetJYNews request failed:%v", err)
		return news
	}
	defer resp.Body.Close()

	d, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Printf("fetch url failed:%v", err)
		return news
	}
	log.Printf("doc:%+v", d)

	d.Find(".nsy-news li").Each(func(i int, n *goquery.Selection) {
		log.Printf("index:%d, node name:%+v", i, goquery.NodeName(n))
		log.Printf("n:%+v", n)
		src, _ := n.Find("a").Attr("href")
		log.Printf("src:%s", src)
		if src != "" {
			info, err := getJyNewsInfo(src)
			if err != nil {
				log.Printf("getJyNewsInfo failed:%s %v", src, err)
				return
			}
			if isExpire(info.Date) {
				return
			}
			news = append(news, info)
		}
	})

	return news
}

func getJyNewsInfo(url string) (news News, err error) {
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

	base := getURLBase(url)
	title := d.Find(".nsy-newsTitle").Text()
	news.Title = title
	news.Md5 = util.GetMD5Hash(news.Title)
	ptime := d.Find(".nsy-newsReleasetime").Text()
	news.Date = ptime
	arr := strings.Split(ptime, "：")
	if len(arr) == 2 {
		news.Date = arr[1]
	}
	log.Printf("title:%s time:%s ", news.Title, news.Date)

	var images []string
	var content []Content
	d.Find(".nsy-newsContent p").Each(func(i int, n *goquery.Selection) {
		if img, ok := n.Find("img").Attr("src"); ok {
			image := base + img
			images = append(images, image)
			var cont Content
			cont.Type = typeImg
			cont.Src = image
			content = append(content, cont)
		} else {
			txt := n.Text()
			var cont Content
			cont.Type = typeText
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
	err = tpl.Execute(w, &DgPage{Title: news.Title, Source: "东莞教育局", Ctime: news.Date, Infos: content})
	filename := util.GenSalt() + ".html"
	if flag := aliyun.UploadOssFile(filename, buf.String()); !flag {
		log.Printf("UploadOssFile failed %s:%v", filename, err)
		return news, err
	}
	log.Printf("UploadOssFile success: %s", filename)
	news.URL = aliyun.GenOssNewsURL(filename)
	news.Author = "东莞教育局"

	for i := 0; i < 3 && i < len(images); i++ {
		news.Pics[i] = images[i]
	}

	news.Date = extractDate(news.Date)
	return news, nil
}

//GetWJNews get dongguan weijiju news
func GetWJNews() []News {
	var news []News
	client := &http.Client{}
	req, err := http.NewRequest("GET", wjURL, nil)
	req.Header.Add("User-Agent", "Mozilla/5.0 (Linux; Android 6.0; Nexus 5 Build/MRA58N) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/46.0.2490.76 Mobile Safari/537.36")
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("GetWJNews request failed:%v", err)
		return news
	}
	defer resp.Body.Close()

	d, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Printf("fetch url failed:%v", err)
		return news
	}

	d.Find(".list_div .list-right_title").Each(func(i int, n *goquery.Selection) {
		log.Printf("index:%d, node name:%+v", i, goquery.NodeName(n))
		log.Printf("n:%+v", n)
		src, _ := n.Find("a").Attr("href")
		log.Printf("src:%s", src)
		if src != "" {
			src = wjBase + src
			info, err := getWjNewsInfo(src)
			if err != nil {
				log.Printf("getWjNewsInfo failed:%s %v", src, err)
				return
			}
			if isExpire(info.Date) {
				return
			}
			news = append(news, info)
		}
	})

	return news
}

func gbToUtf8(input []byte) []byte {
	out := make([]byte, len(input)*2)
	out = out[:]
	bytesRead, bytesWritten, err := iconv.Convert(input, out, "gb2312", "utf-8")
	if err != nil {
		log.Printf("iconv.Convert failed:%s %v", string(input), err)
	}
	log.Printf("read:%d write:%d", bytesRead, bytesWritten)
	return out
}

func getURLBase(url string) string {
	pos := strings.LastIndex(url, "/")
	if pos != -1 {
		return url[:pos+1]
	}
	return url
}

func getWjNewsInfo(url string) (news News, err error) {
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

	base := getURLBase(url)
	title := d.Find(".zwgk_comr2 ucaptitle").Text()
	news.Title = title
	news.Md5 = util.GetMD5Hash(news.Title)
	time := d.Find("publishtime").Text()
	news.Date = cleanText(time)
	log.Printf("title:%s time:%s", news.Title, news.Date)

	var images []string
	var content []Content
	d.Find(".zwgk_comr3 p").Each(func(i int, n *goquery.Selection) {
		if img, ok := n.Find("img").Attr("src"); ok {
			image := base + img
			images = append(images, image)
			var cont Content
			cont.Type = typeImg
			cont.Src = image
			content = append(content, cont)
		} else {
			txt := n.Text()
			var cont Content
			cont.Type = typeText
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
	err = tpl.Execute(w, &DgPage{Title: news.Title, Source: "东莞市卫生和计划生育局", Ctime: news.Date, Infos: content})
	filename := util.GenSalt() + ".html"
	if flag := aliyun.UploadOssFile(filename, buf.String()); !flag {
		log.Printf("UploadOssFile failed %s:%v", filename, err)
		return news, err
	}
	log.Printf("UploadOssFile success: %s", filename)
	news.URL = aliyun.GenOssNewsURL(filename)
	news.Author = "东莞市卫生和计划生育局"

	for i := 0; i < 3 && i < len(images); i++ {
		news.Pics[i] = images[i]
	}

	news.Date = extractDate(news.Date)
	return news, nil
}
