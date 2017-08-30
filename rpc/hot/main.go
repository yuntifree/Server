package main

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"Server/proto/common"
	"Server/proto/hot"
	"Server/util"

	_ "github.com/go-sql-driver/mysql"
	nsq "github.com/nsqio/go-nsq"
	"golang.org/x/net/context"
)

const (
	homeNewsNum     = 10
	saveRate        = 0.1 / (1024.0 * 1024.0)
	marqueeInterval = 30
	weatherDst      = "http://www.dg121.com/mobile"
	jokeTime        = 1487779200 // 2017-02-23
	hourSeconds     = 3600
)

const (
	typeHotNews = iota
	typeVideos
	typeApp
	typeGame
	typeDgNews
	typeAmuse
	typeJoke
)

type server struct{}

var db *sql.DB
var w *nsq.Producer

func getMaxNewsSeq(db *sql.DB) int64 {
	var id int64
	err := db.QueryRow("SELECT MAX(id) FROM news").Scan(&id)
	if err != nil {
		log.Printf("getMaxNewsSeq query failed:%v", err)
	}
	return id
}

func queryNews(db *sql.DB, query string) []*hot.HotsInfo {
	var infos []*hot.HotsInfo
	log.Printf("query string:%s", query)
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("query failed:%v", err)
		return infos
	}
	defer rows.Close()

	for rows.Next() {
		var img [3]string
		var info hot.HotsInfo
		err = rows.Scan(&info.Seq, &info.Title, &img[0], &img[1], &img[2],
			&info.Source, &info.Dst, &info.Ctime, &info.Stype)
		if err != nil {
			log.Printf("scan rows failed: %v", err)
			continue
		}
		info.Id = info.Seq
		info.Stype = 0
		var pics [3]string
		j, k := 0, 0
		for ; j < 3; j++ {
			if img[j] != "" {
				pics[k] = img[j]
				k++
			}
		}
		info.Images = pics[:k]
		infos = append(infos, &info)
	}
	return infos
}

func getHotNews(db *sql.DB, seq, num int64) []*hot.HotsInfo {
	query := "SELECT id, title, img1, img2, img3, source, dst, ctime, stype FROM news WHERE deleted = 0 AND stype IN (0,1,2,3) AND top = 0 "
	if seq != 0 {
		query += " AND id < " + strconv.Itoa(int(seq))
	}
	query += " ORDER BY id DESC LIMIT " + strconv.Itoa(int(num))
	return queryNews(db, query)
}

func getDgNews(db *sql.DB, seq, num int64) []*hot.HotsInfo {
	query := "SELECT id, title, img1, img2, img3, source, dst, ctime, UNIX_TIMESTAMP(ctime) FROM news WHERE deleted = 0 AND top = 0 AND stype = 10 AND source != '东莞阳光网' "
	if seq != 0 {
		query += fmt.Sprintf(" AND ctime < FROM_UNIXTIME(%d,", seq)
		query += "'%y-%m-%d')"
	}
	query += " ORDER BY ctime DESC LIMIT " + strconv.Itoa(int(num))
	var infos []*hot.HotsInfo
	log.Printf("query string:%s", query)
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("query failed:%v", err)
		return infos
	}
	defer rows.Close()

	for rows.Next() {
		var img [3]string
		var info hot.HotsInfo
		err = rows.Scan(&info.Id, &info.Title, &img[0], &img[1], &img[2],
			&info.Source, &info.Dst, &info.Ctime, &info.Seq)
		if err != nil {
			log.Printf("scan rows failed: %v", err)
			continue
		}
		info.Stype = 0
		var pics [3]string
		j, k := 0, 0
		for ; j < 3; j++ {
			if img[j] != "" {
				pics[k] = img[j]
				k++
			}
		}
		info.Images = pics[:k]
		infos = append(infos, &info)
	}
	return infos
}

func getNews(db *sql.DB, seq, num, newsType int64) []*hot.HotsInfo {
	query := "SELECT id, title, img1, img2, img3, source, dst, ctime, stype FROM news WHERE deleted = 0 AND top = 0 AND stype = " +
		strconv.Itoa(int(newsType))
	if seq != 0 {
		query += " AND id < " + strconv.Itoa(int(seq))
	}
	query += " ORDER BY id DESC LIMIT " + strconv.Itoa(int(num))
	return queryNews(db, query)
}

