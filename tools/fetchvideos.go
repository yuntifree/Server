package main

import (
	"log"
	"os"
	"strconv"

	"../juhe"
	util "../util"
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	db, err := util.InitDB()
	if err != nil {
		log.Printf("InitDB failed:%v", err)
		os.Exit(1)
	}

	for i := 0; i < 34; i++ {
		files := juhe.GetYoukuFiles(i*30, 30)
		for j := 0; j < len(files); j++ {
			f := files[j]
			duration, _ := strconv.Atoi(f.Duration)
			_, err := db.Exec("INSERT IGNORE INTO youku_video(id, origin_id, title, img, play_url, duration, ctime) VALUES (?, ?, ?, ?, ?, ?, NOW())", f.ID, f.OriginID, f.Title, f.ImgURL, f.PlayURL, duration)
			if err != nil {
				log.Printf("insert failed:%v", err)
			}
		}
	}
}
