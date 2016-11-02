package main

import (
	"database/sql"
	"log"
	"net"
	"strconv"

	"golang.org/x/net/context"

	"google.golang.org/grpc"

	common "../../proto/common"
	fetch "../../proto/fetch"
	util "../../util"
	_ "github.com/go-sql-driver/mysql"
)

const (
	port = ":50055"
)

type server struct{}

func getReviewNews(db *sql.DB, seq, num int64) []*fetch.NewsInfo {
	var infos []*fetch.NewsInfo
	query := "select id, title from news where 1 = 1 "
	if seq != 0 {
		query += " and id < " + strconv.Itoa(int(seq))
	}
	query += " order by id desc limit " + strconv.Itoa(int(num))
	log.Printf("query string:%s", query)
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("query failed:%v", err)
		return infos
	}

	for rows.Next() {
		var info fetch.NewsInfo
		err = rows.Scan(&info.Id, &info.Title)
		if err != nil {
			log.Printf("scan rows failed: %v", err)
			return infos
		}
		infos = append(infos, &info)
		log.Printf("id:%s title:%s ", info.Id, info.Title)
	}
	return infos
}

func (s *server) FetchReviewNews(ctx context.Context, in *fetch.CommRequest) (*fetch.NewsReply, error) {
	db, err := util.InitDB(true)
	if err != nil {
		log.Printf("connect mysql failed:%v", err)
		return &fetch.NewsReply{Head: &common.Head{Retcode: 1}}, err
	}
	log.Printf("request uid:%d, sid:%s seq:%d, num:%d", in.Head.Uid, in.Head.Sid, in.Seq, in.Num)
	news := getReviewNews(db, in.Seq, int64(in.Num))
	return &fetch.NewsReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid}, Infos: news}, nil
}

func getTags(db *sql.DB, seq, num int64) []*fetch.TagInfo {
	var infos []*fetch.TagInfo
	query := "select id, content from tags where 1 = 1 "
	if seq != 0 {
		query += " and id < " + strconv.Itoa(int(seq))
	}
	query += " order by id desc limit " + strconv.Itoa(int(num))
	log.Printf("query string:%s", query)
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("query failed:%v", err)
		return infos
	}

	for rows.Next() {
		var info fetch.TagInfo
		err = rows.Scan(&info.Id, &info.Content)
		if err != nil {
			log.Printf("scan rows failed: %v", err)
			return infos
		}
		infos = append(infos, &info)
		log.Printf("id:%s content:%s ", info.Id, info.Content)
	}
	return infos
}

func (s *server) FetchTags(ctx context.Context, in *fetch.CommRequest) (*fetch.TagsReply, error) {
	db, err := util.InitDB(true)
	if err != nil {
		log.Printf("connect mysql failed:%v", err)
		return &fetch.TagsReply{Head: &common.Head{Retcode: 1}}, err
	}
	log.Printf("request uid:%d, sid:%s seq:%d, num:%d", in.Head.Uid, in.Head.Sid, in.Seq, in.Num)
	tags := getTags(db, in.Seq, int64(in.Num))
	return &fetch.TagsReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid}, Infos: tags}, nil
}

func getAps(db *sql.DB, longitude, latitude float64) []*fetch.ApInfo {
	var infos []*fetch.ApInfo
	rows, err := db.Query("SELECT id, longitude, latitude FROM ap WHERE longitude > ? - 0.1 AND longitude < ? + 0.1 AND latitude > ? - 0.1 AND latitude < ? + 0.1 ORDER BY (pow(abs(longitude - ?), 2) + pow(abs(latitude - ?), 2)) LIMIT 20", longitude, longitude, latitude, latitude, longitude, latitude)
	if err != nil {
		log.Printf("query failed:%v", err)
		return infos
	}

	for rows.Next() {
		var info fetch.ApInfo
		err = rows.Scan(&info.Id, &info.Longitude, &info.Latitude)
		if err != nil {
			log.Printf("scan rows failed: %v", err)
			return infos
		}
		infos = append(infos, &info)
		log.Printf("id:%s longitude:%f latitude:%f ", info.Id, info.Longitude, info.Latitude)
	}
	return infos
}

func (s *server) FetchAps(ctx context.Context, in *fetch.ApRequest) (*fetch.ApReply, error) {
	db, err := util.InitDB(true)
	if err != nil {
		log.Printf("connect mysql failed:%v", err)
		return &fetch.ApReply{Head: &common.Head{Retcode: 1}}, err
	}
	log.Printf("request uid:%d, sid:%s longitude:%f latitude:%f", in.Head.Uid, in.Head.Sid, in.Longitude, in.Latitude)
	infos := getAps(db, in.Longitude, in.Latitude)
	return &fetch.ApReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid}, Infos: infos}, nil
}

func main() {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	fetch.RegisterFetchServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
