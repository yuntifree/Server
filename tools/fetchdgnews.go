package main

import (
	"log"
	"os"

	juhe "../juhe"
	util "../util"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	db, err := util.InitDB(false)
	if err != nil {
		log.Printf("InitDB failed:%v", err)
		os.Exit(1)
	}
	news := juhe.GetDgNews()
	log.Printf("num:%d\n", len(news))
	for j := 0; j < len(news); j++ {
		ns := news[j]
		_, err := db.Exec("INSERT IGNORE INTO news(title, img1, img2, img3, dst, source, md5, ctime, stype, dtime) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NOW())", ns.Title, ns.Pics[0], ns.Pics[1], ns.Pics[2], ns.URL, ns.Author, ns.Md5, ns.Date, 10)
		if err != nil {
			log.Printf("insert failed:%v", err)
		}
	}
}
