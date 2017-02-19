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

	"Server/aliyun"
	"Server/proto/common"
	"Server/proto/fetch"
	"Server/util"

	simplejson "github.com/bitly/go-simplejson"
	_ "github.com/go-sql-driver/mysql"
	nsq "github.com/nsqio/go-nsq"
)

const (
	maxDistance   = 3000
	addressPrefix = "广东省东莞市东莞市市辖区"
	provinceType  = 0
	townType      = 1
	districtType  = 2
	highPrice     = 500000
	priceTips     = "温馨提示:获奖者拥有奖品10年免费使用权"
)

var expressList = []string{
	"",
	"顺丰速递",
	"京东快递",
	"申通快递",
	"圆通快递",
	"中通快递",
	"EMS快递",
	"韵达快递",
	"百世汇通",
	"当当网",
	"苏宁",
}

type server struct{}

var db *sql.DB
var w *nsq.Producer

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

func genTypeQuery(ctype int64) string {
	switch ctype {
	default:
		return " AND review = 0 "
	case 1:
		return " AND review = 1 AND deleted = 0 "
	case 2:
		return " AND review = 1 AND deleted = 1 "
	}
}

func getTotalNews(db *sql.DB, ctype, stype int64, search string) int64 {
	query := "SELECT COUNT(id) FROM news WHERE 1 = 1 " + genTypeQuery(ctype)
	if stype != 0 {
		query += " AND stype = 10 "
	}
	if search != "" {
		query += " AND title LIKE '%" + search + "%' "
	}
	var total int64
	err := db.QueryRow(query).Scan(&total)
	if err != nil {
		log.Printf("get total failed:%v", err)
		return 0
	}
	return total
}

func getTotalVideos(db *sql.DB, ctype int64) int64 {
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

func getTotalBanners(db *sql.DB, btype int64) int64 {
	query := "SELECT COUNT(id) FROM banner WHERE deleted = 0 AND type = " +
		strconv.Itoa(int(btype))
	var total int64
	err := db.QueryRow(query).Scan(&total)
	if err != nil {
		log.Printf("get total failed:%v", err)
		return 0
	}
	return total
}

func getReviewNews(db *sql.DB, seq, num, ctype, stype int64, search string) []*fetch.NewsInfo {
	var infos []*fetch.NewsInfo
	query := "SELECT id, title, ctime, source FROM news WHERE 1 = 1 " +
		genTypeQuery(ctype)
	if stype != 0 {
		query += " AND stype = 10 "
	}
	if search != "" {
		query += " AND title LIKE '%" + search + "%' "
	}
	query += " ORDER BY id DESC LIMIT " + strconv.Itoa(int(seq)) + "," +
		strconv.Itoa(int(num))
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
		log.Printf("id:%s title:%s ctime:%s source:%s ", info.Id, info.Title,
			info.Ctime, info.Source)
		if ctype == 1 {
			info.Tag = getNewsTag(db, info.Id)
		}

	}
	return infos
}

func (s *server) FetchReviewNews(ctx context.Context, in *common.CommRequest) (*fetch.NewsReply, error) {
	util.PubRPCRequest(w, "fetch", "FetchReviewNews")
	log.Printf("request uid:%d, sid:%s seq:%d, num:%d type:%d search:%s", in.Head.Uid,
		in.Head.Sid, in.Seq, in.Num, in.Type, in.Search)
	news := getReviewNews(db, in.Seq, in.Num, in.Type, in.Subtype, in.Search)
	total := getTotalNews(db, in.Type, in.Subtype, in.Search)
	util.PubRPCSuccRsp(w, "fetch", "FetchReviewNews")
	return &fetch.NewsReply{
		Head:  &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid},
		Infos: news, Total: total}, nil
}

func getTags(db *sql.DB, seq, num int64) []*fetch.TagInfo {
	var infos []*fetch.TagInfo
	query := "SELECT id, content FROM tags WHERE deleted = 0 ORDER BY id DESC LIMIT " +
		strconv.Itoa(int(seq)) + "," + strconv.Itoa(int(num))
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

func (s *server) FetchTags(ctx context.Context, in *common.CommRequest) (*fetch.TagsReply, error) {
	util.PubRPCRequest(w, "fetch", "FetchTags")
	log.Printf("request uid:%d, sid:%s seq:%d, num:%d", in.Head.Uid,
		in.Head.Sid, in.Seq, in.Num)
	tags := getTags(db, in.Seq, in.Num)
	total := getTotalTags(db)
	util.PubRPCSuccRsp(w, "fetch", "FetchTags")
	return &fetch.TagsReply{
		Head:  &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid},
		Infos: tags, Total: total}, nil
}

func getAps(db *sql.DB, longitude, latitude float64) []*fetch.ApInfo {
	var infos []*fetch.ApInfo
	rows, err := db.Query("SELECT id, longitude, latitude, address FROM ap WHERE longitude > ? - 0.1 AND longitude < ? + 0.1 AND latitude > ? - 0.1 AND latitude < ? + 0.1 GROUP BY longitude, latitude ORDER BY (POW(ABS(longitude - ?), 2) + POW(ABS(latitude- ?), 2)) LIMIT 20",
		longitude, longitude, latitude, latitude, longitude, latitude)
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

		log.Printf("id:%s longitude:%f latitude:%f ", info.Id, info.Longitude,
			info.Latitude)
		if distance > maxDistance {
			break
		}
		infos = append(infos, &info)
	}
	return infos
}

func getAllAps(db *sql.DB) []*fetch.ApInfo {
	var infos []*fetch.ApInfo
	rows, err := db.Query("SELECT id, longitude, latitude, address FROM ap GROUP BY longitude, latitude")
	if err != nil {
		log.Printf("getAllAps query failed:%v", err)
		return infos
	}
	defer rows.Close()
	for rows.Next() {
		var info fetch.ApInfo
		err := rows.Scan(&info.Id, &info.Longitude, &info.Latitude, &info.Address)
		if err != nil {
			continue
		}
		infos = append(infos, &info)
	}
	return infos
}

func (s *server) FetchAps(ctx context.Context, in *fetch.ApRequest) (*fetch.ApReply, error) {
	util.PubRPCRequest(w, "fetch", "FetchAps")
	log.Printf("request uid:%d, sid:%s longitude:%f latitude:%f", in.Head.Uid,
		in.Head.Sid, in.Longitude, in.Latitude)
	infos := getAps(db, in.Longitude, in.Latitude)
	util.PubRPCSuccRsp(w, "fetch", "FetchAps")
	return &fetch.ApReply{
		Head:  &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid},
		Infos: infos}, nil
}

func (s *server) FetchAllAps(ctx context.Context, in *common.CommRequest) (*fetch.ApReply, error) {
	util.PubRPCRequest(w, "fetch", "FetchAllAps")
	infos := getAllAps(db)
	util.PubRPCSuccRsp(w, "fetch", "FetchAllAps")
	return &fetch.ApReply{
		Head:  &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid},
		Infos: infos}, nil
}

