package main

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"

	common "../../proto/common"
	hot "../../proto/hot"
	util "../../util"
	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const (
	homeNewsNum     = 6
	saveRate        = 0.1 / (1024.0 * 1024.0)
	marqueeInterval = 30
)

const (
	typeHotNews = iota
	typeVideos
	typeApp
	typeGame
	typeDgNews
)

type server struct{}

var db *sql.DB

func getDgNews(db *sql.DB, seq, num int64) []*hot.HotsInfo {
	return getNews(db, seq, num, true)
}

func getHotNews(db *sql.DB, seq, num int64) []*hot.HotsInfo {
	return getNews(db, seq, num, false)
}

func getNews(db *sql.DB, seq, num int64, isDgNews bool) []*hot.HotsInfo {
	var infos []*hot.HotsInfo
	query := "SELECT id, title, img1, img2, img3, source, dst, ctime, stype FROM news WHERE deleted = 0 "
	if isDgNews {
		query += " AND stype = 10 "
	} else {
		query += " AND stype != 10 "
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
	defer rows.Close()

	for rows.Next() {
		var img [3]string
		var info hot.HotsInfo
		err = rows.Scan(&info.Seq, &info.Title, &img[0], &img[1], &img[2], &info.Source, &info.Dst, &info.Ctime, &info.Stype)
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
				log.Printf("k:%d pic:%s", k, pics[k])
				k++
			}
		}
		info.Images = pics[:k]
		infos = append(infos, &info)
		log.Printf("title:%s source:%s", info.Title, info.Source)
	}
	return infos
}

func getVideos(db *sql.DB, seq int64) []*hot.HotsInfo {
	var infos []*hot.HotsInfo
	query := "SELECT vid, title, img, source, dst, ctime, play FROM youku_video WHERE 1 = 1 AND duration < 300 "
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
		err = rows.Scan(&info.Seq, &info.Title, &img[0], &info.Source, &info.Dst, &info.Ctime, &info.Play)
		if err != nil {
			log.Printf("scan rows failed: %v", err)
			return infos
		}
		info.Images = img[:1]
		info.Id = info.Seq
		infos = append(infos, &info)
		log.Printf("title:%s source:%s", info.Title, info.Source)
	}
	return infos
}

func (s *server) GetHots(ctx context.Context, in *common.CommRequest) (*hot.HotsReply, error) {
	log.Printf("request uid:%d, sid:%s ctype:%d, seq:%d", in.Head.Uid, in.Head.Sid, in.Type, in.Seq)
	var infos []*hot.HotsInfo
	if in.Type == typeHotNews {
		infos = getHotNews(db, in.Seq, util.MaxListSize)
	} else if in.Type == typeVideos {
		infos = getVideos(db, in.Seq)
	} else if in.Type == typeDgNews {
		infos = getDgNews(db, in.Seq, util.MaxListSize)
	}
	return &hot.HotsReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid}, Infos: infos}, nil
}

func getCategoryTitleIcon(category int) (string, string) {
	switch category {
	default:
		return "智慧政务", "http://file.yunxingzh.com/ico_government.png"
	case 2:
		return "交通出行", "http://file.yunxingzh.com/ico_traffic.png"
	case 3:
		return "医疗服务", "http://file.yunxingzh.com/ico_medical.png"
	case 4:
		return "网上充值", "http://file.yunxingzh.com/ico_recharge.png"
	}
}

func getService(db *sql.DB) ([]*hot.ServiceCategory, error) {
	var infos []*hot.ServiceCategory
	rows, err := db.Query("SELECT title, dst, category, sid FROM service WHERE category != 0 AND deleted = 0 AND dst != '' ORDER BY category")
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
		err := rows.Scan(&info.Title, &info.Dst, &cate, &info.Sid)
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
	categories, err := getService(db)
	if err != nil {
		log.Printf("getServie failed:%v", err)
		return &hot.ServiceReply{Head: &common.Head{Retcode: 1}}, err
	}

	return &hot.ServiceReply{Head: &common.Head{Retcode: 0}, Services: categories}, nil
}

func getWeather(db *sql.DB) (hot.WeatherInfo, error) {
	var info hot.WeatherInfo
	err := db.QueryRow("SELECT type, temp, info FROM weather ORDER BY wid DESC LIMIT 1").Scan(&info.Type, &info.Temp, &info.Info)
	if err != nil {
		log.Printf("select weather failed:%v", err)
		return info, err
	}
	return info, nil
}

