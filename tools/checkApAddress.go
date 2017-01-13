package main

import (
	"log"
	"math"
	"os"
	"strconv"

	util "../util"
	_ "github.com/go-sql-driver/mysql"
)

func checkPosition(p, q util.Point) bool {
	if math.Abs(p.Longitude-q.Longitude)+math.Abs(p.Latitude-q.Latitude) > 0.002 {
		return false
	}
	return true
}

func main() {
	if len(os.Args) < 3 {
		log.Printf("not enough param")
		os.Exit(1)
	}
	start, _ := strconv.Atoi(os.Args[1])
	end, _ := strconv.Atoi(os.Args[2])
	db, err := util.InitDB(false)
	if err != nil {
		log.Printf("InitDB failed:%v", err)
		os.Exit(1)
	}
	rows, err := db.Query("SELECT id, mac, longitude, latitude, address FROM ap WHERE address != '' AND id >= ? AND id < ? ORDER BY id",
		start, end)
	if err != nil {
		log.Printf("query failed:%v", err)
		os.Exit(1)
	}

	defer rows.Close()
	for rows.Next() {
		var id int64
		var p util.Point
		var address, mac string
		err := rows.Scan(&id, &mac, &p.Longitude, &p.Latitude, &address)
		if err != nil {
			log.Printf("scan failed:%v", err)
			continue
		}
		q := util.GeoEncode(address)
		if q.Latitude == 0 || q.Longitude == 0 {
			continue
		}
		log.Printf("id:%d", id)
		if !checkPosition(p, q) {
			log.Printf("checkPosition failed, id:%d address:%s mac:%s position origion:%f,%f query:%f, %f",
				id, address, mac, p.Longitude, p.Latitude, q.Longitude, q.Latitude)
			log.Printf("wifi--ap,%s,%s,%f,%f,%f,%f", address, mac, p.Longitude, p.Latitude, q.Longitude, q.Latitude)
		}
	}
}