func getWifis(db *sql.DB, longitude, latitude float64) []*common.WifiInfo {
	var infos []*common.WifiInfo
	rows, err := db.Query("SELECT ssid, username, password, longitude, latitude FROM wifi WHERE longitude > ? - 0.1 AND longitude < ? + 0.1 AND latitude > ? - 0.1 AND latitude < ? + 0.1 ORDER BY (POW(ABS(longitude - ?), 2) + POW(ABS(latitude - ?), 2)) LIMIT 20",
		longitude, longitude, latitude, latitude, longitude, latitude)
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
		err = rows.Scan(&info.Ssid, &info.Username, &info.Password,
			&info.Longitude, &info.Latitude)
		if err != nil {
			log.Printf("scan rows failed: %v", err)
			return infos
		}
		var p2 util.Point
		p2.Longitude = info.Longitude
		p2.Latitude = info.Latitude
		distance := util.GetDistance(p1, p2)

		log.Printf("ssid:%s username:%s password:%s longitude:%f latitude:%f ",
			info.Ssid, info.Username, info.Password, info.Longitude,
			info.Latitude)
		if distance > maxDistance {
			break
		}
		infos = append(infos, &info)
	}
	return infos
}

func (s *server) FetchWifi(ctx context.Context, in *fetch.WifiRequest) (*fetch.WifiReply, error) {
	util.PubRPCRequest(w, "fetch", "FetchWifi")
	log.Printf("request uid:%d, sid:%s longitude:%f latitude:%f", in.Head.Uid,
		in.Head.Sid, in.Longitude, in.Latitude)
	infos := getWifis(db, in.Longitude, in.Latitude)
	util.PubRPCSuccRsp(w, "fetch", "FetchWifi")
	return &fetch.WifiReply{
		Head:  &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid},
		Infos: infos}, nil
}

func getApStat(db *sql.DB, seq, num int64) []*fetch.ApStatInfo {
	var infos []*fetch.ApStatInfo
	query := "SELECT id, address, mac, count, bandwidth, online FROM ap ORDER BY id DESC LIMIT " +
		strconv.Itoa(int(seq)) + "," + strconv.Itoa(int(num))
	log.Printf("query string:%s", query)
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("query failed:%v", err)
		return infos
	}
	defer rows.Close()

	for rows.Next() {
		var info fetch.ApStatInfo
		err = rows.Scan(&info.Id, &info.Address, &info.Mac, &info.Count,
			&info.Bandwidth, &info.Online)
		if err != nil {
			log.Printf("scan rows failed: %v", err)
			return infos
		}
		infos = append(infos, &info)
		log.Printf("id:%s address:%s mac:%s count:%d bandwidth:%d online:%d ",
			info.Id, info.Address, info.Mac, info.Count, info.Bandwidth,
			info.Online)
	}
	return infos
}

func (s *server) FetchApStat(ctx context.Context, in *common.CommRequest) (*fetch.ApStatReply, error) {
	util.PubRPCRequest(w, "fetch", "FetchApStat")
	log.Printf("request uid:%d, sid:%s seq:%d num:%d", in.Head.Uid,
		in.Head.Sid, in.Seq, in.Num)
	infos := getApStat(db, in.Seq, in.Num)
	total := getTotalAps(db)
	util.PubRPCSuccRsp(w, "fetch", "FetchApStat")
	return &fetch.ApStatReply{
		Head:  &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid},
		Infos: infos, Total: total}, nil
}

func getUsers(db *sql.DB, seq, num int64) []*fetch.UserInfo {
	var infos []*fetch.UserInfo
	query := "SELECT uid, phone, udid, atime, remark, times, duration, traffic, aptime, aid FROM user ORDER BY uid DESC LIMIT " +
		strconv.Itoa(int(seq)) + "," + strconv.Itoa(int(num))
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
			err := db.QueryRow("SELECT address FROM ap WHERE id = ?", aid).
				Scan(&info.Address)
			if err != nil {
				log.Printf("get ap address failed aid:%d err:%v", aid, err)
			}
		}
		infos = append(infos, &info)
		log.Printf("uid:%d phone:%s udid:%s active:%s remark:%s", info.Id,
			info.Phone, info.Imei, info.Active, info.Remark)
	}
	return infos
}

func (s *server) FetchUsers(ctx context.Context, in *common.CommRequest) (*fetch.UserReply, error) {
	util.PubRPCRequest(w, "fetch", "FetchUsers")
	log.Printf("request uid:%d, sid:%s seq:%d num:%d", in.Head.Uid, in.Head.Sid,
		in.Seq, in.Num)
	infos := getUsers(db, in.Seq, in.Num)
	total := getTotalUsers(db)
	util.PubRPCSuccRsp(w, "fetch", "FetchUsers")
	return &fetch.UserReply{
		Head:  &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid},
		Infos: infos, Total: total}, nil
}

func getTemplates(db *sql.DB, seq, num int64) []*fetch.TemplateInfo {
	var infos []*fetch.TemplateInfo
	query := "SELECT id, title, content, online FROM template ORDER BY id DESC LIMIT " +
		strconv.Itoa(int(seq)) + "," + strconv.Itoa(int(num))
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

func (s *server) FetchTemplates(ctx context.Context, in *common.CommRequest) (*fetch.TemplateReply, error) {
	util.PubRPCRequest(w, "fetch", "FetchTemplates")
	log.Printf("request uid:%d, sid:%s seq:%d num:%d", in.Head.Uid, in.Head.Sid,
		in.Seq, in.Num)
	infos := getTemplates(db, in.Seq, in.Num)
	total := getTotalTemplates(db)
	util.PubRPCSuccRsp(w, "fetch", "FetchTemplates")
	return &fetch.TemplateReply{
		Head:  &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid},
		Infos: infos, Total: total}, nil
}

func getVideos(db *sql.DB, seq, num, ctype int64) []*fetch.VideoInfo {
	var infos []*fetch.VideoInfo
	query := "SELECT vid, img, title, dst, ctime, source, duration FROM youku_video WHERE 1 = 1 " +
		genTypeQuery(ctype)
	query += " ORDER BY vid DESC LIMIT " + strconv.Itoa(int(seq)) + "," +
		strconv.Itoa(int(num))
	log.Printf("query string:%s", query)
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("query failed:%v", err)
		return infos
	}
	defer rows.Close()

	for rows.Next() {
		var info fetch.VideoInfo
		err = rows.Scan(&info.Id, &info.Img, &info.Title, &info.Dst, &info.Ctime,
			&info.Source, &info.Duration)
		if err != nil {
			log.Printf("scan rows failed: %v", err)
			return infos
		}
		infos = append(infos, &info)
		log.Printf("id:%d title:%s dst:%s ", info.Id, info.Title, info.Dst)
	}
	return infos
}

func (s *server) FetchVideos(ctx context.Context, in *common.CommRequest) (*fetch.VideoReply, error) {
	util.PubRPCRequest(w, "fetch", "FetchVideos")
	log.Printf("request uid:%d, sid:%s seq:%d num:%d", in.Head.Uid, in.Head.Sid,
		in.Seq, in.Num)
	infos := getVideos(db, in.Seq, in.Num, in.Type)
	total := getTotalVideos(db, in.Type)
	util.PubRPCSuccRsp(w, "fetch", "FetchVideos")
	return &fetch.VideoReply{
		Head:  &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid},
		Infos: infos, Total: total}, nil
}

func getBanners(db *sql.DB, seq, btype, num int64) []*common.BannerInfo {
	var infos []*common.BannerInfo
	query := fmt.Sprintf("SELECT id, img, dst, online, priority, title, etime, dbg FROM banner WHERE deleted = 0 AND type = %d ORDER BY priority DESC LIMIT %d, %d",
		btype, seq, num)
	log.Printf("query string:%s", query)
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("query failed:%v", err)
		return infos
	}
	defer rows.Close()

	for rows.Next() {
		var info common.BannerInfo
		err = rows.Scan(&info.Id, &info.Img, &info.Dst, &info.Online,
			&info.Priority, &info.Title, &info.Expire, &info.Dbg)
		if err != nil {
			log.Printf("scan rows failed: %v", err)
			return infos
		}
		infos = append(infos, &info)
		log.Printf("id:%d img:%s dst:%s Online:%d priority:%d\n", info.Id,
			info.Img, info.Dst, info.Online, info.Priority)
	}
	return infos
}

