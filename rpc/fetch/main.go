package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"

	"golang.org/x/net/context"

	"google.golang.org/grpc"

	aliyun "../../aliyun"
	common "../../proto/common"
	fetch "../../proto/fetch"
	util "../../util"
	simplejson "github.com/bitly/go-simplejson"
	_ "github.com/go-sql-driver/mysql"
)

const (
	maxDistance   = 3000
	addressPrefix = "广东省东莞市东莞市市辖区"
)

type server struct{}

var db *sql.DB

func getNewsTag(db *sql.DB, id int64) string {
	rows, err := db.Query("SELECT t.content FROM news_tags n, tags t WHERE n.tid = t.id AND n.nid = ?", id)
	if err != nil {
		log.Printf("query failed:%v", err)
		return ""
	}
	defer rows.Close()

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

func genTypeQuery(ctype int32) string {
	switch ctype {
	default:
		return " AND review = 0 "
	case 1:
		return " AND review = 1 AND deleted = 0 "
	case 2:
		return " AND review = 1 AND deleted = 1 "
	}
}

func getTotalNews(db *sql.DB, ctype int32) int64 {
	query := "SELECT COUNT(id) FROM news WHERE 1 = 1 " + genTypeQuery(ctype)
	var total int64
	err := db.QueryRow(query).Scan(&total)
	if err != nil {
		log.Printf("get total failed:%v", err)
		return 0
	}
	return total
}

func getTotalVideos(db *sql.DB, ctype int32) int64 {
	query := "SELECT COUNT(vid) FROM youku_video WHERE 1 = 1 " + genTypeQuery(ctype)
	var total int64
	err := db.QueryRow(query).Scan(&total)
	if err != nil {
		log.Printf("get total failed:%v", err)
		return 0
	}
	return total
}

func getTotalTags(db *sql.DB) int64 {
	query := "SELECT COUNT(id) FROM tags WHERE deleted = 0"
	var total int64
	err := db.QueryRow(query).Scan(&total)
	if err != nil {
		log.Printf("get total tags failed:%v", err)
		return 0
	}
	return total
}

func getTotalAps(db *sql.DB) int64 {
	query := "SELECT COUNT(id) FROM ap "
	var total int64
	err := db.QueryRow(query).Scan(&total)
	if err != nil {
		log.Printf("get total ap failed:%v", err)
		return 0
	}
	return total
}

func getTotalTemplates(db *sql.DB) int64 {
	query := "SELECT COUNT(id) FROM template "
	var total int64
	err := db.QueryRow(query).Scan(&total)
	if err != nil {
		log.Printf("get total ap failed:%v", err)
		return 0
	}
	return total
}

func getTotalUsers(db *sql.DB) int64 {
	query := "SELECT COUNT(uid) FROM user "
	var total int64
	err := db.QueryRow(query).Scan(&total)
	if err != nil {
		log.Printf("get total user failed:%v", err)
		return 0
	}
	return total
}

func getTotalBanners(db *sql.DB) int64 {
	query := "SELECT COUNT(id) FROM banner WHERE deleted = 0"
	var total int64
	err := db.QueryRow(query).Scan(&total)
	if err != nil {
		log.Printf("get total failed:%v", err)
		return 0
	}
	return total
}

func getReviewNews(db *sql.DB, seq, num, ctype int64) []*fetch.NewsInfo {
	var infos []*fetch.NewsInfo
	query := "SELECT id, title, ctime, source FROM news WHERE 1 = 1 " + genTypeQuery(int32(ctype))
	query += " ORDER BY id DESC LIMIT " + strconv.Itoa(int(seq)) + "," + strconv.Itoa(int(num))
	log.Printf("query string:%s", query)
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("query failed:%v", err)
		return infos
	}
	defer rows.Close()

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
	log.Printf("request uid:%d, sid:%s seq:%d, num:%d type:%d", in.Head.Uid, in.Head.Sid, in.Seq, in.Num, in.Type)
	news := getReviewNews(db, in.Seq, int64(in.Num), int64(in.Type))
	total := getTotalNews(db, in.Type)
	return &fetch.NewsReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid}, Infos: news, Total: total}, nil
}