func getHospitalNews(db *sql.DB, seq, num, hid int64) []*hot.HotsInfo {
	query := "SELECT id, title, img1, img2, img3, source, dst, ctime, stype FROM hospital_news WHERE deleted = 0 AND top = 0 AND hid = " +
		strconv.Itoa(int(hid))
	if seq != 0 {
		query += " AND id < " + strconv.Itoa(int(seq))
	}
	query += " ORDER BY id DESC LIMIT " + strconv.Itoa(int(num))
	return queryNews(db, query)
}

func getTopNews(db *sql.DB, newsType int64) []*hot.HotsInfo {
	query := "SELECT id, title, img1, img2, img3, source, dst, ctime, stype FROM news WHERE deleted = 0 AND top = 1 AND stype = " +
		strconv.Itoa(int(newsType))
	return queryNews(db, query)
}

func getVideos(db *sql.DB, seq int64) []*hot.HotsInfo {
	var infos []*hot.HotsInfo
	query := "SELECT vid, title, img, source, dst, ctime, play FROM youku_video WHERE 1 = 1 AND duration < 300 AND deleted = 0 "
	if seq != 0 {
		query += " AND vid < " + strconv.Itoa(int(seq))
	}
	query += " ORDER BY vid DESC LIMIT " + strconv.Itoa(util.MaxListSize)
	log.Printf("query string:%s", query)
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("query failed:%v", err)
		return infos
	}
	defer rows.Close()

	for rows.Next() {
		var img [3]string
		var info hot.HotsInfo
		err = rows.Scan(&info.Seq, &info.Title, &img[0], &info.Source, &info.Dst,
			&info.Ctime, &info.Play)
		if err != nil {
			log.Printf("scan rows failed: %v", err)
			return infos
		}
		info.Images = img[:1]
		info.Id = info.Seq
		info.Dst += "&uc_param_str=frdnpfvecplabtbmntnwpvssbinipr"
		infos = append(infos, &info)
	}
	return infos
}

func cleanContent(content string) string {
	str := content
	if content[0] == '.' {
		str = content[1:]
	}
	str = strings.Replace(str, "&nbsp;", "", -1)
	str = strings.TrimSpace(str)
	return str
}

func getJokes(db *sql.DB, seq, num int64) []*hot.JokeInfo {
	var infos []*hot.JokeInfo
	query := "SELECT id, content, heart, bad FROM joke WHERE dst = '' "
	if seq != 0 {
		query += fmt.Sprintf(" AND id < %d", seq)
	}
	query += fmt.Sprintf(" ORDER BY id DESC LIMIT %d", num)

	rows, err := db.Query(query)
	if err != nil {
		log.Printf("getJokes query failed:%v", err)
		return infos
	}

	defer rows.Close()
	for rows.Next() {
		var info hot.JokeInfo
		var content string
		err := rows.Scan(&info.Id, &content, &info.Heart, &info.Bad)
		if err != nil {
			log.Printf("getJokes scan failed:%v", err)
			continue
		}
		info.Content = cleanContent(content)
		info.Seq = info.Id
		infos = append(infos, &info)
	}
	return infos
}

func calcJokeSeq() int64 {
	return (time.Now().Unix() - jokeTime) / hourSeconds * 100
}

func getAdBanner(db *sql.DB, adtype int64) *hot.HotsInfo {
	var info hot.HotsInfo
	var img string
	err := db.QueryRow("SELECT img, dst FROM ad_banner WHERE type = ? AND stype = 2 AND online = 1 AND deleted = 0 ORDER BY id DESC LIMIT 1", adtype).Scan(&img, &info.Dst)
	if err != nil {
		log.Printf("getAdvertise query failed:%v", err)
		return nil
	}
	info.Images = append(info.Images, img)
	info.Stype = 1
	return &info
}