func (s *server) FetchBanners(ctx context.Context, in *common.CommRequest) (*fetch.BannerReply, error) {
	util.PubRPCRequest(w, "fetch", "FetchBanners")
	log.Printf("request uid:%d, sid:%s seq:%d num:%d", in.Head.Uid,
		in.Head.Sid, in.Seq, in.Num)
	infos := getBanners(db, in.Seq, in.Type, in.Num)
	total := getTotalBanners(db, in.Type)
	util.PubRPCSuccRsp(w, "fetch", "FetchBanners")
	return &fetch.BannerReply{
		Head:  &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid},
		Infos: infos, Total: total}, nil
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
	util.PubRPCRequest(w, "fetch", "FetchWifiPass")
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

	util.PubRPCSuccRsp(w, "fetch", "FetchWifiPass")
	return &fetch.WifiPassReply{
		Head:     &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid},
		Wifipass: wifis}, nil
}

func (s *server) FetchStsCredentials(ctx context.Context, in *common.CommRequest) (*fetch.StsReply, error) {
	util.PubRPCRequest(w, "fetch", "FetchStsCredentials")
	res := aliyun.FetchStsCredentials()
	log.Printf("FetchStsCredentials resp:%s", res)
	if res == "" {
		return &fetch.StsReply{Head: &common.Head{Retcode: 1}},
			errors.New("fetch sts failed")
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
	util.PubRPCSuccRsp(w, "fetch", "FetchStsCredential")
	return &fetch.StsReply{
		Head:       &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid},
		Credential: &cred}, nil
}

func (s *server) FetchZipcode(ctx context.Context, in *fetch.ZipcodeRequest) (*fetch.ZipcodeReply, error) {
	util.PubRPCRequest(w, "fetch", "FetchZipcode")
	log.Printf("FetchZipcode request uid:%d, type:%d code:%d",
		in.Head.Uid, in.Type, in.Code)
	query := "SELECT code, address FROM zipcode WHERE"
	switch in.Type {
	default:
		query += " code % 10000 = 0"
	case townType:
		query += " code % 100 = 0 AND code % 10000 != 0 AND FLOOR(code/10000) = " +
			strconv.Itoa(int(in.Code/10000))
	case districtType:
		query += " code % 100 != 0 AND FLOOR(code/100) = " +
			strconv.Itoa(int(in.Code/100))
	}
	log.Printf("query:%s", query)
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("FetchZipcode query failed:%v", err)
		return &fetch.ZipcodeReply{Head: &common.Head{Retcode: 1}}, nil
	}

	defer rows.Close()
	var infos []*fetch.ZipcodeInfo
	for rows.Next() {
		var info fetch.ZipcodeInfo
		err := rows.Scan(&info.Code, &info.Address)
		if err != nil {
			log.Printf("FetchZipcode scan failed:%v", err)
			continue
		}
		infos = append(infos, &info)
	}

	util.PubRPCSuccRsp(w, "fetch", "FetchZipcode")
	return &fetch.ZipcodeReply{
		Head:    &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid},
		Zipcode: infos}, nil
}

func (s *server) FetchAddress(ctx context.Context, in *common.CommRequest) (*fetch.AddressReply, error) {
	util.PubRPCRequest(w, "fetch", "FetchAddress")
	log.Printf("FetchAddress request uid:%d", in.Head.Uid)
	rows, err := db.Query("SELECT aid, consignee, phone, province, city, district, addr, detail, flag, zip FROM address WHERE deleted = 0 AND uid = ?",
		in.Head.Uid)
	if err != nil {
		log.Printf("FetchAddress query failed uid:%d, %v", in.Head.Uid, err)
		return &fetch.AddressReply{Head: &common.Head{Retcode: 1}}, err
	}

	defer rows.Close()
	var infos []*common.AddressInfo
	for rows.Next() {
		var info common.AddressInfo
		err := rows.Scan(&info.Aid, &info.User, &info.Mobile, &info.Province,
			&info.City, &info.Zone, &info.Addr, &info.Detail, &info.Def,
			&info.Zip)
		if err != nil {
			log.Printf("FetchAddress scan failed:%v", err)
			continue
		}
		infos = append(infos, &info)
	}

	util.PubRPCSuccRsp(w, "fetch", "FetchAddress")
	return &fetch.AddressReply{
		Head:    &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid},
		Address: infos}, nil
}

func getFlashAd(db *sql.DB, flag bool) common.BannerInfo {
	var info common.BannerInfo
	query := "SELECT img, dst, title, etime FROM banner WHERE deleted = 0 AND type = 1 "
	if flag {
		query += " AND (online = 1 OR dbg = 1) "
	} else {
		query += " AND online = 1 "
	}
	query += " ORDER BY priority DESC LIMIT 1"
	err := db.QueryRow(query).Scan(&info.Img, &info.Dst, &info.Title, &info.Expire)
	if err != nil {
		log.Printf("FetchFlashAd query failed %v", err)
	}
	return info
}

func isAdBan(db *sql.DB, term, version int64) bool {
	var num int
	err := db.QueryRow("SELECT COUNT(id) FROM ad_ban WHERE deleted = 0 AND term = ? AND version = ?",
		term, version).
		Scan(&num)
	if err != nil {
		log.Printf("isAdBan query failed:%v", err)
	}
	if num > 0 {
		return true
	}
	return false
}

func (s *server) FetchFlashAd(ctx context.Context, in *fetch.AdRequest) (*fetch.AdReply, error) {
	util.PubRPCRequest(w, "fetch", "FetchFlashAd")
	log.Printf("FetchFlashAd request uid:%d term:%d versoin:%d", in.Head.Uid,
		in.Term, in.Version)
	if !util.IsWhiteUser(db, in.Head.Uid, util.FlashAdWhiteType) &&
		isAdBan(db, in.Term, in.Version) {
		log.Printf("FetchFlashAd ban uid:%d term:%d version:%d", in.Head.Uid,
			in.Term, in.Version)
		return &fetch.AdReply{
			Head: &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid}}, nil
	}
	flag := util.IsWhiteUser(db, in.Head.Uid, util.FlashAdDbgType)
	info := getFlashAd(db, flag)
	util.PubRPCSuccRsp(w, "fetch", "FetchFlashAd")
	return &fetch.AdReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid},
		Info: &info}, nil
}

func (s *server) FetchConf(ctx context.Context, in *common.CommRequest) (*fetch.ConfReply, error) {
	util.PubRPCRequest(w, "fetch", "FetchConf")
	log.Printf("FetchConf request uid:%d", in.Head.Uid)
	var infos []*common.KvInfo
	rows, err := db.Query("SELECT name, val FROM kv_config WHERE deleted = 0")
	if err != nil {
		log.Printf("FetchConf query failed uid:%d, %v", in.Head.Uid, err)
		return &fetch.ConfReply{Head: &common.Head{Retcode: 1}}, err
	}
	defer rows.Close()

	for rows.Next() {
		var info common.KvInfo
		err := rows.Scan(&info.Key, &info.Val)
		if err != nil {
			log.Printf("FetchConf scan failed:%v", err)
			continue
		}
		infos = append(infos, &info)
	}

	util.PubRPCSuccRsp(w, "fetch", "FetchConf")
	return &fetch.ConfReply{
		Head:  &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid},
		Infos: infos}, nil
}