func getTags(db *sql.DB, seq, num int64) []*fetch.TagInfo {
	var infos []*fetch.TagInfo
	query := "SELECT id, content FROM tags WHERE deleted = 0 ORDER BY id DESC LIMIT " + strconv.Itoa(int(seq)) + "," + strconv.Itoa(int(num))
	log.Printf("query string:%s", query)
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("query failed:%v", err)
		return infos
	}
	defer rows.Close()

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
	log.Printf("request uid:%d, sid:%s seq:%d, num:%d", in.Head.Uid, in.Head.Sid, in.Seq, in.Num)
	tags := getTags(db, in.Seq, int64(in.Num))
	total := getTotalTags(db)
	return &fetch.TagsReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid}, Infos: tags, Total: total}, nil
}

func getAps(db *sql.DB, longitude, latitude float64) []*fetch.ApInfo {
	var infos []*fetch.ApInfo
	rows, err := db.Query("SELECT id, bd_lon, bd_lat, address FROM ap WHERE bd_lon > ? - 0.1 AND bd_lon < ? + 0.1 AND bd_lat > ? - 0.1 AND bd_lat < ? + 0.1 GROUP BY bd_lon, bd_lat ORDER BY (POW(ABS(bd_lon - ?), 2) + POW(ABS(bd_lat - ?), 2)) LIMIT 20", longitude, longitude, latitude, latitude, longitude, latitude)
	if err != nil {
		log.Printf("query failed:%v", err)
		return infos
	}
	defer rows.Close()

	var p1 util.Point
	p1.Longitude = longitude
	p1.Latitude = latitude
	for rows.Next() {
		var info fetch.ApInfo
		err = rows.Scan(&info.Id, &info.Longitude, &info.Latitude, &info.Address)
		if err != nil {
			log.Printf("scan rows failed: %v", err)
			return infos
		}
		var p2 util.Point
		p2.Longitude = info.Longitude
		p2.Latitude = info.Latitude
		distance := util.GetDistance(p1, p2)
		if strings.HasPrefix(info.Address, addressPrefix) {
			info.Address = info.Address[len(addressPrefix):]
		}

		log.Printf("id:%s longitude:%f latitude:%f ", info.Id, info.Longitude, info.Latitude)
		if distance > maxDistance {
			break
		}
		infos = append(infos, &info)
	}
	return infos
}

func (s *server) FetchAps(ctx context.Context, in *fetch.ApRequest) (*fetch.ApReply, error) {
	log.Printf("request uid:%d, sid:%s longitude:%f latitude:%f", in.Head.Uid, in.Head.Sid, in.Longitude, in.Latitude)
	infos := getAps(db, in.Longitude, in.Latitude)
	return &fetch.ApReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid}, Infos: infos}, nil
}

func getWifis(db *sql.DB, longitude, latitude float64) []*common.WifiInfo {
	var infos []*common.WifiInfo
	rows, err := db.Query("SELECT ssid, username, password, longitude, latitude FROM wifi WHERE longitude > ? - 0.1 AND longitude < ? + 0.1 AND latitude > ? - 0.1 AND latitude < ? + 0.1 ORDER BY (POW(ABS(longitude - ?), 2) + POW(ABS(latitude - ?), 2)) LIMIT 20", longitude, longitude, latitude, latitude, longitude, latitude)
	if err != nil {
		log.Printf("query failed:%v", err)
		return infos
	}
	defer rows.Close()

	var p1 util.Point
	p1.Longitude = longitude
	p1.Latitude = latitude
	for rows.Next() {
		var info common.WifiInfo
		err = rows.Scan(&info.Ssid, &info.Username, &info.Password, &info.Longitude, &info.Latitude)
		if err != nil {
			log.Printf("scan rows failed: %v", err)
			return infos
		}
		var p2 util.Point
		p2.Longitude = info.Longitude
		p2.Latitude = info.Latitude
		distance := util.GetDistance(p1, p2)

		log.Printf("ssid:%s username:%s password:%s longitude:%f latitude:%f ", info.Ssid, info.Username, info.Password, info.Longitude, info.Latitude)
		if distance > maxDistance {
			break
		}
		infos = append(infos, &info)
	}
	return infos
}

func (s *server) FetchWifi(ctx context.Context, in *fetch.WifiRequest) (*fetch.WifiReply, error) {
	log.Printf("request uid:%d, sid:%s longitude:%f latitude:%f", in.Head.Uid, in.Head.Sid, in.Longitude, in.Latitude)
	infos := getWifis(db, in.Longitude, in.Latitude)
	return &fetch.WifiReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid}, Infos: infos}, nil
}