func getAdvertise(db *sql.DB, adtype int64) *hot.HotsInfo {
	var info hot.HotsInfo
	var img string
	err := db.QueryRow("SELECT name, img, dst FROM advertise WHERE areaid = ? AND type = 1", adtype).Scan(&info.Title, &img, &info.Dst)
	if err != nil {
		log.Printf("getAdvertise query failed:%v", err)
		return nil
	}
	info.Images = append(info.Images, img)
	info.Stype = 1
	return &info
}
func (s *server) GetHospitalNews(ctx context.Context, in *common.CommRequest) (*hot.HotsReply, error) {
	util.PubRPCRequest(w, "hot", "GetHospitalNews")
	infos := getHospitalNews(db, in.Seq, util.MaxListSize, in.Type)
	util.PubRPCSuccRsp(w, "hot", "GetHospitalNews")
	return &hot.HotsReply{
		Head:  &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid},
		Infos: infos}, nil
}

func getTopVideo(uid int64, token string) *hot.TopInfo {
	return &hot.TopInfo{
		Title: "360安全教育视频集",
		Dst:   fmt.Sprintf("http://wx.yunxingzh.com/app/video.html?uid=%d&token=%s", uid, token),
		Img:   "http://img.yunxingzh.com/a9c36ff0-486c-4e3a-874a-fe8c5f61e09b.png",
	}
}

func getUserToken(db *sql.DB, uid int64) string {
	var token string
	err := db.QueryRow("SELECT token FROM user WHERE uid = ?", uid).Scan(&token)
	if err != nil {
		log.Printf("getUserToken failed:%v", err)
	}
	return token
}

func (s *server) GetHots(ctx context.Context, in *common.CommRequest) (*hot.HotsReply, error) {
	util.PubRPCRequest(w, "hot", "GetHots")
	log.Printf("request uid:%d, sid:%s ctype:%d, seq:%d term:%d version:%d subtype:%d",
		in.Head.Uid, in.Head.Sid, in.Type, in.Seq, in.Head.Term, in.Head.Version,
		in.Subtype)
	var infos []*hot.HotsInfo
	var top *hot.TopInfo
	if in.Type == typeHotNews {
		if util.CheckTermVersion(in.Head.Term, in.Head.Version) {
			infos = getHotNews(db, in.Seq, util.MaxListSize)
			if in.Seq == 0 {
				max := getMaxNewsSeq(db)
				tops := getTopNews(db, 0)
				if in.Subtype != 0 {
					ad := getAdBanner(db, in.Subtype)
					log.Printf("ad:%v", ad)
					if ad != nil {
						tops = append(tops, ad)
					}
				}
				for i := 0; i < len(tops); i++ {
					tops[i].Seq += max
				}
				infos = append(tops, infos...)
			}
		} else {
			if in.Seq == 0 {
				max := getMaxNewsSeq(db)
				tops := getTopNews(db, 0)
				for i := 0; i < len(tops); i++ {
					tops[i].Seq += max
				}
				infos = getDgNews(db, in.Seq, util.MaxListSize/2)
				for i := 0; i < len(infos); i++ {
					infos[i].Seq += max
				}
				infos = append(tops, infos...)
			} else {
				infos = getHotNews(db, in.Seq, util.MaxListSize)
			}
		}
	} else if in.Type == typeVideos {
		infos = getVideos(db, in.Seq)
		if in.Seq == 0 {
			token := getUserToken(db, in.Head.Uid)
			top = getTopVideo(in.Head.Uid, token)
		}
	} else if in.Type == typeDgNews {
		infos = getDgNews(db, in.Seq, util.MaxListSize)
	} else if in.Type == typeAmuse {
		infos = getNews(db, in.Seq, util.MaxListSize, 4)
	}
	util.PubRPCSuccRsp(w, "hot", "GetHots")
	return &hot.HotsReply{
		Head:  &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid},
		Infos: infos, Top: top}, nil
}

func getCategoryTitleIcon(category int) (string, string) {
	switch category {
	default:
		return "智慧政务", "http://file.yunxingzh.com/ico_government_xxxh.png"
	case 2:
		return "交通出行", "http://file.yunxingzh.com/ico_traffic_xxxh.png"
	case 3:
		return "医疗服务", "http://file.yunxingzh.com/ico_medical_xxxh.png"
	case 4:
		return "网上充值", "http://file.yunxingzh.com/ico_recharge.png"
	}
}

