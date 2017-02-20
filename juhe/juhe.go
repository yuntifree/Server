package juhe

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"html/template"

	"github.com/axgle/mahonia"
	simplejson "github.com/bitly/go-simplejson"

	"Server/aliyun"
	"Server/util"

	"github.com/PuerkitoBio/goquery"
)

const (
	baseurl    = "http://v.juhe.cn/toutiao/index"
	appkey     = "1fe1ae10361f6ae1ffa0bc4a5d7a212f"
	dgurl      = "http://news.sun0769.com/dg/"
	weatherurl = "https://api.thinkpage.cn/v3/weather/now.json?key=uixmdugjglekq1ng&location=dongguan&language=zh-Hans&unit=c"
	jokeurl    = "http://japi.juhe.cn/joke/content/list.from"
	jokeimgurl = "http://japi.juhe.cn/joke/img/list.from"
	jokekey    = "deecda0845a613ae153c3ea7efa30a06"
)

//News information for news
type News struct {
	Title, Date, Author, URL, Md5, Origin string
	Pics                                  [3]string
	Stype                                 int
}

//Joke information for jokes
type Joke struct {
	Content, Date, Md5, URL string
	Type                    int64
}

//Page page info
type Page struct {
	Title   string
	Content template.HTML
}

//DgPage dongguan page info
type DgPage struct {
	Title  string
	Source string
	Ctime  string
	Infos  []Content
}

//Content content information contains image and text
type Content struct {
	Type int
	Src  string
}

//Weather weather info
type Weather struct {
	Temperature int
	Info        string
	Type        int
	Date        string
}

const (
	typeImg = iota
	typeText
)

func getTypeStr(stype int) string {
	switch stype {
	case 1:
		return "shehui"
	case 2:
		return "guonei"
	case 3:
		return "guoji"
	case 4:
		return "yule"
	case 5:
		return "tiyu"
	case 6:
		return "junshi"
	case 7:
		return "keji"
	case 8:
		return "caijing"
	case 9:
		return "shishang"
	case 10:
		return "dongguan"
	default:
		return "top"
	}

}

func extractDate(date string) string {
	var digitReg = regexp.MustCompile(`(\d+)\D+(\d+)\D+(\d+)\D+(\d+)\D+(\d+)`)
	arr := digitReg.FindStringSubmatch(date)
	if len(arr) == 6 {
		return arr[1] + "-" + arr[2] + "-" + arr[3] + " " + arr[4] + ":" + arr[5] + ":00"
	}
	return date
}

func initTemplate() *template.Template {
	tpl, err := template.ParseFiles("/data/server/templates/news.html")
	if err != nil {
		panic("parse template failed")
	}
	return tpl
}

//GetDgNews get dongguan news
func GetDgNews() []News {
	var news []News
	client := &http.Client{}
	req, err := http.NewRequest("GET", dgurl, nil)
	req.Header.Add("User-Agent", "Mozilla/5.0 (Linux; Android 6.0; Nexus 5 Build/MRA58N) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/46.0.2490.76 Mobile Safari/537.36")
	resp, err := client.Do(req)
	defer resp.Body.Close()

	d, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Printf("fetch url failed:%v", err)
		return news
	}

	d.Find(".scrollLeft .mListA").Each(func(i int, n *goquery.Selection) {
		src, _ := n.Find(".postBody a").Attr("href")
		ns, err := GetContent(src)
		if err == nil {
			news = append(news, ns)
			log.Printf("src:%s\n", src)
		}
	})
	return news
}

func cleanText(text string) string {
	enc := mahonia.NewDecoder("GB18030")
	txt := enc.ConvertString(text)
	txt = strings.TrimSpace(txt)
	txt = strings.Replace(txt, "聽", " ", -1)
	return txt
}

func getBaseURL(url string) string {
	pos := strings.LastIndex(url, "/")
	prefix := url[:pos]
	return prefix
}

