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

func getNewsTag(db *sql.DB, id int64) string {
	rows, err := db.Query("SELECT t.content FROM news_tags n, tags t WHERE n.tid = t.id AND n.nid = ?", id)
	if err != nil {
		log.Printf("query failed:%v", err)
		return ""
	}

	var tags string
	for rows.Next() {
		var tag string
		err = rows.Scan(&tag)
		if err != nil {
			log.Printf("scan rows failed: %v", err)
			return tags
		}
		if len(tags) > 0 {
			tags += "," + tag
		} else {
			tags += tag
		}
	}
	return tags
}

func getReviewNews(db *sql.DB, seq, num, ctype int64) []*fetch.NewsInfo {
	var infos []*fetch.NewsInfo
	query := "SELECT id, title, ctime, source FROM news WHERE 1 = 1 "
	switch ctype {
	default:
		query += " AND review = 0 "
	case 1:
		query += " AND review = 1 AND deleted = 0 "
	case 2:
		query += " AND review = 1 AND deleted = 1 "
	}
	if seq != 0 {
		query += " AND id < " + strconv.Itoa(int(seq))
	}
	query += " ORDER BY id DESC LIMIT " + strconv.Itoa(int(num))
	log.Printf("query string:%s", query)
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("query failed:%v", err)
		return infos
	}

	for rows.Next() {
		var info fetch.NewsInfo
		err = rows.Scan(&info.Id, &info.Title, &info.Ctime, &info.Source)
		if err != nil {
			log.Printf("scan rows failed: %v", err)
			return infos
		}
		infos = append(infos, &info)
		log.Printf("id:%s title:%s ctime:%s source:%s ", info.Id, info.Title, info.Ctime, info.Source)
		if ctype == 1 {
			info.Tag = getNewsTag(db, info.Id)
		}

	}
	return infos
}

func (s *server) FetchReviewNews(ctx context.Context, in *fetch.CommRequest) (*fetch.NewsReply, error) {
	db, err := util.InitDB(true)
	if err != nil {
		log.Printf("connect mysql failed:%v", err)
		return &fetch.NewsReply{Head: &common.Head{Retcode: 1}}, err
	}
	log.Printf("request uid:%d, sid:%s seq:%d, num:%d type:%d", in.Head.Uid, in.Head.Sid, in.Seq, in.Num, in.Type)
	news := getReviewNews(db, in.Seq, int64(in.Num), int64(in.Type))
	return &fetch.NewsReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid}, Infos: news}, nil
}

func getTags(db *sql.DB, seq, num int64) []*fetch.TagInfo {
	var infos []*fetch.TagInfo
	query := "SELECT id, content FROM tags WHERE 1 = 1 "
	if seq != 0 {
		query += " AND id < " + strconv.Itoa(int(seq))
	}
	query += " ORDER BY id DESC LIMIT " + strconv.Itoa(int(num))
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
	rows, err := db.Query("SELECT id, bd_lon, bd_lat FROM ap WHERE bd_lon > ? - 0.1 AND bd_lon < ? + 0.1 AND bd_lat > ? - 0.1 AND bd_lat < ? + 0.1 ORDER BY (POW(ABS(bd_lon - ?), 2) + POW(ABS(bd_lat - ?), 2)) LIMIT 20", longitude, longitude, latitude, latitude, longitude, latitude)
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

func getApStat(db *sql.DB, seq, num int32) []*fetch.ApStatInfo {
	var infos []*fetch.ApStatInfo
	query := "SELECT id, address, mac, count, bandwidth, online FROM ap WHERE 1 = 1 "
	if seq != 0 {
		query += " AND id < " + strconv.Itoa(int(seq))
	}
	query += " ORDER BY id DESC LIMIT " + strconv.Itoa(int(num))
	log.Printf("query string:%s", query)
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("query failed:%v", err)
		return infos
	}

	for rows.Next() {
		var info fetch.ApStatInfo
		err = rows.Scan(&info.Id, &info.Address, &info.Mac, &info.Count, &info.Bandwidth, &info.Online)
		if err != nil {
			log.Printf("scan rows failed: %v", err)
			return infos
		}
		infos = append(infos, &info)
		log.Printf("id:%s address:%s mac:%s count:%d bandwidth:%d online:%d ", info.Id, info.Address, info.Mac, info.Count, info.Bandwidth, info.Online)
	}
	return infos
}