func getService(db *sql.DB, term int64) ([]*hot.ServiceCategory, error) {
	var infos []*hot.ServiceCategory
	rows, err := db.Query("SELECT title, dst, category, sid, icon FROM service WHERE category != 0 AND deleted = 0 AND dst != '' ORDER BY category")
	if err != nil {
		log.Printf("query failed:%v", err)
		return infos, err
	}
	defer rows.Close()

	category := 0
	var srvs []*hot.ServiceInfo
	for rows.Next() {
		var info hot.ServiceInfo
		var cate int
		err := rows.Scan(&info.Title, &info.Dst, &cate, &info.Sid, &info.Icon)
		if err != nil {
			continue
		}

		if cate != category {
			if len(srvs) > 0 {
				var cateinfo hot.ServiceCategory
				cateinfo.Title, cateinfo.Icon = getCategoryTitleIcon(category)
				cateinfo.Items = srvs[:]
				infos = append(infos, &cateinfo)
				srvs = srvs[len(srvs):]
			}
			category = cate
		}
		if info.Title == "公交查询" && term == util.WxTerm {
			continue
		}
		srvs = append(srvs, &info)
	}

	if len(srvs) > 0 {
		var cateinfo hot.ServiceCategory
		cateinfo.Title, cateinfo.Icon = getCategoryTitleIcon(category)
		cateinfo.Items = srvs[:]
		infos = append(infos, &cateinfo)
	}

	return infos, nil
}

func (s *server) GetServices(ctx context.Context, in *common.CommRequest) (*hot.ServiceReply, error) {
	util.PubRPCRequest(w, "hot", "GetServices")
	categories, err := getService(db, in.Head.Term)
	if err != nil {
		log.Printf("getServie failed:%v", err)
		return &hot.ServiceReply{Head: &common.Head{Retcode: 1}}, err
	}

	util.PubRPCSuccRsp(w, "hot", "GetServices")
	return &hot.ServiceReply{
		Head: &common.Head{Retcode: 0}, Services: categories}, nil
}

func getWeather(db *sql.DB) (hot.WeatherInfo, error) {
	var info hot.WeatherInfo
	err := db.QueryRow("SELECT type, temp, info FROM weather ORDER BY wid DESC LIMIT 1").
		Scan(&info.Type, &info.Temp, &info.Info)
	if err != nil {
		log.Printf("select weather failed:%v", err)
		return info, err
	}
	info.Dst = weatherDst
	return info, nil
}

func getNotice(db *sql.DB) *hot.NoticeInfo {
	var info hot.NoticeInfo
	err := db.QueryRow("SELECT title, content, dst FROM notice WHERE etime > NOW() ORDER BY id DESC LIMIT 1").
		Scan(&info.Title, &info.Content, &info.Dst)
	if err != nil {
		return nil
	}
	return &info
}

func (s *server) GetWeatherNews(ctx context.Context, in *common.CommRequest) (*hot.WeatherNewsReply, error) {
	util.PubRPCRequest(w, "hot", "GetWeatherNews")
	weather, err := getWeather(db)
	if err != nil {
		log.Printf("getWeather failed:%v", err)
		return &hot.WeatherNewsReply{Head: &common.Head{Retcode: 1}}, err
	}

	infos := getDgNews(db, 0, homeNewsNum)
	if len(infos) >= 9 {
		infos = append(infos[:0], infos[1], infos[3], infos[5], infos[7], infos[8])
	}
	notice := getNotice(db)
	util.PubRPCSuccRsp(w, "hot", "GetWeatherNews")
	return &hot.WeatherNewsReply{Head: &common.Head{Retcode: 0},
		Weather: &weather, News: infos, Notice: notice}, nil
}

func getUseInfo(db *sql.DB, uid int64) (hot.UseInfo, error) {
	var info hot.UseInfo
	err := db.QueryRow("SELECT times, traffic FROM user WHERE uid = ?", uid).
		Scan(&info.Total, &info.Save)
	if err != nil {
		log.Printf("select use info failed:%v", err)
		return info, err
	}
	return info, nil
}

func getBanners(db *sql.DB, flag bool) ([]*common.BannerInfo, error) {
	var infos []*common.BannerInfo
	query := "SELECT img, dst FROM banner WHERE deleted = 0 AND type = 0"
	if flag {
		query += " AND (online = 1 OR dbg = 1) "
	} else {
		query += " AND online = 1 "
	}
	query += " ORDER BY priority DESC LIMIT 20"
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("select banner info failed:%v", err)
		return infos, err
	}
	for rows.Next() {
		var info common.BannerInfo
		err := rows.Scan(&info.Img, &info.Dst)
		if err != nil {
			log.Printf("scan failed:%v", err)
			continue
		}

		infos = append(infos, &info)

	}
	return infos, nil
}

