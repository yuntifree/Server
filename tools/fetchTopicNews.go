package main

import (
	"log"
	"os"
	"sort"
	"strings"

	"Server/juhe"
	"Server/util"

	_ "github.com/go-sql-driver/mysql"
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

type sortNews []juhe.News

func (s sortNews) Len() int      { return len(s) }
func (s sortNews) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s sortNews) Less(i, j int) bool {
	if strings.Compare(s[i].Date, s[j].Date) >= 0 {
		return false
	}
	return true
}

func main() {
	db, err := util.InitDB(false)
	if err != nil {
		log.Printf("InitDB failed:%v", err)
		os.Exit(1)
	}

	var total []juhe.News

	for i := 0; i < len(targets); i++ {
		news := juhe.GetTopicNews(targets[i])
		total = append(total, news...)
	}
	sort.Sort(sortNews(total))
	for i := 0; i < len(total); i++ {
		log.Printf("news:%+v", total[i])
		ns := total[i]
		_, err := db.Exec("INSERT INTO news(title, img1, img2, img3, dst, source, md5, ctime, origin, stype, dtime) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW()) ON DUPLICATE KEY UPDATE ctime = ?",
			ns.Title, ns.Pics[0], ns.Pics[1], ns.Pics[2], ns.URL, ns.Author, ns.Md5,
			ns.Date, ns.Origin, 19, ns.Date)
		if err != nil {
			log.Printf("insert failed:%v", err)
		}
	}
}