func (s *server) FetchApStat(ctx context.Context, in *fetch.CommRequest) (*fetch.ApStatReply, error) {
	db, err := util.InitDB(true)
	if err != nil {
		log.Printf("connect mysql failed:%v", err)
		return &fetch.ApStatReply{Head: &common.Head{Retcode: 1}}, err
	}
	log.Printf("request uid:%d, sid:%s seq:%d num:%d", in.Head.Uid, in.Head.Sid, in.Seq, in.Num)
	infos := getApStat(db, int32(in.Seq), in.Num)
	return &fetch.ApStatReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid}, Infos: infos}, nil
}

func getUsers(db *sql.DB, seq, num int64) []*fetch.UserInfo {
	var infos []*fetch.UserInfo
	query := "SELECT uid, phone, udid, atime, remark FROM user WHERE 1 = 1 "
	if seq != 0 {
		query += " AND uid < " + strconv.Itoa(int(seq))
	}
	query += " ORDER BY uid DESC LIMIT " + strconv.Itoa(int(num))
	log.Printf("query string:%s", query)
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("query failed:%v", err)
		return infos
	}

	for rows.Next() {
		var info fetch.UserInfo
		err = rows.Scan(&info.Id, &info.Phone, &info.Imei, &info.Active, &info.Remark)
		if err != nil {
			log.Printf("scan rows failed: %v", err)
			return infos
		}
		infos = append(infos, &info)
		log.Printf("uid:%d phone:%s udid:%s active:%s remark:%s", info.Id, info.Phone, info.Imei, info.Active, info.Remark)
	}
	return infos
}

func (s *server) FetchUsers(ctx context.Context, in *fetch.CommRequest) (*fetch.UserReply, error) {
	db, err := util.InitDB(true)
	if err != nil {
		log.Printf("connect mysql failed:%v", err)
		return &fetch.UserReply{Head: &common.Head{Retcode: 1}}, err
	}
	log.Printf("request uid:%d, sid:%s seq:%d num:%d", in.Head.Uid, in.Head.Sid, in.Seq, in.Num)
	infos := getUsers(db, in.Seq, int64(in.Num))
	return &fetch.UserReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid}, Infos: infos}, nil
}

func getTemplates(db *sql.DB, seq, num int32) []*fetch.TemplateInfo {
	var infos []*fetch.TemplateInfo
	query := "SELECT id, title, content, online FROM template WHERE 1 = 1 "
	if seq != 0 {
		query += " AND id < " + strconv.Itoa(int(seq))
	}
	query += " ORDER BY id DESC LIMIT " + strconv.Itoa(int(num))
	log.Printf("query string:%s", query)
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("query failed:%v", err)
		return infos
	}

	for rows.Next() {
		var info fetch.TemplateInfo
		err = rows.Scan(&info.Id, &info.Title, &info.Content, &info.Online)
		if err != nil {
			log.Printf("scan rows failed: %v", err)
			return infos
		}
		infos = append(infos, &info)
		log.Printf("id:%d title:%s Online:%d ", info.Id, info.Title, info.Online)
	}
	return infos
}

func (s *server) FetchTemplates(ctx context.Context, in *fetch.CommRequest) (*fetch.TemplateReply, error) {
	db, err := util.InitDB(true)
	if err != nil {
		log.Printf("connect mysql failed:%v", err)
		return &fetch.TemplateReply{Head: &common.Head{Retcode: 1}}, err
	}
	log.Printf("request uid:%d, sid:%s seq:%d num:%d", in.Head.Uid, in.Head.Sid, in.Seq, in.Num)
	infos := getTemplates(db, int32(in.Seq), in.Num)
	return &fetch.TemplateReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid}, Infos: infos}, nil
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
