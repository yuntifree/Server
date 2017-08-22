package main

import (
	"Server/util"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
	_ "github.com/go-sql-driver/mysql"
)

const (
	queryURL  = "http://www.gsdata.cn/query/wx?q="
	baseURL   = "http://www.gsdata.cn"
	infoURL   = "http://www.gsdata.cn/rank/toparc"
	detailURL = "http://www.gsdata.cn/rank/wxdetail"
	wxName    = "NF_Daily"
)

type arcResponse struct {
	Error int64     `json:"error"`
	Data  []arcInfo `json:"data"`
}

type arcInfo struct {
	Title    string `json:"title"`
	PostTime string `json:"posttime"`
	ReadNum  int64  `json:"readnum_newest"`
	LikeNum  int64  `json:"likenum_newest"`
	URL      string `json:"url"`
	Md5      string `json:"md5"`
}

func main() {
	db, err := util.InitWxDB()
	if err != nil {
		log.Fatal(err)
	}

	rows, err := db.Query("SELECT id, wx_id FROM gzh WHERE wx_id = ?", wxName)
	if err != nil {
		log.Fatal(err)
	}

	for rows.Next() {
		var id int64
		var name string
		err := rows.Scan(&id, &name)
		if err != nil {
			continue
		}
		scrapyWxInfo(db, id, name)
	}
}

func scrapyWxInfo(db *sql.DB, wid int64, wxname string) {
	name, err := getWxName(wxname)
	if err != nil {
		log.Printf("getWxName failed: %s %v", wxname, err)
		return
	}
	url := fmt.Sprintf("%s?wxname=%s&wx=%s&sort=-1", infoURL,
		name, wxname)
	referer := fmt.Sprintf("%s?wxname=%s", detailURL, name)
	log.Printf("url:%s referer:%s", url, referer)
	rsp, err := getDetailInfo(url, referer)
	if err != nil {
		log.Printf("getDetailInfo failed:%v", err)
		return
	}

	for _, v := range rsp.Data {
		v.Md5 = util.GetMD5Hash(v.Title)
		_, err := db.Exec("INSERT IGNORE INTO gzh_article(wid, title, md5, readnum, likenum, url, post_time, ctime) VALUES(?, ?, ?, ?, ?, ?, ?, NOW())",
			wid, v.Title, v.Md5, v.ReadNum, v.LikeNum, v.URL, v.PostTime)
		if err != nil {
			log.Printf("record article info failed:%s %v", v.Title, err)
		}
	}
}

func genHeaders(referer string) map[string]string {
	headers := make(map[string]string)
	headers["Accept"] = "*/*"
	//headers["Accept-Encoding"] = "gzip, deflate"
	headers["Accept-Language"] = "zh-CN,zh;q=0.8,en;q=0.6"
	headers["Cache-Control"] = "no-cache"
	headers["Connection"] = "keep-alive"
	headers["Host"] = "www.gsdata.cn"
	headers["Origin"] = "http://www.gsdata.cn"
	headers["Pragma"] = "no-cache"
	headers["Referer"] = referer
	headers["User-Agent"] = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/59.0.3071.109 Safari/537.36"
	headers["X-Requested-With"] = "XMLHttpRequest"
	return headers
}

func getDetailInfo(url, referer string) (*arcResponse, error) {
	var arc arcResponse
	headers := genHeaders(referer)
	resp, err := util.HTTPRequestWithHeaders(url, "", headers)
	if err != nil {
		return nil, err
	}
	log.Printf("resp:%s", resp)
	err = json.Unmarshal([]byte(resp), &arc)
	if err != nil {
		return nil, err
	}
	log.Printf("arc:%+v", arc)
	return &arc, nil
}

func getWxName(wxname string) (string, error) {
	url := queryURL + wxname
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Add("User-Agent", "Mozilla/5.0 (Linux; Android 6.0; Nexus 5 Build/MRA58N) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/46.0.2490.76 Mobile Safari/537.36")
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("getWxDetail http request failed:%v", err)
		return "", err
	}
	defer resp.Body.Close()

	d, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Printf("getWxDetail parse doc failed:%v", err)
		return "", err
	}

	div := d.Find(".img-word")
	a, flag := div.Find(".img").First().Attr("href")
	if flag {
		pos := strings.Index(a, "=")
		if pos != -1 {
			name := a[pos+1:]
			return name, nil
		}
	}
	return "", errors.New("not found")
}