func (s *server) FetchKvConf(ctx context.Context, in *fetch.KvRequest) (*fetch.KvReply, error) {
	util.PubRPCRequest(w, "fetch", "FetchKvConf")
	log.Printf("FetchKvConf request uid:%d", in.Head.Uid)
	var val string
	err := db.QueryRow("SELECT val FROM kv_config WHERE deleted = 0 AND name = ?", in.Key).
		Scan(&val)
	if err != nil {
		log.Printf("FetchKvConf query failed uid:%d, %v", in.Head.Uid, err)
		return &fetch.KvReply{Head: &common.Head{Retcode: 1}}, err
	}
	util.PubRPCSuccRsp(w, "fetch", "FetchKvConf")
	return &fetch.KvReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid},
		Val:  val}, nil
}

func (s *server) FetchActivity(ctx context.Context, in *common.CommRequest) (*fetch.ActivityReply, error) {
	util.PubRPCRequest(w, "fetch", "FetchActivity")
	log.Printf("FetchActivity request uid:%d", in.Head.Uid)
	var info common.BannerInfo
	query := "SELECT title, dst FROM banner WHERE deleted = 0 AND type = 2 "
	if util.IsWhiteUser(db, in.Head.Uid, util.ActivityWhiteType) {
		query += " AND (online = 1 OR dbg = 1) "
	} else {
		query += " AND online = 1 "
	}
	query += " ORDER BY priority DESC LIMIT 1"
	err := db.QueryRow(query).Scan(&info.Title, &info.Dst)
	if err != nil {
		log.Printf("FetchActivity query failed uid:%d, %v", in.Head.Uid, err)
		return &fetch.ActivityReply{Head: &common.Head{Retcode: 1}}, err
	}
	log.Printf("title:%s dst:%s", info.Title, info.Dst)
	util.PubRPCSuccRsp(w, "fetch", "FetchActivity")
	return &fetch.ActivityReply{
		Head:     &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid},
		Activity: &info}, nil
}

func getGoodsIntro(db *sql.DB, gid int64) fetch.GoodsIntro {
	var info fetch.GoodsIntro
	rows, err := db.Query("SELECT g.price, i.image FROM goods g, goods_image i WHERE g.gid = i.gid AND i.type = 1 AND i.deleted = 0 AND g.gid = ?", gid)
	if err != nil {
		log.Printf("getGoodsIntro failed, gid:%d %v", gid, err)
		return info
	}
	defer rows.Close()

	flag := false
	var images []string
	for rows.Next() {
		var price int
		var image string
		err := rows.Scan(&price, &image)
		if err != nil {
			log.Printf("getGoodsIntro scan failed:%v", err)
			continue
		}
		if price > highPrice {
			flag = true
		}
		images = append(images, image)
	}
	if flag {
		info.Text = priceTips
	}
	info.Images = images
	return info
}

func (s *server) FetchGoodsIntro(ctx context.Context, in *common.CommRequest) (*fetch.GoodsIntroReply, error) {
	util.PubRPCRequest(w, "fetch", "FetchGoodsIntro")
	log.Printf("FetchGoodsIntro request uid:%d gid:%d", in.Head.Uid, in.Id)
	info := getGoodsIntro(db, in.Id)
	util.PubRPCSuccRsp(w, "fetch", "FetchGoodsIntro")
	return &fetch.GoodsIntroReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid},
		Info: &info}, nil
}

func getPurchaseRecords(db *sql.DB, sid, seq, num int64) []*fetch.PurchaseRecord {
	var infos []*fetch.PurchaseRecord
	query := `SELECT hid, h.uid, nickname, headurl, num, ctime FROM purchase_history h, user u
		WHERE h.uid = u.uid AND h.ack_flag = 1 AND h.sid = `
	query += strconv.Itoa(int(sid))
	if seq > 0 {
		query += fmt.Sprintf(" AND hid < %d ", seq)
	}
	query += fmt.Sprintf(" ORDER BY hid DESC LIMIT %d", num)
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("getPurchaseRecords query failed:%v", err)
		return infos
	}
	defer rows.Close()

	for rows.Next() {
		var info fetch.PurchaseRecord
		err = rows.Scan(&info.Rid, &info.Uid, &info.Nickname, &info.Head,
			&info.Num, &info.Ctime)
		if err != nil {
			log.Printf("getPurchaseRecords scan failed:%v", err)
			continue
		}
		info.Seq = info.Rid
		infos = append(infos, &info)
	}
	return infos
}

func (s *server) FetchPurchaseRecord(ctx context.Context, in *common.CommRequest) (*fetch.PurchaseRecordReply, error) {
	util.PubRPCRequest(w, "fetch", "FetchPurchaseRecord")
	log.Printf("FetchPurchaseRecord request uid:%d gid:%d", in.Head.Uid, in.Id)
	infos := getPurchaseRecords(db, in.Id, in.Seq, in.Num)
	util.PubRPCSuccRsp(w, "fetch", "FetchPurchaseRecord")
	return &fetch.PurchaseRecordReply{
		Head:    &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid},
		Records: infos}, nil
}

func getBetHistory(db *sql.DB, gid, seq, num int64) []*common.BidInfo {
	var infos []*common.BidInfo
	query := `SELECT sid, num, UNIX_TIMESTAMP(atime), win_uid, win_code, u.nickname, u.headurl 
		FROM sales s, user u WHERE s.win_uid = u.uid AND s.status >= 3 AND gid = `
	query += strconv.Itoa(int(gid))
	if seq > 0 {
		query += fmt.Sprintf(" AND sid < %d ", seq)
	}
	query += " ORDER BY sid DESC "
	if num > 0 {
		query += fmt.Sprintf(" LIMIT %d", num)
	}
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("getBetHistory query failed:%v", err)
		return infos
	}
	defer rows.Close()

	for rows.Next() {
		var info common.BidInfo
		var award common.AwardInfo
		err := rows.Scan(&info.Bid, &info.Period, &info.End,
			&award.Uid, &award.Awardcode, &award.Nickname,
			&award.Head)
		if err != nil {
			log.Printf("getBetHistory scan failed:%v", err)
			continue
		}
		info.Seq = info.Bid
		info.End *= 1000
		info.Award = &award
		infos = append(infos, &info)
	}
	return infos
}

func (s *server) FetchBetHistory(ctx context.Context, in *common.CommRequest) (*fetch.BetHistoryReply, error) {
	util.PubRPCRequest(w, "fetch", "FetchBetHistory")
	log.Printf("FetchBetHistory request uid:%d gid:%d", in.Head.Uid, in.Id)
	infos := getBetHistory(db, in.Id, in.Seq, in.Num)
	util.PubRPCSuccRsp(w, "fetch", "FetchBetHistory")
	return &fetch.BetHistoryReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid},
		Bets: infos}, nil
}

func getShareImage(db *sql.DB, sid int64) string {
	query := fmt.Sprintf("SELECT url FROM share_image WHERE deleted = 0 AND sid = %d LIMIT 1", sid)
	var url string
	err := db.QueryRow(query).Scan(&url)
	if err != nil {
		log.Printf("getShareImage failed:%v", err)
		return ""
	}
	return url
}

