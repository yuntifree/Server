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
	db, err := util.InitDB(false)
	if err != nil {
		log.Printf("InitDB failed:%v", err)
		os.Exit(1)
	}
	for {
		infos := zte.GetAPInfoList(seq)
		for i := 0; i < len(infos); i++ {
			info := infos[i]
			seq = info.Aid
			log.Printf("%d %f %f %s %s\n", info.Aid, info.Longitude, info.Latitude, info.Address, info.Mac)
			var p1, p2 util.Point
			p1.Longitude = info.Longitude
			p1.Latitude = info.Latitude
			if p1.Longitude != 0 || p1.Latitude != 0 {
				p2 = util.Gps2Bd(p1)
			}
			db.Exec("INSERT IGNORE INTO ap(id, longitude, latitude, address, mac, bd_lon, bd_lat) VALUES (?, ?, ?, ?, ?, ?, ?)", info.Aid, info.Longitude, info.Latitude, info.Address, info.Mac, p2.Longitude, p2.Latitude)
		}
		if len(infos) < 20 {
			break
		}
		seq++
	}
}