func getApStat(db *sql.DB, seq, num int32) []*fetch.ApStatInfo {
	var infos []*fetch.ApStatInfo
	query := "SELECT id, address, mac, count, bandwidth, online FROM ap ORDER BY id DESC LIMIT " + strconv.Itoa(int(seq)) + "," + strconv.Itoa(int(num))
	log.Printf("query string:%s", query)
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("query failed:%v", err)
		return infos
	}
	defer rows.Close()

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
	log.Printf("request uid:%d, sid:%s seq:%d num:%d", in.Head.Uid, in.Head.Sid, in.Seq, in.Num)
	infos := getApStat(db, int32(in.Seq), in.Num)
	total := getTotalAps(db)
	return &fetch.ApStatReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid}, Infos: infos, Total: total}, nil
}

func getUsers(db *sql.DB, seq, num int64) []*fetch.UserInfo {
	var infos []*fetch.UserInfo
	query := "SELECT uid, phone, udid, atime, remark, times, duration, traffic, aptime, aid FROM user ORDER BY uid DESC LIMIT " + strconv.Itoa(int(seq)) + "," + strconv.Itoa(int(num))
	log.Printf("query string:%s", query)
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("query failed:%v", err)
		return infos
	}
	defer rows.Close()

	for rows.Next() {
		var info fetch.UserInfo
		var aid int
		err = rows.Scan(&info.Id, &info.Phone, &info.Imei, &info.Active, &info.Remark,
			&info.Times, &info.Duration, &info.Traffic, &info.Utime, &aid)
		if err != nil {
			log.Printf("scan rows failed: %v", err)
			continue
		}
		if aid != 0 {
			err := db.QueryRow("SELECT address FROM ap WHERE id = ?", aid).Scan(&info.Address)
			if err != nil {
				log.Printf("get ap address failed aid:%d err:%v", aid, err)
			}
		}
		infos = append(infos, &info)
		log.Printf("uid:%d phone:%s udid:%s active:%s remark:%s", info.Id, info.Phone, info.Imei, info.Active, info.Remark)
	}
	return infos
}

func (s *server) FetchUsers(ctx context.Context, in *fetch.CommRequest) (*fetch.UserReply, error) {
	log.Printf("request uid:%d, sid:%s seq:%d num:%d", in.Head.Uid, in.Head.Sid, in.Seq, in.Num)
	infos := getUsers(db, in.Seq, int64(in.Num))
	total := getTotalUsers(db)
	return &fetch.UserReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid}, Infos: infos, Total: total}, nil
}

func getTemplates(db *sql.DB, seq, num int32) []*fetch.TemplateInfo {
	var infos []*fetch.TemplateInfo
	query := "SELECT id, title, content, online FROM template ORDER BY id DESC LIMIT " + strconv.Itoa(int(seq)) + "," + strconv.Itoa(int(num))
	log.Printf("query string:%s", query)
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("query failed:%v", err)
		return infos
	}
	defer rows.Close()

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
	log.Printf("request uid:%d, sid:%s seq:%d num:%d", in.Head.Uid, in.Head.Sid, in.Seq, in.Num)
	infos := getTemplates(db, int32(in.Seq), in.Num)
	total := getTotalTemplates(db)
	return &fetch.TemplateReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid}, Infos: infos, Total: total}, nil
}

func getVideos(db *sql.DB, seq, num, ctype int32) []*fetch.VideoInfo {
	var infos []*fetch.VideoInfo
	query := "SELECT vid, img, title, dst, ctime, source, duration FROM youku_video WHERE 1 = 1 " + genTypeQuery(ctype)
	query += " ORDER BY id DESC LIMIT " + strconv.Itoa(int(seq)) + "," + strconv.Itoa(int(num))
	log.Printf("query string:%s", query)
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("query failed:%v", err)
		return infos
	}
	defer rows.Close()

	for rows.Next() {
		var info fetch.VideoInfo
		err = rows.Scan(&info.Id, &info.Img, &info.Title, &info.Dst, &info.Ctime, &info.Source, &info.Duration)
		if err != nil {
			log.Printf("scan rows failed: %v", err)
			return infos
		}
		infos = append(infos, &info)
		log.Printf("id:%d title:%s dst:%s ", info.Id, info.Title, info.Dst)
	}
	return infos
}
func (s *server) FetchVideos(ctx context.Context, in *fetch.CommRequest) (*fetch.VideoReply, error) {
	log.Printf("request uid:%d, sid:%s seq:%d num:%d", in.Head.Uid, in.Head.Sid, in.Seq, in.Num)
	infos := getVideos(db, int32(in.Seq), in.Num, in.Type)
	total := getTotalVideos(db, in.Type)
	return &fetch.VideoReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid}, Infos: infos, Total: total}, nil
}

