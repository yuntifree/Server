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
	f, err := os.Open("wxData.csv")
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
		arr := strings.Split(line, "||")
		if len(arr) != 10 {
			continue
		}
		res, err := db.Exec("INSERT INTO wx_mp_info(wxid, name, abstract, icon, dst, ctime) VALUES (?, ?, ?, ?, ?, NOW()) ON DUPLICATE KEY UPDATE icon = ?", arr[0], arr[1], arr[2], arr[3], arr[4], arr[3])
		if err != nil {
			log.Printf("insert wx_mp_info failed %s %v", line, err)
			continue
		}
		id, err := res.LastInsertId()
		if err != nil {
			log.Printf("get last insert id failed:%v", err)
			continue
		}
		if id == 0 {
			err = db.QueryRow("SELECT id FROM wx_mp_info WHERE wxid = ?", arr[0]).Scan(&id)
			if err != nil || id == 0 {
				log.Printf("select id failed:%v", err)
				continue
			}
		}
		_, err = db.Exec("INSERT INTO wx_mp_article(wid, title, img, dst, ctime, md5) VALUES (?, ?, ?, ?, ?, MD5(?))", id, arr[5], arr[7], arr[8], arr[9], arr[8])
		if err != nil {
			log.Printf("insert wx_mp_artile failed %s %v", line, err)
		}
	}
}
