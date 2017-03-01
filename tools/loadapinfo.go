package main

import (
	"Server/util"
	"bufio"
	"log"
	"os"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	f, err := os.Open("20170301.csv")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	db, err := util.InitDB(false)
	if err != nil {
		panic(err)
	}

	rd := bufio.NewReader(f)
	for {
		data, err := rd.ReadSlice('\n')
		if err != nil {
			break
		}
		line := string(data)
		arr := strings.Split(line, ",")
		unit := arr[0]
		address := arr[1]
		mac := arr[2]
		lon := arr[3]
		lat := arr[4]
		_, err = db.Exec("INSERT IGNORE INTO ap_info(unit, address, mac, longitude, latitude) VALUES (?, ?, ?, ?, ?)", unit, address, mac, lon, lat)
		if err != nil {
			log.Printf("insert failed:%v", err)
		}
	}
}
