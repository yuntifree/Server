package main

import (
	"log"
	"os"

	util "../util"
	zte "../zte"
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	seq := 0
	db, err := util.InitDB()
	if err != nil {
		log.Printf("InitDB failed:%v", err)
		os.Exit(1)
	}
	for {
		infos := zte.GetAPInfoList(seq)
		for i := 0; i < len(infos); i++ {
			info := infos[i]
			seq = info.Aid
			log.Printf("%d %f %f %s\n", info.Aid, info.Longitude, info.Latitude, info.Address)
			db.Exec("INSERT IGNORE INTO ap(id, longitude, latitude, address) VALUES (?, ?, ?, ?)", info.Aid, info.Longitude, info.Latitude, info.Address)
		}
		if len(infos) < 20 {
			break
		}
		seq++
	}
}
