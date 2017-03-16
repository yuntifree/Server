package main

import (
	"Server/util"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

type apUnit struct {
	Name      string
	Address   string
	Longitude float64
	Latitude  float64
	Cnt       int64
}

func main() {
	db, err := util.InitDB(false)
	if err != nil {
		log.Printf("InitDB failed:%v", err)
		os.Exit(1)
	}

	rows, err := db.Query("SELECT unit, address, longitude, latitude, COUNT(id) FROM ap_info GROUP BY longitude, latitude")
	if err != nil {
		log.Printf("query failed:%v", err)
		os.Exit(1)
	}

	defer rows.Close()
	for rows.Next() {
		var unit apUnit
		err := rows.Scan(&unit.Name, &unit.Address, &unit.Longitude, &unit.Latitude,
			&unit.Cnt)
		if err != nil {
			log.Printf("scan failed:%v", err)
			continue
		}
		_, err = db.Exec("INSERT INTO unit(name, address, longitude, latitude, cnt, ctime) VALUES (?, ?, ?, ?, ?, NOW())",
			unit.Name, unit.Address, unit.Longitude, unit.Latitude,
			unit.Cnt)
		if err != nil {
			log.Printf("insert failed:%v", unit)
		}
	}
}