func getShareInfo(db *sql.DB, uid, stype, id, num, seq int64) []*fetch.ShareInfo {
	query := `SELECT hid, s.sid, s.uid, gid, title, nickname, headurl, UNIX_TIMESTAMP(s.ctime),
		s.image_num, LEFT(s.content, 100) FROM share_history s, user u WHERE s.uid = u.uid`
	switch stype {
	default:
		query += fmt.Sprintf(" AND s.uid = %d ", id)
	case util.GidShareType:
		query += fmt.Sprintf(" AND s.gid = %d ", id)
	case util.ListShareType:
		query += " AND s.top_flag = 0 "
	case util.TopShareType:
		query += " AND s.top_flag = 1 "
	}
	if seq > 0 && stype != util.TopShareType {
		query += fmt.Sprintf(" AND hid < %d ", seq)
	}
	query += " AND s.reviewed = 1 AND s.deleted = 0 ORDER by hid DESC "
	if num > 0 {
		query += fmt.Sprintf(" LIMIT %d ", num)
	}
	log.Printf("FetchShare query:%s", query)
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("FetchShare query failed:%v", err)
		return nil
	}
	defer rows.Close()

	var infos []*fetch.ShareInfo
	for rows.Next() {
		var info fetch.ShareInfo
		var imageNum int
		err := rows.Scan(&info.Sid, &info.Bid, &info.Uid, &info.Gid, &info.Title,
			&info.Nickname, &info.Head, &info.Sharetime, &imageNum, &info.Text)
		if err != nil {
			log.Printf("FetchShare scan failed:%v", err)
			continue
		}
		if stype == util.TopShareType {
			info.Seq = info.Sid + 1000000
		} else {
			info.Seq = info.Sid
		}
		if imageNum > 0 {
			info.Image = getShareImage(db, info.Bid)
		}
		infos = append(infos, &info)
	}
	return infos
}

func getMyShare(db *sql.DB, uid, num, seq int64) []*fetch.ShareInfo {
	var infos []*fetch.ShareInfo
	query := "SELECT s.sid, s.num, g.image, g.title, l.share FROM sales s, goods g, logistics l WHERE s.gid = g.gid AND s.sid = l.sid AND l.status >= 6 AND s.win_uid = " +
		strconv.Itoa(int(uid))
	if seq > 0 {
		query += fmt.Sprintf(" AND s.sid = %d", seq)
	}
	query += " ORDER BY s.sid DESC "
	if num > 0 {
		query += fmt.Sprintf(" LIMIT %d", num)
	}

	rows, err := db.Query(query)
	if err != nil {
		log.Printf("FetchShare query failed:%v", err)
		return nil
	}
	defer rows.Close()

	for rows.Next() {
		var info fetch.ShareInfo
		err := rows.Scan(&info.Bid, &info.Period, &info.Image, &info.Title, &info.Share)
		if err != nil {
			log.Printf("FetchShare scan failed:%v", err)
			continue
		}
		info.Seq = info.Bid
		infos = append(infos, &info)
	}
	return infos
}

func (s *server) FetchShare(ctx context.Context, in *fetch.ShareRequest) (*fetch.ShareReply, error) {
	util.PubRPCRequest(w, "fetch", "FetchShare")
	log.Printf("FetchShare uid:%d type:%d seq:%d num:%d id:%d", in.Head.Uid,
		in.Type, in.Seq, in.Num, in.Id)
	var reddot int64
	if in.Type != util.UidShareType && util.HasReddot(db, in.Head.Uid) {
		reddot = 1
	}
	var infos []*fetch.ShareInfo
	switch in.Type {
	case util.GidShareType:
		infos = getShareInfo(db, in.Head.Uid, util.GidShareType, in.Id, in.Num,
			in.Seq)
	case util.ListShareType:
		top := getShareInfo(db, in.Head.Uid, util.TopShareType, 0, in.Num, in.Seq)
		list := getShareInfo(db, in.Head.Uid, util.ListShareType, 0, in.Num, in.Seq)
		infos = append(top, list...)
	case util.UidShareType:
		infos = getMyShare(db, in.Head.Uid, in.Num, in.Seq)
	}

	util.PubRPCSuccRsp(w, "fetch", "FetchShare")
	return &fetch.ShareReply{
		Head:   &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid},
		Shares: infos, Reddot: reddot}, nil
}

func getShareImages(db *sql.DB, sid int64) []string {
	var images []string
	rows, err := db.Query("SELECT url FROM share_image WHERE review = 1 AND deleted = 0 AND sid = ?", sid)
	if err != nil {
		log.Printf("getShareImages failed:%v", err)
		return images
	}
	defer rows.Close()

	for rows.Next() {
		var url string
		err = rows.Scan(&url)
		if err != nil {
			log.Printf("getShareImages scan failed:%v", err)
			continue
		}
		images = append(images, url)
	}
	return images
}

func (s *server) FetchShareDetail(ctx context.Context, in *common.CommRequest) (*fetch.ShareDetailReply, error) {
	util.PubRPCRequest(w, "fetch", "FetchShareDetail")
	log.Printf("FetchShareDetail request uid:%d sid:%d", in.Head.Uid, in.Id)
	var share fetch.ShareInfo
	var imageNum int
	err := db.QueryRow("SELECT hid, h.uid, h.title, content, image_num, UNIX_TIMESTAMP(h.ctime), nickname FROM share_history h, user u WHERE h.uid = u.uid AND h.deleted = 0 AND h.reviewed = 1 AND h.sid = ?", in.Id).
		Scan(&share.Sid, &share.Uid, &share.Title, &share.Text, &imageNum,
			&share.Sharetime, &share.Nickname)
	if err != nil {
		log.Printf("FetchShareDetail query share failed sid:%d %v", in.Id, err)
		return &fetch.ShareDetailReply{Head: &common.Head{Retcode: 1}}, err
	}
	share.Images = getShareImages(db, in.Id)
	var bet common.BidInfo
	var award common.AwardInfo
	err = db.QueryRow("SELECT num, title, win_code, UNIX_TIMESTAMP(atime) FROM sales s, goods g WHERE s.gid = g.gid AND s.sid = ?", in.Id).
		Scan(&bet.Period, &bet.Title, &award.Awardcode, &bet.End)
	if err != nil {
		log.Printf("FetchShareDetail query bet failed sid:%d %v", in.Id, err)
		return &fetch.ShareDetailReply{Head: &common.Head{Retcode: 1}}, err
	}
	bet.Bid = in.Id
	award.Num = util.GetSalesCount(db, in.Id, share.Uid)
	bet.Award = &award

	util.PubRPCSuccRsp(w, "fetch", "FetchShareDetail")
	return &fetch.ShareDetailReply{
		Head:  &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid},
		Share: &share, Bet: &bet}, nil
}

func getUserBanner(db *sql.DB) string {
	var banner string
	err := db.QueryRow("SELECT img FROM banner WHERE type = 3 AND deleted = 0 ORDER BY id DESC LIMIT 1").
		Scan(&banner)
	if err != nil {
		log.Printf("getUserBanner query failed:%v", err)
	}
	return banner
}

