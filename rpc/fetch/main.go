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

func getTotalVideos(db *sql.DB, ctype int64, search string) int64 {
	query := "SELECT COUNT(vid) FROM youku_video WHERE 1 = 1 " + genTypeQuery(ctype)
	var total int64
	if search != "" {
		query += " AND title LIKE '%" + search + "%' "
	}
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
	rows, err := db.Query("SELECT id, longitude, latitude, unit FROM ap_info WHERE deleted = 0 GROUP BY longitude, latitude")
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

func getVideos(db *sql.DB, seq, num, ctype int64, search string) []*fetch.VideoInfo {
	var infos []*fetch.VideoInfo
	query := "SELECT vid, img, title, dst, ctime, source, duration FROM youku_video WHERE 1 = 1 " +
		genTypeQuery(ctype)
	if search != "" {
		query += " AND title LIKE '%" + search + "%' "
	}
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
	}
	return infos
}

func (s *server) FetchVideos(ctx context.Context, in *common.CommRequest) (*fetch.VideoReply, error) {
	util.PubRPCRequest(w, "fetch", "FetchVideos")
	log.Printf("request uid:%d, sid:%s seq:%d num:%d search:%s",
		in.Head.Uid, in.Head.Sid,
		in.Seq, in.Num, in.Search)
	infos := getVideos(db, in.Seq, in.Num, in.Type, in.Search)
	total := getTotalVideos(db, in.Type, in.Search)
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
	util.PubRPCSuccRsp(w, "fetch", "FetchActivity")
	return &fetch.ActivityReply{
		Head:     &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid},
		Activity: &info}, nil
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
	rows, err := db.Query("SELECT u.uid, u.username, u.phone, f.content, f.ctime, u.term FROM feedback f, user u WHERE f.uid = u.uid ORDER BY f.id DESC LIMIT ?, ?", seq, num)
	if err != nil {
		log.Printf("getFeedback query failed:%v", err)
		return infos
	}

	defer rows.Close()
	for rows.Next() {
		var info fetch.FeedbackInfo
		var phone string
		err := rows.Scan(&info.Uid, &info.Phone, &phone, &info.Content, &info.Ctime,
			&info.Term)
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
	if term == 0 && (version <= 6 || version == 11) || term == 1 && version == 10 {
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

func (s *server) FetchLatestVersion(ctx context.Context, in *fetch.VersionRequest) (*fetch.VersionReply, error) {
	util.PubRPCRequest(w, "fetch", "FetchLatestVersion")
	log.Printf("FetchLatestVersion request uid:%d term:%d versoin:%d channel:%s",
		in.Head.Uid, in.Head.Term, in.Head.Version, in.Channel)
	var version int64
	var vname, downurl, title, desc string
	err := db.QueryRow("SELECT version, vname, downurl, title, description FROM app_channel WHERE channel = ?",
		in.Channel).Scan(&version, &vname, &downurl, &title, &desc)
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
		Version: vname, Downurl: downurl, Title: title, Desc: desc}, nil
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
	rows, err := db.Query("SELECT id, channel, cname, version, vname, downurl, title, description FROM app_channel ORDER BY id LIMIT ?, ?",
		seq, num)
	if err != nil {
		log.Printf("getChannelVersion failed:%v", err)
		return infos
	}

	defer rows.Close()
	for rows.Next() {
		var info common.ChannelVersionInfo
		err := rows.Scan(&info.Id, &info.Channel, &info.Cname, &info.Version,
			&info.Vname, &info.Downurl, &info.Title, &info.Desc)
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

func getTotalMonitor(db *sql.DB, name string) int64 {
	var cnt int64
	err := db.QueryRow("SELECT COUNT(id) FROM monitor.api_stat WHERE name = ?", name).Scan(&cnt)
	if err != nil {
		log.Printf("getTotalMonitor failed:%v", err)
	}
	return cnt
}

func getMonitor(db *sql.DB, seq, num int64, name string) []*fetch.MonitorInfo {
	var infos []*fetch.MonitorInfo
	query := fmt.Sprintf("SELECT id, req, succrsp, ctime FROM monitor.api_stat WHERE name = '%s' ", name)
	query += fmt.Sprintf(" ORDER BY id DESC LIMIT %d, %d", seq, num)
	log.Printf("getMonitor query:%s", query)
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("getMonitor query failed;%v", err)
		return infos
	}

	defer rows.Close()
	for rows.Next() {
		var info fetch.MonitorInfo
		err := rows.Scan(&info.Id, &info.Req, &info.Succrsp, &info.Ctime)
		if err != nil {
			log.Printf("getMonitor scan failed:%v", err)
			continue
		}
		infos = append(infos, &info)
	}
	return infos

}

func (s *server) FetchMonitor(ctx context.Context, in *fetch.MonitorRequest) (*fetch.MonitorReply, error) {
	util.PubRPCRequest(w, "fetch", "FetchMonitor")
	log.Printf("FetchMonitor seq:%d num:%d uid:%d",
		in.Seq, in.Num, in.Head.Uid)
	infos := getMonitor(db, in.Seq, in.Num, in.Name)
	total := getTotalMonitor(db, in.Name)
	util.PubRPCSuccRsp(w, "fetch", "FetchMonitor")
	return &fetch.MonitorReply{
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

	s := util.NewGrpcServer()
	fetch.RegisterFetchServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
