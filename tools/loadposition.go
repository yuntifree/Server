package main

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"

	"../util"
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	f, err := os.Open("wifi.csv")
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
		for i := 0; i < len(arr); i++ {
			log.Printf("i:%d val:%s", i, arr[i])
		}
		mac := arr[0]
		lon := arr[3]
		lat := arr[4]
		lat = strings.TrimSpace(lat)
		longitude, err := strconv.ParseFloat(lon, 64)
		if err != nil {
			log.Printf("strconv failed:%v", err)
			return
		}
		latitude, err := strconv.ParseFloat(lat, 64)
		if err != nil {
			log.Printf("strconv failed:%v", err)
			return
		}
		log.Printf("longitude:%f latitude:%f mac:%s", longitude, latitude, mac)
		_, err = db.Exec("UPDATE ap SET longitude = ?, latitude = ? WHERE mac = ?", longitude, latitude, mac)
		if err != nil {
			log.Printf("UPDATE failed, longitude:%f latitude:%f mac:%s", longitude, latitude, mac)
		}
		return
	}
}
