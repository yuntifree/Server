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
	db, err := util.InitDB(false)
	if err != nil {
		log.Printf("InitDB failed:%v", err)
		os.Exit(1)
	}

	for i := 0; i < 34; i++ {
		files := juhe.GetYoukuFiles(i*30, 30)
		for j := 0; j < len(files); j++ {
			f := files[j]
			duration, _ := strconv.Atoi(f.Duration)
			if f.OriginID == "" {
				continue
			}
			dst := juhe.GenYoukuURL(f.OriginID)
			if f.Source == "乐视" {
				dst = juhe.GenLetvURL(f.OriginID)
			}
			_, err := db.Exec("INSERT IGNORE INTO youku_video(id, origin_id, title, img, play_url, duration, source, dst, md5, ctime) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NOW())", f.ID, f.OriginID, f.Title, f.ImgURL, f.PlayURL, duration, f.Source, dst, f.MD5)
			if err != nil {
				log.Printf("insert failed:%v", err)
			}
		}
	}
}
