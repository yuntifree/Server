package main

import (
	"Server/util"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

type apUnit struct {
	Id        int64
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

	rows, err := db.Query("SELECT id, name, address, longitude, latitude FROM unit")
	if err != nil {
		log.Printf("query failed:%v", err)
		os.Exit(1)
	}

	defer rows.Close()
	for rows.Next() {
		var unit apUnit
		err := rows.Scan(&unit.Id, &unit.Name, &unit.Address, &unit.Longitude,
			&unit.Latitude)
		if err != nil {
			log.Printf("scan failed:%v", err)
			continue
		}
		_, err = db.Exec("update ap_info set unid = ? WHERE longitude = ? AND latitude = ?",
			unit.Id, unit.Longitude, unit.Latitude)
		if err != nil {
			log.Printf("insert failed:%v", unit)
		}
	}
}
