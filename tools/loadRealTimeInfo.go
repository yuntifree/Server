package main

import (
	"Server/util"
	"Server/zte"
	"log"
	"os"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
)

type apInfo struct {
	aid, online int
	bandwidth   float64
}

func main() {
	db, err := util.InitDB(false)
	if err != nil {
		log.Printf("InitDB failed:%v", err)
		os.Exit(1)
	}

	var total int
	err = db.QueryRow("SELECT MAX(id) FROM ap").Scan(&total)
	if err != nil {
		log.Fatalf("get max aid faled:%v", err)
	}

	log.Printf("start fetch ap realinfo\n")
	var infos []apInfo
	for i := 1; i <= total; i++ {
		info, err := zte.GetRealTimeInfo(i)
		if err != nil {
			log.Printf("GetRealTimeInfo failed, aid:%d err:%v\n", i, info)
			continue
		}
		if info.Online > 0 {
			var ainfo apInfo
			ainfo.aid = i
			ainfo.online = info.Online
			if info.Bandwidth != "" {
				ainfo.bandwidth, err = strconv.ParseFloat(info.Bandwidth, 64)
				if err != nil {
					log.Printf("ParseFloat failed:%s %v\n", info.Bandwidth, err)
					continue
				}
			}
			log.Printf("id:%d online:%d bandwidth:%s\n", i, info.Online, info.Bandwidth)
			infos = append(infos, ainfo)
		}
	}
	log.Printf("finish fetch ap realinfo size:%d\n", len(infos))

	for i := 0; i < len(infos); i++ {
		if infos[i].online > 0 {
			_, err = db.Exec("UPDATE ap SET count = ?, bandwidth = ?, mtime = NOW() WHERE id = ?", infos[i].online, infos[i].bandwidth, infos[i].aid)
			if err != nil {
				log.Printf("db exec failed:%v", err)
			}
		}

	}
	log.Printf("finish store realinfo to db\n")
}