func hasAward(db *sql.DB, uid int64) bool {
	var cnt int
	err := db.QueryRow("SELECT COUNT(sid) FROM sales WHERE status = 3 AND win_uid = ?", uid).
		Scan(&cnt)
	if err != nil {
		log.Printf("hasAward query failed uid:%d %v", uid, err)
	}
	if cnt > 0 {
		return true
	}

	err = db.QueryRow("SELECT COUNT(lid) FROM logistics WHERE status < 6 AND uid = ?", uid).
		Scan(&cnt)
	if err != nil {
		log.Printf("hasAward query failed uid:%d %v", uid, err)
	}
	if cnt > 0 {
		return true
	}
	return false
}

func needShare(db *sql.DB, uid int64) bool {
	var cnt int
	err := db.QueryRow("SELECT COUNT(lid) FROM logistics WHERE status >= 6 AND share = 0 AND uid = ?", uid).
		Scan(&cnt)
	if err != nil {
		log.Printf("needShare query failed uid:%d %v", uid, err)
	}
	if cnt > 0 {
		return true
	}
	return false
}

func getUserInfo(db *sql.DB, uid int64) fetch.UserInfo {
	var info fetch.UserInfo
	err := db.QueryRow("SELECT nickname, headurl, phone, balance FROM user WHERE uid = ?", uid).
		Scan(&info.Nickname, &info.Head, &info.Phone, &info.Coin)
	if err != nil {
		log.Printf("getUserInfo query failed uid: %d %v", uid, err)
		return info
	}
	info.Coin /= 100
	info.Award = hasAward(db, uid)
	info.Share = needShare(db, uid)
	return info
}

func (s *server) FetchUserInfo(ctx context.Context, in *common.CommRequest) (*fetch.UserInfoReply, error) {
	util.PubRPCRequest(w, "fetch", "FetchUserInfo")
	log.Printf("FetchUserInfo uid:%d", in.Head.Uid)
	info := getUserInfo(db, in.Head.Uid)
	banner := getUserBanner(db)
	util.PubRPCSuccRsp(w, "fetch", "FetchUserInfo")
	return &fetch.UserInfoReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid},
		Info: &info, Banner: banner}, nil
}

func getWinnerInfo(db *sql.DB, sid int64) common.AwardInfo {
	var award common.AwardInfo
	err := db.QueryRow("SELECT uid, nickname, win_code FROM sales s, user u WHERE s.win_uid = u.uid AND s.sid = ?", sid).
		Scan(&award.Uid, &award.Nickname, &award.Awardcode)
	if err != nil {
		log.Printf("getWinnerInfo failed sid:%d %v", sid, err)
		return award
	}
	award.Num = util.GetSalesCount(db, sid, award.Uid)
	return award
}

func getUserBets(db *sql.DB, uid, seq, num int64) []*common.BidInfo {
	var bets []*common.BidInfo
	if seq > 0 {
		bets = getUserSales(db, uid, seq, num, true)
	} else {
		t := getUserSales(db, uid, seq, num, false)
		q := getUserSales(db, uid, seq, num, true)
		bets = append(t, q...)
	}
	return bets
}

func getUserSales(db *sql.DB, uid, seq, num int64, flag bool) []*common.BidInfo {
	var bets []*common.BidInfo
	query := `SELECT s.sid, num, title, total, remain, status, image, UNIX_TIMESTAMP(s.etime),
		UNIX_TIMESTAMP(s.atime) FROM sales s, goods g, (SELECT distinct sid FROM purchase_history
		WHERE uid = `
	query += strconv.Itoa(int(uid))
	if seq > 0 {
		query += fmt.Sprintf(" AND sid < %d ", seq)
	}
	if flag {
		query += fmt.Sprintf(" ORDER BY hid DESC LIMIT %d", num)
	}
	query += ") as tl WHERE s.sid = tl.sid AND s.gid = g.gid "
	if flag {
		query += " AND s.status NOT IN (1, 2)"
	} else {
		query += " AND s.status IN (1, 2) ORDER BY s.status DESC"
	}
	log.Printf("getUserSales query:%s", query)
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("getUserSales query failed:%v", err)
		return bets
	}
	defer rows.Close()

	for rows.Next() {
		var info common.BidInfo
		var rest, end int64
		err = rows.Scan(&info.Bid, &info.Period, &info.Title, &info.Total,
			&info.Remain, &info.Status, &info.Image, &rest, &end)
		if err != nil {
			log.Printf("getUserSales scan failed:%v", err)
			continue
		}
		if info.Status == 2 {
			info.Rest = util.GetRemainSeconds(rest)
		} else if info.Status >= 3 {
			info.End = end * 1000
			award := getWinnerInfo(db, info.Bid)
			info.Award = &award
		}
		info.Codes = util.GetSalesCodes(db, info.Bid, uid)
		bets = append(bets, &info)
	}
	return bets
}

func (s *server) FetchUserBet(ctx context.Context, in *common.CommRequest) (*fetch.UserBetReply, error) {
	util.PubRPCRequest(w, "fetch", "FetchUserBet")
	log.Printf("FetchUserBet uid:%d seq:%d num:%d type:%d", in.Head.Uid, in.Seq,
		in.Num, in.Type)
	var infos []*common.BidInfo
	if in.Type == util.UserAwardType {
		infos = getUserAward(db, in.Head.Uid, in.Seq, in.Num)
	} else {
		infos = getUserBets(db, in.Head.Uid, in.Seq, in.Num)
	}
	util.PubRPCSuccRsp(w, "fetch", "FetchUserBet")
	return &fetch.UserBetReply{
		Head:  &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid},
		Infos: infos}, nil
}

func getUserAward(db *sql.DB, uid, seq, num int64) []*common.BidInfo {
	var infos []*common.BidInfo
	query := `SELECT sid, status, num, title, total, remain, image, UNIX_TIMESTAMP(s.atime),
		win_code, UNIX_TIMESTAMP(s.etime), g.type FROM sales s, goods g 
		WHERE s.gid = g.gid AND s.status >= 3 AND s.win_uid = `
	query += strconv.Itoa(int(uid))
	if seq > 0 {
		query += fmt.Sprintf(" AND UNIX_TIMESTAMP(s.etime) < %d", seq)
	}
	query += fmt.Sprintf(" ORDER BY s.etime DESC LIMIT %d", num)
	log.Printf("getUserAward query:%s", query)
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("getUserAward query failed:%v", err)
		return infos
	}

	defer rows.Close()
	for rows.Next() {
		var info common.BidInfo
		var award common.AwardInfo
		err = rows.Scan(&info.Bid, &info.Status, &info.Period, &info.Title,
			&info.Total, &info.Remain, &info.Image, &info.End,
			&award.Awardcode, &info.Seq, &info.Gtype)
		if err != nil {
			log.Printf("getUserAward scan failed:%v", err)
			continue
		}
		info.Codes = util.GetSalesCodes(db, info.Bid, uid)
		info.Award = &award
		infos = append(infos, &info)
	}
	return infos
}

func getAdBan(db *sql.DB) []*common.AdBan {
	var infos []*common.AdBan
	rows, err := db.Query("SELECT id, term, version FROM ad_ban WHERE deleted = 0 ORDER BY id DESC")
	if err != nil {
		log.Printf("getAdBan failed:%v", err)
		return infos
	}

	defer rows.Close()
	for rows.Next() {
		var info common.AdBan
		err = rows.Scan(&info.Id, &info.Term, &info.Version)
		if err != nil {
			log.Printf("getAdBan scan failed:%v", err)
			continue
		}
		infos = append(infos, &info)
	}
	return infos
}