func (s *server) GetWeatherNews(ctx context.Context, in *common.CommRequest) (*hot.WeatherNewsReply, error) {
	weather, err := getWeather(db)
	if err != nil {
		log.Printf("getWeather failed:%v", err)
		return &hot.WeatherNewsReply{Head: &common.Head{Retcode: 1}}, err
	}

	infos := getHotNews(db, 0, homeNewsNum)
	infos = append(infos[:0], infos[1], infos[3], infos[5])
	return &hot.WeatherNewsReply{Head: &common.Head{Retcode: 0}, Weather: &weather, News: infos}, nil
}

func getUseInfo(db *sql.DB, uid int64) (hot.UseInfo, error) {
	var info hot.UseInfo
	err := db.QueryRow("SELECT times, traffic FROM user WHERE uid = ?", uid).Scan(&info.Total, &info.Save)
	if err != nil {
		log.Printf("select use info failed:%v", err)
		return info, err
	}
	return info, nil
}

func getBanners(db *sql.DB) ([]*hot.BannerInfo, error) {
	var infos []*hot.BannerInfo
	rows, err := db.Query("SELECT img, dst FROM banner WHERE deleted = 0 AND online = 1 ORDER BY id DESC LIMIT 20")
	if err != nil {
		log.Printf("select banner info failed:%v", err)
		return infos, err
	}
	for rows.Next() {
		var info hot.BannerInfo
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
	uinfo, err := getUseInfo(db, in.Head.Uid)
	if err != nil {
		log.Printf("getUseInfo failed:%v", err)
		return &hot.FrontReply{Head: &common.Head{Retcode: 1}}, err
	}

	uinfo.Save = int32(float64(uinfo.Save) * saveRate)
	binfos, err := getBanners(db)
	if err != nil {
		log.Printf("getBannerInfo failed:%v", err)
		return &hot.FrontReply{Head: &common.Head{Retcode: 1}}, err
	}

	return &hot.FrontReply{Head: &common.Head{Retcode: 0}, Uinfo: &uinfo, Binfos: binfos}, nil
}

func getSalesCount(db *sql.DB, sid, uid int64) int32 {
	var num int32
	err := db.QueryRow("SELECT COUNT(*) FROM sales_history WHERE uid = ? AND sid = ?", uid, sid).
		Scan(&num)
	if err != nil {
		log.Printf("getSalesCount query failed:%v", err)
	}
	return num
}

func getOpenedSales(db *sql.DB, num int32, seq int64) []*hot.BidInfo {
	var opened []*hot.BidInfo
	query := `SELECT sid, s.gid, num, title, UNIX_TIMESTAMP(s.ctime), UNIX_TIMESTAMP(s.etime),
	 image, total, win_uid, win_code, nickname, s.status, sub_title FROM sales s, 
	 goods g, user i WHERE s.gid = g.gid AND s.win_uid = i.uid AND s.status >= 3 `
	if seq != 0 {
		query += fmt.Sprintf(" AND UNIX_TIMESTAMP(etime) < %d ", seq)
	}
	query += fmt.Sprintf(" ORDER BY s.etime DESC LIMIT %d", num)
	log.Printf("getOpenedSales query:%s", query)
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("getOpenedSales query failed:%v", err)
		return opened
	}
	defer rows.Close()

	for rows.Next() {
		var info hot.BidInfo
		var award hot.AwardInfo
		err := rows.Scan(&info.Bid, &info.Gid, &info.Period, &info.Title, &info.Start,
			&info.End, &info.Image, &info.Total, &award.Uid, &award.Awardcode,
			&award.Nickname, &info.Status, &info.Subtitle)
		if err != nil {
			log.Printf("getOpenedSales scan failed:%v", err)
			continue
		}
		info.Start *= 1000
		info.End *= 1000
		log.Printf("bid:%d gid:%d", info.Bid, info.Gid)
		award.Num = getSalesCount(db, info.Bid, award.Uid)
		info.Award = &award
		opened = append(opened, &info)
	}
	return opened
}

func getRemainSeconds(tt time.Time) int32 {
	award := util.GetNextCqssc(tt)
	award = award.Add(120 * time.Second)
	return int32(award.Unix() - tt.Unix())
}