func getBanners(db *sql.DB, seq, num int32) []*common.BannerInfo {
	var infos []*common.BannerInfo
	query := "SELECT id, img, dst, online, priority FROM banner WHERE deleted = 0 ORDER BY priority DESC LIMIT " + strconv.Itoa(int(seq)) + "," + strconv.Itoa(int(num))
	log.Printf("query string:%s", query)
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("query failed:%v", err)
		return infos
	}
	defer rows.Close()

	for rows.Next() {
		var info common.BannerInfo
		err = rows.Scan(&info.Id, &info.Img, &info.Dst, &info.Online, &info.Priority)
		if err != nil {
			log.Printf("scan rows failed: %v", err)
			return infos
		}
		infos = append(infos, &info)
		log.Printf("id:%d img:%s dst:%s Online:%d priority:%d\n", info.Id, info.Img, info.Dst, info.Online, info.Priority)
	}
	return infos
}

func (s *server) FetchBanners(ctx context.Context, in *fetch.CommRequest) (*fetch.BannerReply, error) {
	log.Printf("request uid:%d, sid:%s seq:%d num:%d", in.Head.Uid, in.Head.Sid, in.Seq, in.Num)
	infos := getBanners(db, int32(in.Seq), in.Num)
	total := getTotalBanners(db)
	return &fetch.BannerReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid}, Infos: infos, Total: total}, nil
}

func genSsidStr(ssids []string) string {
	var str string
	for i := 0; i < len(ssids); i++ {
		str += "'" + ssids[i] + "'"
		if i < len(ssids)-1 {
			str += ","
		}
	}
	return str
}

func (s *server) FetchWifiPass(ctx context.Context, in *fetch.WifiPassRequest) (*fetch.WifiPassReply, error) {
	log.Printf("FetchWifiPass request uid:%d, longitude:%f latitude:%f ssid:%v",
		in.Head.Uid, in.Longitude, in.Latitude, in.Ssids)
	ssids := genSsidStr(in.Ssids)
	query := fmt.Sprintf("SELECT ssid, password FROM wifi WHERE longitude > %f - 0.01 AND longitude < %f + 0.01 AND latitude > %f - 0.01 AND latitude < %f + 0.01 AND ssid IN (%s) AND deleted = 0",
		in.Longitude, in.Longitude, in.Latitude, in.Latitude, ssids)
	log.Printf("FetchWifiPass query:%s", query)
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("FetchWifiPass query failed:%v", err)
		return &fetch.WifiPassReply{Head: &common.Head{Retcode: 1}}, nil
	}
	defer rows.Close()

	var wifis []*fetch.WifiPass
	for rows.Next() {
		var info fetch.WifiPass
		err := rows.Scan(&info.Ssid, &info.Pass)
		if err != nil {
			log.Printf("FetchWifiPass scan failed:%v", err)
			continue
		}
		wifis = append(wifis, &info)
	}

	return &fetch.WifiPassReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid}, Wifis: wifis}, nil
}

func (s *server) FetchStsCredentials(ctx context.Context, in *fetch.CommRequest) (*fetch.StsReply, error) {
	res := aliyun.FetchStsCredentials()
	log.Printf("FetchStsCredentials resp:%s", res)
	if res == "" {
		return &fetch.StsReply{Head: &common.Head{Retcode: 1}}, errors.New("fetch sts failed")
	}
	js, err := simplejson.NewJson([]byte(res))
	if err != nil {
		log.Printf("FetchStsCredentials parse resp failed:%v", err)
		return &fetch.StsReply{Head: &common.Head{Retcode: 1}}, err
	}
	credential := js.Get("Credentials")
	var cred fetch.StsCredential
	cred.Accesskeysecret, _ = credential.Get("AccessKeySecret").String()
	cred.Accesskeyid, _ = credential.Get("AccessKeyId").String()
	cred.Expiration, _ = credential.Get("Expiration").String()
	cred.Securitytoken, _ = credential.Get("SecurityToken").String()
	return &fetch.StsReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid},
		Credential: &cred}, nil
}

func main() {
	lis, err := net.Listen("tcp", util.FetchServerPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	db, err = util.InitDB(true)
	if err != nil {
		log.Fatalf("failed to init db connection: %v", err)
	}
	db.SetMaxIdleConns(util.MaxIdleConns)

	kv := util.InitRedis()
	go util.ReportHandler(kv, util.FetchServerName, util.FetchServerPort)

	s := grpc.NewServer()
	fetch.RegisterFetchServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