func (s *server) GetFrontInfo(ctx context.Context, in *common.CommRequest) (*hot.FrontReply, error) {
	util.PubRPCRequest(w, "hot", "GetFrontInfo")
	uinfo, err := getUseInfo(db, in.Head.Uid)
	if err != nil {
		log.Printf("getUseInfo failed:%v", err)
		return &hot.FrontReply{Head: &common.Head{Retcode: 1}}, err
	}

	uinfo.Save = int64(float64(uinfo.Save) * saveRate)
	flag := util.IsWhiteUser(db, in.Head.Uid, util.BannerWhiteType)
	binfos, err := getBanners(db, flag)
	if err != nil {
		log.Printf("getBannerInfo failed:%v", err)
		return &hot.FrontReply{Head: &common.Head{Retcode: 1}}, err
	}

	util.PubRPCSuccRsp(w, "hot", "GetFrontInfo")
	return &hot.FrontReply{
		Head: &common.Head{Retcode: 0}, User: &uinfo, Banner: binfos}, nil
}

func isNewUser(db *sql.DB, uid int64) bool {
	if uid == 0 {
		return false
	}

	var flag bool
	err := db.QueryRow("SELECT IF(ctime > CURDATE(), true, false) FROM user WHERE uid = ?", uid).
		Scan(&flag)
	if err != nil {
		return false
	}
	return flag
}

func getLiveInfos(db *sql.DB, seq int64) []*hot.LiveInfo {
	var infos []*hot.LiveInfo
	query := "SELECT uid, avatar, nickname, live_id, img, p_time, location, watches, live, priority FROM live WHERE seq > (UNIX_TIMESTAMP(NOW())-180)*1000 "
	if seq != 0 {
		query += fmt.Sprintf(" AND priority < %d ", seq)
	}
	query += fmt.Sprintf(" ORDER BY priority DESC LIMIT %d", util.MaxListSize)
	log.Printf("getLiveInfos query:%s", query)
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("getLiveInfos query failed:%v", err)
		return infos
	}
	defer rows.Close()
	for rows.Next() {
		var info hot.LiveInfo
		err := rows.Scan(&info.Uid, &info.Avatar, &info.Nickname, &info.LiveId,
			&info.Img, &info.PTime, &info.Location, &info.Watches, &info.Live,
			&info.Seq)
		if err != nil {
			log.Printf("getLiveInfos scan failed:%v", err)
			continue
		}
		infos = append(infos, &info)
	}
	return infos
}

func (s *server) GetLive(ctx context.Context, in *common.CommRequest) (*hot.LiveReply, error) {
	util.PubRPCRequest(w, "hot", "GetLive")
	log.Printf("GetLive uid:%d seq:%d", in.Head.Uid, in.Seq)
	infos := getLiveInfos(db, in.Seq)
	util.PubRPCSuccRsp(w, "hot", "GetLive")
	return &hot.LiveReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}, List: infos}, nil
}

func (s *server) GetJoke(ctx context.Context, in *common.CommRequest) (*hot.JokeReply, error) {
	util.PubRPCRequest(w, "hot", "GetJoke")
	log.Printf("GetJoke uid:%d seq:%d", in.Head.Uid, in.Seq)
	seq := in.Seq
	if in.Seq == 0 {
		seq = calcJokeSeq()
	}
	infos := getJokes(db, seq, util.MaxListSize)
	util.PubRPCSuccRsp(w, "hot", "GetJoke")
	return &hot.JokeReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}, Infos: infos}, nil
}

func main() {
	lis, err := net.Listen("tcp", util.HotServerPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	w = util.NewNsqProducer()
	db, err = util.InitDB(true)
	if err != nil {
		log.Fatalf("failed to init db connection: %v", err)
	}
	db.SetMaxIdleConns(util.MaxIdleConns)
	kv := util.InitRedis()
	go util.ReportHandler(kv, util.HotServerName, util.HotServerPort)

	s := util.NewGrpcServer()
	hot.RegisterHotServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
