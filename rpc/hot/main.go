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
	weatherDst      = "http://www.dg121.com/mobile"
	jokeTime        = 1483027200 // 2016-12-30
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

func getNews(db *sql.DB, seq, num, newsType int64) []*hot.HotsInfo {
	query := "SELECT id, title, img1, img2, img3, source, dst, ctime, stype FROM news WHERE deleted = 0 AND top = 0 AND stype = " +
		strconv.Itoa(int(newsType))
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
		err = rows.Scan(&info.Seq, &info.Title, &img[0], &info.Source, &info.Dst,
			&info.Ctime, &info.Play)
		if err != nil {
			log.Printf("scan rows failed: %v", err)
			return infos
		}
		info.Images = img[:1]
		info.Id = info.Seq
		infos = append(infos, &info)
	}
	return infos
}

func getJokes(db *sql.DB, seq, num int64) []*hot.HotsInfo {
	var infos []*hot.HotsInfo
	query := "SELECT id, content, dst, heart FROM joke "
	if seq != 0 {
		query += fmt.Sprintf(" WHERE id < %d", seq)
	}
	query += fmt.Sprintf(" ORDER BY id DESC LIMIT %d", num)

	rows, err := db.Query(query)
	if err != nil {
		log.Printf("getJokes query failed:%v", err)
		return infos
	}

	defer rows.Close()
	for rows.Next() {
		var info hot.HotsInfo
		err := rows.Scan(&info.Id, &info.Content, &info.Image, &info.Heart)
		if err != nil {
			log.Printf("getJokes scan failed:%v", err)
			continue
		}
		info.Seq = info.Id
		infos = append(infos, &info)
	}
	return infos
}

func calcJokeSeq() int64 {
	return (time.Now().Unix() - jokeTime) / hourSeconds * 100
}

func (s *server) GetHots(ctx context.Context, in *common.CommRequest) (*hot.HotsReply, error) {
	log.Printf("request uid:%d, sid:%s ctype:%d, seq:%d term:%d version:%d",
		in.Head.Uid, in.Head.Sid, in.Type, in.Seq, in.Head.Term, in.Head.Version)
	var infos []*hot.HotsInfo
	if in.Type == typeHotNews {
		if util.CheckTermVersion(in.Head.Term, in.Head.Version) {
			infos = getHotNews(db, in.Seq, util.MaxListSize)
			if in.Seq == 0 {
				max := getMaxNewsSeq(db)
				tops := getTopNews(db, 0)
				for i := 0; i < len(tops); i++ {
					infos[i].Seq += max
				}
				infos = append(tops, infos...)
			}
		} else {
			if in.Seq == 0 {
				max := getMaxNewsSeq(db)
				tops := getTopNews(db, 0)
				for i := 0; i < len(tops); i++ {
					infos[i].Seq += max
				}
				infos = getNews(db, in.Seq, util.MaxListSize/2, 10)
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
	} else if in.Type == typeDgNews {
		infos = getNews(db, in.Seq, util.MaxListSize, 10)
	} else if in.Type == typeAmuse {
		infos = getNews(db, in.Seq, util.MaxListSize, 4)
	} else if in.Type == typeJoke {
		seq := in.Seq
		if in.Seq == 0 {
			seq = calcJokeSeq()
		}
		infos = getJokes(db, seq, util.MaxListSize)
	}
	return &hot.HotsReply{
		Head:  &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid},
		Infos: infos}, nil
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

func getService(db *sql.DB) ([]*hot.ServiceCategory, error) {
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
	weather, err := getWeather(db)
	if err != nil {
		log.Printf("getWeather failed:%v", err)
		return &hot.WeatherNewsReply{Head: &common.Head{Retcode: 1}}, err
	}

	infos := getNews(db, 0, homeNewsNum, 10)
	infos = append(infos[:0], infos[1], infos[3], infos[5])
	notice := getNotice(db)
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
	uinfo, err := getUseInfo(db, in.Head.Uid)
	if err != nil {
		log.Printf("getUseInfo failed:%v", err)
		return &hot.FrontReply{Head: &common.Head{Retcode: 1}}, err
	}

	uinfo.Save = int32(float64(uinfo.Save) * saveRate)
	flag := util.IsWhiteUser(db, in.Head.Uid, util.BannerWhiteType)
	binfos, err := getBanners(db, flag)
	if err != nil {
		log.Printf("getBannerInfo failed:%v", err)
		return &hot.FrontReply{Head: &common.Head{Retcode: 1}}, err
	}

	return &hot.FrontReply{
		Head: &common.Head{Retcode: 0}, User: &uinfo, Banner: binfos}, nil
}

func getOpenedSales(db *sql.DB, num int32, seq int64) []*common.BidInfo {
	var opened []*common.BidInfo
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
		var info common.BidInfo
		var award common.AwardInfo
		err := rows.Scan(&info.Bid, &info.Gid, &info.Period, &info.Title,
			&info.Start, &info.End, &info.Image, &info.Total, &award.Uid,
			&award.Awardcode, &award.Nickname, &info.Status, &info.Subtitle)
		if err != nil {
			log.Printf("getOpenedSales scan failed:%v", err)
			continue
		}
		info.Start *= 1000
		info.End *= 1000
		info.Seq = info.Bid
		log.Printf("bid:%d gid:%d", info.Bid, info.Gid)
		award.Num = util.GetSalesCount(db, info.Bid, award.Uid)
		info.Award = &award
		opened = append(opened, &info)
	}
	return opened
}

func getOpeningSales(db *sql.DB, num int32) []*common.BidInfo {
	var opening []*common.BidInfo
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
		var info common.BidInfo
		var end int64
		err = rows.Scan(&info.Bid, &info.Gid, &info.Period, &info.Title,
			&info.Start, &end, &info.Image, &info.Total, &info.Status,
			&info.Subtitle)
		if err != nil {
			log.Printf("getOpening scan failed:%v", err)
			continue
		}
		info.Start *= 1000
		info.Seq = info.Bid
		info.Rest = util.GetRemainSeconds(end)
		opening = append(opening, &info)
	}
	if len(opening) < int(num) {
		opened := getOpenedSales(db, num-int32(len(opening)), 0)
		opening = append(opening, opened...)
	}
	return opening
}