func getOpeningSales(db *sql.DB, num int32) []*hot.BidInfo {
	var opening []*hot.BidInfo
	query := `SELECT sid, s.gid, num, title, UNIX_TIMESTAMP(s.ctime), UNIX_TIMESTAMP(etime), 
	image, s.total, s.status, sub_title FROM sales s, goods g WHERE s.gid = g.gid 
	AND s.status = 2  ORDER BY etime DESC `
	if num != 0 {
		query += fmt.Sprintf(" LIMIT %d", num)
	}
	log.Printf("getOpening query:%s", query)
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("getOpening query failed:%v", err)
		return opening
	}
	defer rows.Close()

	for rows.Next() {
		var info hot.BidInfo
		var end int64
		err = rows.Scan(&info.Bid, &info.Gid, &info.Period, &info.Title, &info.Start, &end,
			&info.Image, &info.Total, &info.Status, &info.Subtitle)
		if err != nil {
			log.Printf("getOpening scan failed:%v", err)
			continue
		}
		info.Start *= 1000
		info.Seq = info.Bid
		tt := time.Unix(end, 0)
		info.Rest = getRemainSeconds(tt)
		opening = append(opening, &info)
	}
	if len(opening) < int(num) {
		opened := getOpenedSales(db, num-int32(len(opening)), 0)
		opening = append(opening, opened...)
	}
	return opening
}

func hasPhone(db *sql.DB, uid int64) bool {
	var phone string
	err := db.QueryRow("SELECT phone FROM user WHERE uid = ?", uid).
		Scan(&phone)
	if err != nil {
		return false
	}
	if phone == "" {
		return false
	}
	return true
}

func hasReceipt(db *sql.DB, uid int64) bool {
	var num int
	err := db.QueryRow("SELECT COUNT(lid) FROM logistics WHERE status = 5 AND uid = ?", uid).
		Scan(&num)
	if err != nil {
		return false
	}
	if num > 0 {
		return true
	}
	return false
}

func hasShare(db *sql.DB, uid int64) bool {
	var num int
	err := db.QueryRow("SELECT COUNT(lid) FROM logistics WHERE share = 0 AND status >= 6 AND uid = ?", uid).
		Scan(&num)
	if err != nil {
		return false
	}
	if num > 0 {
		return true
	}
	return false
}

func hasReddot(db *sql.DB, uid int64) bool {
	if uid == 0 {
		return false
	}

	if !hasPhone(db, uid) || hasReceipt(db, uid) || hasShare(db, uid) {
		return true
	}

	return false
}

func (s *server) GetLatest(ctx context.Context, in *common.CommRequest) (*hot.LatestReply, error) {
	log.Printf("GetLatest uid:%d seq:%d, num:%d", in.Head.Uid, in.Seq, in.Num)
	var opening, opened []*hot.BidInfo
	if in.Seq == 0 {
		opening = getOpeningSales(db, 0)
	}
	opened = getOpenedSales(db, in.Num, in.Seq)
	reddot := 0
	if hasReddot(db, in.Head.Uid) {
		reddot = 1
	}
	return &hot.LatestReply{Head: &common.Head{Retcode: 0},
		Opening: opening, Opened: opened, Reddot: int32(reddot)}, nil
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

func getRunningSales(db *sql.DB, uid int64, num int32, seq int64) []*hot.BidInfo {
	var infos []*hot.BidInfo
	flag := isNewUser(db, uid)
	query := `SELECT sid, s.gid, num, title, UNIX_TIMESTAMP(s.ctime), UNIX_TIMESTAMP(s.etime), 
		image, total, remain, g.priority, sub_title, g.new_rank FROM sales s, goods g 
		WHERE s.gid = g.gid AND s.status = 1`
	if seq > 0 {
		if flag {
			query += fmt.Sprintf(" AND g.new_rank < %d ORDER BY g.new_rank DESC ", seq)
		} else {
			query += fmt.Sprintf(" AND g.priority < %d ORDER BY g.priority DESC ", seq)
		}
		if num > 0 {
			query += fmt.Sprintf(" LIMIT %d", num)
		}
	}

	log.Printf("getRunningSales query:%s", query)
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("getRunningSales query failed:%v", err)
		return infos
	}
	defer rows.Close()

	for rows.Next() {
		var info hot.BidInfo
		var rank int64
		err := rows.Scan(&info.Bid, &info.Gid, &info.Period, &info.Title,
			&info.Image, &info.Total, &info.Remain, &info.Seq, &info.Subtitle,
			&rank)
		if err != nil {
			log.Printf("getRunningSales scan failed:%v", err)
			continue
		}
		if flag {
			info.Seq = rank
		}
		infos = append(infos, &info)
	}
	return infos
}