func (s *server) FetchAdBan(ctx context.Context, in *common.CommRequest) (*fetch.AdBanReply, error) {
	util.PubRPCRequest(w, "fetch", "FetchAdBan")
	log.Printf("FetchUserBet uid:%d", in.Head.Uid)
	infos := getAdBan(db)
	util.PubRPCSuccRsp(w, "fetch", "FetchAdBan")
	return &fetch.AdBanReply{
		Head:  &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid},
		Infos: infos}, nil
}

func getWhiteTotal(db *sql.DB, wtype int64) int64 {
	var total int64
	err := db.QueryRow("SELECT COUNT(id) FROM white_list WHERE deleted = 0 AND type = ?", wtype).
		Scan(&total)
	if err != nil {
		log.Printf("getWhiteTotal query failed:%v", err)
	}
	return total
}

func getWhiteList(db *sql.DB, seq, num int64) []*fetch.WhiteUser {
	var infos []*fetch.WhiteUser
	rows, err := db.Query("SELECT u.uid, u.username, u.phone FROM white_list w, user u WHERE w.uid = u.uid AND w.deleted = 0 AND u.deleted = 0 AND w.type = 0 ORDER BY id DESC LIMIT ?, ?", seq, num)
	if err != nil {
		log.Printf("getWhiteList query failed:%v", err)
		return infos
	}

	defer rows.Close()
	for rows.Next() {
		var info fetch.WhiteUser
		var phone string
		err = rows.Scan(&info.Uid, &info.Phone, &phone)
		if err != nil {
			log.Printf("getWhiteList scan failed:%v", err)
			continue
		}
		if phone != "" {
			info.Phone = phone
		}
		infos = append(infos, &info)
	}
	return infos
}

func (s *server) FetchWhiteList(ctx context.Context, in *common.CommRequest) (*fetch.WhiteReply, error) {
	util.PubRPCRequest(w, "fetch", "FetchWhiteList")
	log.Printf("FetchWhiteList uid:%d", in.Head.Uid)
	infos := getWhiteList(db, in.Seq, in.Num)
	total := getWhiteTotal(db, in.Type)
	util.PubRPCSuccRsp(w, "fetch", "FetchWhiteList")
	return &fetch.WhiteReply{
		Head:  &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid},
		Infos: infos, Total: total}, nil
}

func getFeedback(db *sql.DB, seq, num int64) []*fetch.FeedbackInfo {
	var infos []*fetch.FeedbackInfo
	rows, err := db.Query("SELECT u.uid, u.username, u.phone, f.content, f.ctime FROM feedback f, user u WHERE f.uid = u.uid ORDER BY f.id DESC LIMIT ?, ?", seq, num)
	if err != nil {
		log.Printf("getFeedback query failed:%v", err)
		return infos
	}

	defer rows.Close()
	for rows.Next() {
		var info fetch.FeedbackInfo
		var phone string
		err := rows.Scan(&info.Uid, &info.Phone, &phone, &info.Content, &info.Ctime)
		if err != nil {
			log.Printf("getFeedback scan failed:%v", err)
			continue
		}
		if phone != "" {
			info.Phone = phone
		}
		infos = append(infos, &info)
	}
	return infos
}

func getFeedbackTotal(db *sql.DB) int64 {
	var total int64
	err := db.QueryRow("SELECT COUNT(id) FROM feedback").Scan(&total)
	if err != nil {
		log.Printf("getFeedbackTotal scan failed:%v", err)
	}
	return total
}

func (s *server) FetchFeedback(ctx context.Context, in *common.CommRequest) (*fetch.FeedbackReply, error) {
	util.PubRPCRequest(w, "fetch", "FetchFeedback")
	log.Printf("FetchFeedback uid:%d seq:%d num:%d", in.Head.Uid, in.Seq, in.Num)
	infos := getFeedback(db, in.Seq, in.Num)
	total := getFeedbackTotal(db)
	util.PubRPCSuccRsp(w, "fetch", "FetchFeedback")
	return &fetch.FeedbackReply{
		Head:  &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid},
		Infos: infos, Total: total}, nil
}

func getMenus(db *sql.DB, term, version int64) []*fetch.MenuInfo {
	var infos []*fetch.MenuInfo
	query := "SELECT type, ctype, title, dst FROM menu WHERE deleted = 0"
	if term == 0 && version <= 6 {
		query += " AND ctype < 4 "
	}
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("getMenus failed:%v", err)
		return infos
	}

	defer rows.Close()
	for rows.Next() {
		var info fetch.MenuInfo
		err := rows.Scan(&info.Type, &info.Ctype, &info.Title, &info.Dst)
		if err != nil {
			log.Printf("getMenus scan failed:%v", err)
		}
		infos = append(infos, &info)
	}
	return infos
}

func (s *server) FetchMenu(ctx context.Context, in *common.CommRequest) (*fetch.MenuReply, error) {
	util.PubRPCRequest(w, "fetch", "FetchMenu")
	log.Printf("FetchMenu uid:%d", in.Head.Uid)
	infos := getMenus(db, in.Head.Term, in.Head.Version)
	util.PubRPCSuccRsp(w, "fetch", "FetchMenu")
	return &fetch.MenuReply{
		Head:  &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid},
		Infos: infos}, nil
}

func getLogisticsStatus(db *sql.DB, sid int64) int64 {
	var status int64
	err := db.QueryRow("SELECT status FROM logistics WHERE sid = ?", sid).
		Scan(&status)
	if err != nil {
		log.Printf("getLogisticsStatus query failed, sid:%d %v", sid, err)
	}
	return status
}

func getWinInfo(db *sql.DB, sid int64) common.BidInfo {
	var info common.BidInfo
	var award common.AwardInfo
	err := db.QueryRow("SELECT g.title, s.num, s.total, s.win_code, s.win_uid, UNIX_TIMESTAMP(s.atime), s.status, g.image, g.type FROM sales s, goods g WHERE s.gid = g.gid AND s.sid = ?", sid).
		Scan(&info.Title, &info.Period, &info.Total, &award.Awardcode, &award.Uid,
			&info.End, &info.Status, &info.Image, &info.Gtype)
	if err != nil {
		log.Printf("getWinInfo failed sid:%d %v", sid, err)
		return info
	}
	info.Bid = sid
	award.Num = util.GetSalesCount(db, sid, award.Uid)
	info.Award = &award
	if info.Status >= 4 {
		info.Status = getLogisticsStatus(db, sid)
	}
	return info
}

func getUserAddress(db *sql.DB, uid int64) fetch.AddressInfo {
	var info fetch.AddressInfo
	err := db.QueryRow("SELECT consignee, a.phone, detail,aid FROM address a, user_info u  WHERE u.default_address = a.aid AND u.uid = ?", uid).
		Scan(&info.Name, &info.Phone, &info.Detail, &info.Id)
	if err != nil {
		log.Printf("getUserAddress query failed uid:%d %v", uid, err)
	}
	return info
}

func getAddress(db *sql.DB, aid int64) fetch.AddressInfo {
	var info fetch.AddressInfo
	err := db.QueryRow("SELECT consignee, phone, detail,aid FROM address WHERE aid = ?", aid).
		Scan(&info.Name, &info.Phone, &info.Detail, &info.Id)
	if err != nil {
		log.Printf("getAddress query failed uid:%d %v", aid, err)
	}
	info.Id = aid
	return info
}