func (s *server) GetOpening(ctx context.Context, in *common.CommRequest) (*hot.OpeningReply, error) {
	log.Printf("GetOpening uid:%d seq:%d, num:%d", in.Head.Uid, in.Seq, in.Num)
	opening := getOpeningSales(db, 0)
	var reddot int32
	if util.HasReddot(db, in.Head.Uid) {
		reddot = 1
	}
	return &hot.OpeningReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid},
		Opening: opening, Reddot: reddot}, nil
}

func (s *server) GetOpened(ctx context.Context, in *common.CommRequest) (*hot.OpenedReply, error) {
	log.Printf("GetOpened uid:%d seq:%d, num:%d", in.Head.Uid, in.Seq, in.Num)
	opened := getOpenedSales(db, in.Num, in.Seq)
	return &hot.OpenedReply{Head: &common.Head{Retcode: 0},
		Opened: opened}, nil
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

func getRunningSales(db *sql.DB, uid int64, num int32, seq int64) []*common.BidInfo {
	var infos []*common.BidInfo
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
		var info common.BidInfo
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
	opening := getOpeningSales(db, 4)
	reddot := 0
	if util.HasReddot(db, in.Head.Uid) {
		reddot = 1
	}
	slides := getSlides(db)
	promotion := getPromotion(db)
	return &hot.HotListReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid},
		Opening: opening, Slides: slides,
		Promotion: &promotion, Reddot: int32(reddot)}, nil
}

func (s *server) GetMarquee(ctx context.Context, in *common.CommRequest) (*hot.MarqueeReply, error) {
	log.Printf("GetLatest uid:%d seq:%d, num:%d", in.Head.Uid, in.Seq, in.Num)
	marquee := getMarquee(db, marqueeInterval)
	return &hot.MarqueeReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid},
		Marquee: marquee}, nil
}

func (s *server) GetRunning(ctx context.Context, in *common.CommRequest) (*hot.RunningReply, error) {
	log.Printf("GetLatest uid:%d seq:%d, num:%d", in.Head.Uid, in.Seq, in.Num)
	running := getRunningSales(db, in.Head.Uid, in.Num, in.Seq)
	return &hot.RunningReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid},
		Running: running}, nil
}

