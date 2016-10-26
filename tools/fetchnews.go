package main

import (
	"log"
	"os"

	juhe "../juhe"

	util "../util"
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	db, err := util.InitDB()
	if err != nil {
		log.Printf("InitDB failed:%v", err)
		os.Exit(1)
	}

	for i := 0; i < 9; i++ {
		log.Printf("fetch type:%d", i)
		news := juhe.GetNews(i)
		for j := 0; j < len(news); j++ {
			ns := news[j]
			log.Printf("db title:%s md5:%s", ns.Title, ns.Md5)
			_, err := db.Exec("INSERT IGNORE INTO news(title, img1, img2, img3, dst, source, md5, ctime, dtime) VALUES (?, ?, ?, ?, ?, ?, ?, ?, NOW())", ns.Title, ns.Pics[0], ns.Pics[1], ns.Pics[2], ns.URL, ns.Author, ns.Md5, ns.Date)
			if err != nil {
				log.Printf("insert failed:%v", err)
			}
		}
	}

}