func (s *server) FetchWinStatus(ctx context.Context, in *common.CommRequest) (*fetch.WinStatusReply, error) {
	util.PubRPCRequest(w, "fetch", "FetchWinStatus")
	log.Printf("FetchWinStatus uid:%d sid:%d", in.Head.Uid, in.Id)
	bet := getWinInfo(db, in.Id)
	var address fetch.AddressInfo
	var info fetch.WinInfo
	if bet.Status <= 3 && bet.Award.Uid > 0 {
		address = getUserAddress(db, bet.Award.Uid)
	}
	if bet.Status >= 4 {
		query := `SELECT aid, express, track_num, UNIX_TIMESTAMP(ctime), UNIX_TIMESTAMP(etime),
			UNIX_TIMESTAMP(rtime), share, account, award_account FROM logistics WHERE sid = `
		query += strconv.Itoa(int(in.Id))
		var eid int64
		err := db.QueryRow(query).Scan(&address.Id, &eid, &info.Num,
			&info.Addresstime, &info.Shiptime, &info.Confirmtime,
			&info.Share, &info.Account, &info.Award)
		if err != nil {
			log.Printf("FetchWinStatus query failed:%v", err)
		}
		info.Vendor = expressList[eid]
		address = getAddress(db, address.Id)
	}

	util.PubRPCSuccRsp(w, "fetch", "FetchWinStatus")
	return &fetch.WinStatusReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid},
		Bet:  &bet, Address: &address, Info: &info}, nil
}

func genApkDownURL(channel, version string) string {
	file := fmt.Sprintf("wireless.%s.%s.apk", channel, version)
	return aliyun.GenOssFileURL(file)
}

func (s *server) FetchLatestVersion(ctx context.Context, in *fetch.VersionRequest) (*fetch.VersionReply, error) {
	util.PubRPCRequest(w, "fetch", "FetchLatestVersion")
	log.Printf("FetchLatestVersion request uid:%d term:%d versoin:%d channel:%s",
		in.Head.Uid, in.Head.Term, in.Head.Version, in.Channel)
	var version int64
	var vname, downurl string
	err := db.QueryRow("SELECT version, vname, downurl FROM app_channel WHERE channel = ?",
		in.Channel).Scan(&version, &vname, &downurl)
	if err != nil {
		log.Printf("FetchLatestVersion failed:%v", err)
		return &fetch.VersionReply{
			Head: &common.Head{Retcode: common.ErrCode_NO_NEW_VERSION}}, nil
	}
	if version <= in.Head.Version {
		return &fetch.VersionReply{
			Head: &common.Head{Retcode: common.ErrCode_NO_NEW_VERSION}}, nil
	}
	util.PubRPCSuccRsp(w, "fetch", "FetchLatestVersion")
	return &fetch.VersionReply{
		Head:    &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid},
		Version: vname, Downurl: downurl}, nil
}

func (s *server) FetchPortal(ctx context.Context, in *common.CommRequest) (*fetch.PortalReply, error) {
	util.PubRPCRequest(w, "fetch", "FetchPortal")
	log.Printf("FetchPortal request uid:%d sid:%s",
		in.Head.Uid, in.Head.Sid)
	dir, err := util.GetPortalDir(db, util.LoginType)
	if err != nil {
		log.Printf("getPortalDir failed:%v", err)
		return &fetch.PortalReply{
			Head: &common.Head{
				Retcode: common.ErrCode_NOT_EXIST, Uid: in.Head.Uid,
				Sid: in.Head.Sid}}, nil
	}
	log.Printf("FetchPortal dir:%s", dir)
	util.PubRPCSuccRsp(w, "fetch", "FetchPortal")
	return &fetch.PortalReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid},
		Dir:  dir}, nil
}

func getTotalPortalDir(db *sql.DB, ptype int64) int64 {
	var num int64
	err := db.QueryRow("SELECT COUNT(id) FROM portal_page WHERE type = ?", ptype).Scan(&num)
	if err != nil {
		log.Printf("getTotalPortalDir failed:%v", err)
	}
	return num
}

func getPortalDirInfos(db *sql.DB, seq, num, ptype int64) []*common.PortalDirInfo {
	var infos []*common.PortalDirInfo
	rows, err := db.Query("SELECT id, type, dir, description, online, ctime FROM portal_page WHERE type = ? ORDER BY id DESC LIMIT ?,?", ptype, seq, num)
	if err != nil {
		log.Printf("getPortalDirInfos query failed:%v", err)
		return infos
	}
	defer rows.Close()
	for rows.Next() {
		var info common.PortalDirInfo
		err := rows.Scan(&info.Id, &info.Type, &info.Dir, &info.Description,
			&info.Online, &info.Ctime)
		if err != nil {
			log.Printf("getPortalDirInfos scan failed:%v", err)
			continue
		}
		infos = append(infos, &info)
	}
	return infos
}

func (s *server) FetchPortalDir(ctx context.Context, in *common.CommRequest) (*fetch.PortalDirReply, error) {
	util.PubRPCRequest(w, "fetch", "FetchPortalDir")
	log.Printf("FetchPortalDir seq:%d num:%d type:%d uid:%d",
		in.Seq, in.Num, in.Type, in.Head.Uid)
	infos := getPortalDirInfos(db, in.Seq, in.Num, in.Type)
	total := getTotalPortalDir(db, in.Type)
	util.PubRPCSuccRsp(w, "fetch", "FetchPortalDir")
	return &fetch.PortalDirReply{
		Head:  &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid},
		Infos: infos, Total: total}, nil
}

func getTotalChannelVersion(db *sql.DB) int64 {
	var total int64
	err := db.QueryRow("SELECT COUNT(id) FROM app_channel").Scan(&total)
	if err != nil {
		log.Printf("getTotalChannelVersion failed:%v", err)
	}
	return total
}

func getChannelVersion(db *sql.DB, seq, num int64) []*common.ChannelVersionInfo {
	var infos []*common.ChannelVersionInfo
	rows, err := db.Query("SELECT id, channel, cname, version, vname, downurl FROM app_channel ORDER BY id LIMIT ?, ?",
		seq, num)
	if err != nil {
		log.Printf("getChannelVersion failed:%v", err)
		return infos
	}

	defer rows.Close()
	for rows.Next() {
		var info common.ChannelVersionInfo
		err := rows.Scan(&info.Id, &info.Channel, &info.Cname, &info.Version, &info.Vname, &info.Downurl)
		if err != nil {
			log.Printf("getChannelVersion scan failed:%v", err)
			continue
		}
		infos = append(infos, &info)
	}
	return infos
}

func (s *server) FetchChannelVersion(ctx context.Context, in *common.CommRequest) (*fetch.ChannelVersionReply, error) {
	util.PubRPCRequest(w, "fetch", "FetchChannelVersion")
	log.Printf("FetchChannelVersion seq:%d num:%d uid:%d",
		in.Seq, in.Num, in.Head.Uid)
	infos := getChannelVersion(db, in.Seq, in.Num)
	total := getTotalChannelVersion(db)
	util.PubRPCSuccRsp(w, "fetch", "FetchChannelVersion")
	return &fetch.ChannelVersionReply{
		Head:  &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid},
		Infos: infos, Total: total}, nil
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

	w = util.NewNsqProducer()
	kv := util.InitRedis()
	go util.ReportHandler(kv, util.FetchServerName, util.FetchServerPort)
	//cli := util.InitEtcdCli()
	//go util.ReportEtcd(cli, util.FetchServerName, util.FetchServerPort)

	s := grpc.NewServer()
	fetch.RegisterFetchServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