func getGrapInfo(db *sql.DB, interval int) []*hot.MarqueeInfo {
	var infos []*hot.MarqueeInfo
	query := fmt.Sprintf("SELECT h.uid, u.nickname, g.name FROM sales_hisotry h, user u, sales s, goods g WHERE h.uid = u.uid AND h.sid = s.sid AND s.gid = g.gid AND h.ctime > DATE_SUB(NOW(), INTERVAL %d MINUTE) ORDER BY h.ctime DESC LIMIT 50", interval)
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("getGrapInfo query failed:%v", err)
		return infos
	}
	defer rows.Close()

	for rows.Next() {
		var info hot.MarqueeInfo
		err := rows.Scan(&info.Uid, &info.Nickname, &info.Gname)
		if err != nil {
			log.Printf("getGrapInfo scan failed:%v", err)
			continue
		}
		infos = append(infos, &info)
	}
	return infos
}

func getAwardInfo(db *sql.DB, interval int) []*hot.MarqueeInfo {
	var infos []*hot.MarqueeInfo
	query := fmt.Sprintf("SELECT s.sid, s.win_uid, s.num, g.name, u.nickname FROM sales s, goods g, user u WHERE s.win_uid = u.uid AND s.gid = g.gid AND s.etime > DATE_SUB(NOW(), INTERVAL %d MINUTE)", interval)
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("getAwardInfo query failed:%v", err)
		return infos
	}
	defer rows.Close()

	for rows.Next() {
		var info hot.MarqueeInfo
		err := rows.Scan(&info.Bid, &info.Uid, &info.Nickname, &info.Gname)
		if err != nil {
			log.Printf("getAwardInfo scan failed:%v", err)
			continue
		}
		info.Type = 1
		infos = append(infos, &info)
	}
	return infos
}

func getMarquee(db *sql.DB, interval int) []*hot.MarqueeInfo {
	grap := getGrapInfo(db, interval)
	award := getAwardInfo(db, interval)
	marquee := append(grap, award...)
	return marquee
}

func getPromotion(db *sql.DB) hot.PromotionInfo {
	var info hot.PromotionInfo
	err := db.QueryRow("SELECT title, target FROM promotion WHERE online = 1 AND deleted = 0 ORDER BY id DESC LIMIT 1").
		Scan(&info.Title, &info.Target)
	if err != nil {
		log.Printf("get promotion failed:%v", err)
	}
	return info
}

func getSlides(db *sql.DB) []*hot.SlideInfo {
	var infos []*hot.SlideInfo
	rows, err := db.Query("SELECT image, target FROM slides WHERE online = 1 AND deleted = 0 ORDER BY id DESC")
	if err != nil {
		log.Printf("getSlides failed:%v", err)
		return infos
	}
	defer rows.Close()

	for rows.Next() {
		var info hot.SlideInfo
		err := rows.Scan(&info.Image, &info.Target)
		if err != nil {
			log.Printf("getSlides scan failed:%v", err)
			continue
		}
		infos = append(infos, &info)
	}
	return infos
}

func (s *server) GetHotList(ctx context.Context, in *common.CommRequest) (*hot.HotListReply, error) {
	log.Printf("GetLatest uid:%d seq:%d, num:%d", in.Head.Uid, in.Seq, in.Num)
	var opening, running []*hot.BidInfo
	if in.Seq == 0 {
		opening = getOpeningSales(db, 4)
	}
	running = getRunningSales(db, in.Head.Uid, in.Num, in.Seq)
	reddot := 0
	if hasReddot(db, in.Head.Uid) {
		reddot = 1
	}
	marquee := getMarquee(db, marqueeInterval)
	slides := getSlides(db)
	promotion := getPromotion(db)
	return &hot.HotListReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid},
		Opening: opening, Slides: slides, Marquee: marquee, List: running,
		Promotion: &promotion, Reddot: int32(reddot)}, nil
}

func main() {
	lis, err := net.Listen("tcp", util.HotServerPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	db, err = util.InitDB(true)
	if err != nil {
		log.Fatalf("failed to init db connection: %v", err)
	}
	db.SetMaxIdleConns(util.MaxIdleConns)
	kv := util.InitRedis()
	go util.ReportHandler(kv, util.HotServerName, util.HotServerPort)

	s := grpc.NewServer()
	hot.RegisterHotServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
