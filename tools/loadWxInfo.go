package main

import (
	"Server/util"
	"bufio"
	"log"
	"os"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

func getType(typestr string) int64 {
	switch typestr {
	case "东莞":
		return 0
	case "热门":
		return 1
	}
	return 0
}

func getSubtype(substr string) int64 {
	switch substr {
	case "东莞":
		return 0
	case "生活":
		return 1
	case "媒体":
		return 2
	case "娱乐":
		return 3
	case "美食":
		return 4
	case "教育":
		return 5
	case "科技":
		return 6
	case "金融":
		return 7
	case "电影":
		return 8
	case "音乐":
		return 9
	case "汽车":
		return 10
	case "读书":
		return 11
	}
	return 0
}

func main() {
	f, err := os.Open("mpwx.csv")
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
		if len(arr) != 3 {
			continue
		}
		name := strings.TrimSpace(arr[0])
		typestr := strings.TrimSpace(arr[2])
		substr := strings.TrimSpace(arr[1])
		mtype := getType(typestr)
		subtype := getSubtype(substr)
		log.Printf("name:%s typestr:%s substr:%s mtype:%d subtype:%d", name,
			typestr, substr, mtype, subtype)
		_, err = db.Exec("UPDATE wx_mp_info SET type = ?, subtype = ? WHERE name = ?", mtype, subtype, name)
		if err != nil {
			log.Printf("update failed:%v", err)
		}
		var id int64
		err = db.QueryRow("SELECT id FROM wx_mp_info WHERE name = ?", name).Scan(&id)
		if err != nil {
			log.Printf("scan id failed:%v", err)
			continue
		}
		_, err = db.Exec("UPDATE wx_mp_article SET type = ? WHERE wid = ?", subtype, id)
		if err != nil {
			log.Printf("update article failed:%v", err)
		}
	}
}
