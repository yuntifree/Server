package main

import (
	"bufio"
	"database/sql"
	"log"
	"os"
	"strconv"
	"strings"

	"../util"
	_ "github.com/go-sql-driver/mysql"
)

func lowMac(ori string) string {
	return strings.Replace(strings.ToLower(ori), "-", "", -1)
}

func upMac(ori string) string {
	up := strings.ToUpper(ori)
	var mac string
	for i := 0; i < 5; i++ {
		mac += up[i*2:i*2+2] + ":"
	}
	mac += up[10:]
	return mac
}

func updateApAdress(db *sql.DB, mac, address string, p1, p2 util.Point) {
	low := lowMac(mac)
	up := upMac(low)
	var id int
	err := db.QueryRow("SELECT id FROM ap WHERE mac IN (?, ?)", low, up).Scan(&id)
	if err != nil {
		log.Printf("scan failed:%v", err)
		return
	}
	log.Printf("update ap id:%d", id)
	_, err = db.Exec("UPDATE ap SET address = ?, longitude = ?, latitude = ?, bd_lon = ?, bd_lat = ? WHERE id = ?", address, p1.Longitude, p1.Latitude, p2.Longitude, p2.Latitude, id)
	if err != nil {
		log.Printf("update ap info failed:%v", err)
	}
}

func main() {
	f, err := os.Open("ap.csv")
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
		mac1 := arr[2]
		mac2 := arr[3]
		lon := arr[4]
		lat := arr[5]
		if (mac1 == "" && mac2 == "") || (lon == "" && lat == "") {
			log.Printf("illegal %s", line)
			continue
		}
		addr := strings.Split(arr[1], "：")
		address := arr[0] + arr[1]
		if len(addr) > 1 {
			address = addr[1] + arr[0]
		}
		log.Printf("address:%s mac:%s|%s", address, mac1, mac2)
		longitude, _ := strconv.ParseFloat(lon, 64)
		latitude, _ := strconv.ParseFloat(lat, 64)
		var p1, p2 util.Point
		p1.Longitude = longitude
		p1.Latitude = latitude
		p2 = util.Gps2Bd(p1)
		if mac1 != "" {
			updateApAdress(db, mac1, address, p1, p2)
		}

		if mac2 != "" {
			updateApAdress(db, mac2, address, p1, p2)
		}
	}
}
