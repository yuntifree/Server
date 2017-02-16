package main

import (
	"Server/juhe"
	"Server/util"
	"database/sql"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func record(db *sql.DB, info *juhe.LiveInfo) {
	ts := time.Now().UnixNano() / 1000000
	nickname := info.Nickname
	if nickname == "" {
		nickname = "主播"
	}
	location := info.Location
	if location == "" {
		location = "难道在火星？"
	}
	_, err := db.Exec("INSERT INTO live(uid, avatar, nickname, live_id, img, p_time, location, watches, live, seq) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE uid = ?, avatar = ?, nickname = ?, img = ?, p_time = ?, location = ?, watches = ?, live = ?, seq = ?", info.Uid, info.Avatar, nickname, info.LiveId, info.Img, info.PTime, location, info.Watches, info.Live, ts, info.Uid, info.Avatar, nickname, info.Img, info.PTime, location, info.Watches, info.Live, ts)
	if err != nil {
		log.Printf("record info:%v failed:%v", info, err)
	}
}

func main() {
	db, err := util.InitDB(false)
	if err != nil {
		log.Printf("InitDB failed:%v", err)
		os.Exit(1)
	}

	for l := 0; l < 6; l++ {
		for i := 0; i < 50; i++ {
			infos, offset := juhe.GetLiveInfo(int64(i * 10))
			if len(infos) == 0 || offset == 0 {
				break
			}
			for j := 0; j < len(infos); j++ {
				record(db, infos[j])
			}
		}
		time.Sleep(10 * time.Second)
	}
}
