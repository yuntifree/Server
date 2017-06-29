package main

import (
	"Server/util"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

type reserveInfo struct {
	id           int64
	sid          int64
	name         string
	phone        string
	reserve_date string
	ctime        string
	btype        int64
}

func main() {
	db, err := util.InitDB(false)
	if err != nil {
		log.Printf("InitDB failed:%v", err)
		os.Exit(1)
	}

	rows, err := db.Query("SELECT id, sid, name, phone, btype, reserve_date, ctime FROM reserve_info WHERE id > 92")
	if err != nil {
		log.Printf("query failed:%v", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var info reserveInfo
		err := rows.Scan(&info.id, &info.sid, &info.name, &info.phone,
			&info.btype, &info.reserve_date, &info.ctime)
		if err != nil {
			log.Printf("scan failed:%v", err)
			continue
		}
		fmt.Printf("%d,%d,%s,%s,%d,%s,%s\n", info.id, info.sid, info.name,
			info.phone, info.btype, info.reserve_date, info.ctime)
	}
}
