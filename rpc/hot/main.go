package main

import (
	"database/sql"
	"log"
	"net"
	"strconv"

	common "../../proto/common"
	hot "../../proto/hot"
	util "../../util"
	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const (
	homeNewsNum = 6
	saveRate    = 0.1 / (1024.0 * 1024.0)
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

func getDgNews(db *sql.DB, seq, num int32) []*hot.HotsInfo {
	return getNews(db, seq, num, true)
}

func getHotNews(db *sql.DB, seq, num int32) []*hot.HotsInfo {
	return getNews(db, seq, num, false)
}

func getNews(db *sql.DB, seq, num int32, isDgNews bool) []*hot.HotsInfo {
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

func getVideos(db *sql.DB, seq int32) []*hot.HotsInfo {
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
		infos = append(infos, &info)
		log.Printf("title:%s source:%s", info.Title, info.Source)
	}
	return infos
}

func (s *server) GetHots(ctx context.Context, in *hot.HotsRequest) (*hot.HotsReply, error) {
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

func (s *server) GetServices(ctx context.Context, in *hot.ServiceRequest) (*hot.ServiceReply, error) {
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

func (s *server) GetWeatherNews(ctx context.Context, in *hot.HotsRequest) (*hot.WeatherNewsReply, error) {
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

func (s *server) GetFrontInfo(ctx context.Context, in *hot.HotsRequest) (*hot.FrontReply, error) {
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
