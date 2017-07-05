package main

import (
	"database/sql"
	"log"
	"os"

	"Server/juhe"
	"Server/util"

	_ "github.com/go-sql-driver/mysql"
)

func storeNews(db *sql.DB, news []juhe.News) {
	log.Printf("num:%d\n", len(news))
	for j := len(news) - 1; j >= 0; j-- {
		ns := news[j]
		_, err := db.Exec("INSERT IGNORE INTO news(title, img1, img2, img3, dst, source, md5, ctime, stype, origin, dtime) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW())", ns.Title, ns.Pics[0], ns.Pics[1], ns.Pics[2], ns.URL, ns.Author, ns.Md5, ns.Date, 10, ns.Origin)
		if err != nil {
			log.Printf("insert failed:%v", err)
		}
	}
}

func main() {
	db, err := util.InitDB(false)
	if err != nil {
		log.Printf("InitDB failed:%v", err)
		os.Exit(1)
	}
	fgnews := juhe.GetFGNews()
	storeNews(db, fgnews)
	gdnews := juhe.GetGDNews()
	storeNews(db, gdnews)
	jcnews := juhe.GetJCNews()
	storeNews(db, jcnews)
	jynews := juhe.GetJYNews()
	storeNews(db, jynews)
	wjnews := juhe.GetWJNews()
	storeNews(db, wjnews)
}