//GetContent fetch content from url
func GetContent(url string) (news News, err error) {
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

	base := getBaseURL(url)

	title := d.Find(".article_hd").Text()
	news.Title = cleanText(title)
	news.Md5 = util.GetMD5Hash(news.Title)
	time := d.Find(".titbar span.dtp").Text()
	news.Date = cleanText(time)
	log.Printf("title:%s time:%s", news.Title, news.Date)

	var images []string
	var content []Content
	d.Find(".TRS_Editor p").Each(func(i int, n *goquery.Selection) {
		if img, ok := n.Find("img").Attr("src"); ok {
			img = base + img[1:]
			images = append(images, img)
			var cont Content
			cont.Type = typeImg
			cont.Src = img
			content = append(content, cont)
		} else {
			txt := n.Text()
			txt = cleanText(txt)
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
	err = tpl.Execute(w, &DgPage{Title: news.Title, Source: "东莞阳光网", Ctime: news.Date, Infos: content})
	filename := util.GenSalt() + ".html"
	if flag := aliyun.UploadOssFile(filename, buf.String()); !flag {
		log.Printf("UploadOssFile failed %s:%v", filename, err)
		return news, err
	}
	log.Printf("UploadOssFile success: %s", filename)
	news.URL = aliyun.GenOssNewsURL(filename)
	news.Author = "东莞阳光网"

	for i := 0; i < 3 && i < len(images); i++ {
		news.Pics[i] = images[i]
	}

	news.Date = extractDate(news.Date)
	return news, nil
}

//GetNews fetch news
func GetNews(stype int) []News {
	tpl := initTemplate()
	news := make([]News, 50)
	typeStr := getTypeStr(stype)
	url := baseurl + "?type=" + typeStr + "&key=" + appkey

	rspbody, err := util.HTTPRequest(url, "")
	if err != nil {
		log.Printf("HTTPRequest failed:%v", err)
		return nil
	}

	js, _ := simplejson.NewJson([]byte(`{}`))
	err = js.UnmarshalJSON([]byte(rspbody))
	if err != nil {
		log.Printf("parse rspbody failed:%v", err)
		return nil
	}

	errcode, err := js.Get("error_code").Int()
	if err != nil {
		log.Printf("get error code failed:%v", err)
		return nil
	}

	if errcode != 0 {
		log.Printf("get error code failed:%v", err)
		return nil
	}

	arr, err := js.Get("result").Get("data").Array()
	if err != nil {
		log.Printf("get data failed:%v", err)
		return nil
	}

	i := 0
	for ; i < len(arr); i++ {
		info := js.Get("result").Get("data").GetIndex(i)
		var ns News
		ns.Title, _ = info.Get("title").String()
		ns.Stype = stype
		ns.Md5 = util.GetMD5Hash(ns.Title)
		ns.Date, _ = info.Get("date").String()
		ns.URL, _ = info.Get("url").String()
		d, err := goquery.NewDocument(ns.URL)
		if err != nil {
			log.Printf("fetch url failed:%v", err)
			continue
		}

		pics, err := GetImages(d, ns.URL)
		if err != nil {
			log.Printf("fetch images from url failed:%v", err)
			ns.Pics[0], _ = info.Get("thumbnail_pic_s").String()
		} else {
			for i := 0; i < len(pics) && i < 3; i++ {
				ns.Pics[i] = pics[i]
			}
		}
		title := d.Find("title").Text()
		content, err := d.Find("article").Html()
		if err != nil {
			log.Printf("get article failed %s:%v", ns.URL, err)
			continue
		}
		var buf bytes.Buffer
		w := io.Writer(&buf)
		err = tpl.Execute(w, &Page{Title: title, Content: template.HTML(content)})
		filename := util.GenSalt() + ".html"
		if flag := aliyun.UploadOssFile(filename, buf.String()); !flag {
			log.Printf("UploadOssFile failed %s:%v", filename, err)
			continue
		}
		ns.URL = aliyun.GenOssNewsURL(filename)
		ns.Author, _ = info.Get("author_name").String()
		news[i] = ns
		log.Printf("title:%s", ns.Title)
	}

	return news[:i]
}

//GetImages extract images from url
func GetImages(d *goquery.Document, url string) ([]string, error) {
	var images []string
	sel := d.Find("a")
	sel.Each(func(i int, n *goquery.Selection) {
		if val, ok := n.Attr("class"); ok {
			if val == "img-wrap" {
				if href, ok := n.Attr("href"); ok {
					images = append(images, href)
				}
			}
		}
	})

	return images, nil
}

//GetRealWeather get realtime weather of dongguan
func GetRealWeather() (Weather, error) {
	var w Weather
	res, err := util.HTTPRequest(weatherurl, "")
	if err != nil {
		log.Printf("request failed %s:%v", weatherurl, err)
		return w, err
	}

	js, err := simplejson.NewJson([]byte(res))
	if err != nil {
		log.Printf("parse resp failed:%v", err)
		return w, err
	}

	result := js.Get("results").GetIndex(0)
	tm, err := result.Get("last_update").String()
	if err != nil {
		log.Printf("get time failed:%v", err)
		return w, err
	}
	tmp, err := result.Get("now").Get("temperature").String()
	if err != nil {
		log.Printf("get temperature failed:%v", err)
		return w, err
	}

	info, err := result.Get("now").Get("text").String()
	if err != nil {
		log.Printf("get info failed:%v", err)
		return w, err
	}

	w.Temperature, _ = strconv.Atoi(tmp)
	w.Info = info
	w.Date = tm
	switch info {
	default:
		w.Type = 0
	case "阴", "多云", "大部多云":
		w.Type = 1
	case "阵雨", "雷阵雨", "小雨", "中雨", "大雨", "暴雨", "特大暴雨":
		w.Type = 2
	case "雪", "小雪", "中雪", "大雪", "暴雪", "阵雪":
		w.Type = 3
	}

	return w, nil
}

//GetJoke return jokes
func GetJoke(ts, page, pagesize, ctype int64) []*Joke {
	var infos []*Joke
	host := jokeurl
	if ctype != 0 {
		host = jokeimgurl
	}
	url := fmt.Sprintf("%s?key=%s&page=%d&pagesize=%d&sort=desc&time=%d", host,
		jokekey, page, pagesize, ts)
	res, err := util.HTTPRequest(url, "")
	if err != nil {
		log.Printf("request failed %s:%v", url, err)
		return infos
	}

	js, err := simplejson.NewJson([]byte(res))
	if err != nil {
		log.Printf("parse resp failed:%v", err)
		return infos
	}

	errcode, err := js.Get("error_code").Int()
	if err != nil || errcode != 0 {
		log.Printf("check error_code failed")
		return infos
	}
	arr, err := js.Get("result").Get("data").Array()
	if err != nil {
		log.Printf("get data failed:%v", err)
		return nil
	}

	i := 0
	for ; i < len(arr); i++ {
		var joke Joke
		info := js.Get("result").Get("data").GetIndex(i)
		joke.Content, _ = info.Get("content").String()
		joke.Date, _ = info.Get("updatetime").String()
		joke.Md5, _ = info.Get("hashId").String()
		joke.Type = ctype
		if ctype != 0 {
			joke.URL, _ = info.Get("url").String()
		}
		infos = append(infos, &joke)
	}
	return infos
}