func getAwardDetail(db *sql.DB, sid, uid, awardcode int64) common.AwardInfo {
	var award common.AwardInfo
	award.Uid = uid
	award.Awardcode = awardcode
	err := db.QueryRow("SELECT nickname, headurl FROM user WHERE uid = ?", uid).
		Scan(&award.Nickname, &award.Head)
	if err != nil {
		log.Printf("getAwardDetail failed:%v", err)
	}
	award.Num = util.GetSalesCount(db, sid, uid)
	award.Codes = util.GetSalesCodes(db, sid, uid)

	return award
}

func getSalesDetail(db *sql.DB, sid, uid int64) (common.BidInfo, common.AwardInfo) {
	var bet common.BidInfo
	var award common.AwardInfo
	query := `SELECT sid, num, status, s.gid, total, remain, 
		UNIX_TIMESTAMP(atime), title, win_uid, win_code, g.image, g.sub_title, 
		UNIX_TIMESTAMP(s.etime) FROM sales s, goods g WHERE s.gid = g.gid 
		AND s.sid = `
	query += strconv.Itoa(int(sid))
	var winuid, awardcode, rest int64
	err := db.QueryRow(query).Scan(&bet.Bid, &bet.Period, &bet.Status, &bet.Gid,
		&bet.Total, &bet.Remain, &bet.End, &bet.Title, &winuid, &awardcode,
		&bet.Image, &bet.Subtitle, &rest)
	if err != nil {
		log.Printf("getSalesDetail scan failed:%v", err)
		return bet, award
	}
	if bet.Status == 2 {
		bet.Rest = util.GetRemainSeconds(rest)
	} else {
		bet.End *= 1000
	}
	if bet.Status >= 3 {
		award = getAwardDetail(db, sid, winuid, awardcode)
	}
	return bet, award
}

func getNextSale(db *sql.DB, gid int64) hot.NextInfo {
	var next hot.NextInfo
	err := db.QueryRow("SELECT sid, num FROM sales WHERE status = 1 AND gid = ?", gid).
		Scan(&next.Bid, &next.Period)
	if err != nil {
		log.Printf("getNextSale failed:%v", err)
		return next
	}
	return next
}

func getGoodsImages(db *sql.DB, gid int64) []string {
	var images []string
	rows, err := db.Query("SELECT image FROM goods_image WHERE type = 0 AND deleted = 0 AND gid = ?",
		gid)
	if err != nil {
		log.Printf("getGoodsImages query failed gid:%d %v", gid, err)
		return images
	}
	defer rows.Close()
	for rows.Next() {
		var img string
		err := rows.Scan(&img)
		if err != nil {
			log.Printf("getGoodsImages scan failed:%v", err)
			continue
		}
		images = append(images, img)
	}
	return images
}

func getUserJoin(db *sql.DB, uid, sid int64) hot.JoinInfo {
	var info hot.JoinInfo
	info.Uid = uid
	codes := util.GetSalesCodes(db, sid, uid)
	if len(codes) > 0 {
		info.Join = 1
		info.Codes = codes
	}
	return info
}

func getRunningGoodsSid(db *sql.DB, gid int64) int64 {
	var bid int64
	err := db.QueryRow("SELECT sid FROM sales WHERE status = 1 AND gid = ? ORDER BY sid DESC LIMIT 1",
		gid).Scan(&bid)
	if err != nil {
		log.Printf("getRunningGoodsSid failed gid:%d %v", gid, err)
	}
	return bid
}

func (s *server) GetDetail(ctx context.Context, in *hot.DetailRequest) (*hot.DetailReply, error) {
	log.Printf("GetLatest uid:%d bid:%d, gid:%d", in.Head.Uid, in.Bid, in.Gid)
	bid := in.Bid
	if in.Bid == 0 && in.Gid != 0 {
		bid = getRunningGoodsSid(db, in.Gid)
	}
	if bid == 0 {
		return &hot.DetailReply{Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}},
			nil
	}
	bet, award := getSalesDetail(db, bid, in.Head.Uid)
	if bet.Gid == 0 {
		log.Printf("getSalesDetail failed, bid:%d uid:%d", bid, in.Head.Uid)
		return &hot.DetailReply{Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}},
			nil
	}
	var next hot.NextInfo
	if bet.Status > 1 {
		next = getNextSale(db, bet.Gid)
	}
	slides := getGoodsImages(db, bet.Gid)
	join := getUserJoin(db, in.Head.Uid, in.Bid)
	return &hot.DetailReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid},
		Bet:  &bet, Award: &award, Next: &next, Slides: slides, Mine: &join}, nil
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
