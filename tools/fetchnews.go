package main

import (
	"log"
	"os"

	"Server/juhe"

	"Server/util"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	db, err := util.InitDB(false)
	if err != nil {
		log.Printf("InitDB failed:%v", err)
		os.Exit(1)
	}

	for i := 0; i <= 9; i++ {
		log.Printf("fetch type:%d", 9-i)
		news := juhe.GetNews(9 - i)
		for j := 0; j < len(news); j++ {
			ns := news[j]
			_, err := db.Exec("INSERT IGNORE INTO news(title, img1, img2, img3, dst, source, md5, ctime, stype, dtime) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NOW())", ns.Title, ns.Pics[0], ns.Pics[1], ns.Pics[2], ns.URL, ns.Author, ns.Md5, ns.Date, 9-i)
			if err != nil {
				log.Printf("insert failed:%v", err)
			}
		}
	}

}
