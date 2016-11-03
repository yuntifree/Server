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
	port = ":50053"
)

type server struct{}

func getNews(db *sql.DB, seq int32) []*hot.HotsInfo {
	var infos []*hot.HotsInfo
	query := "SELECT id, title, img1, img2, img3, source, dst, ctime FROM news WHERE deleted = 0 "
	if seq != 0 {
		query += " AND id < " + strconv.Itoa(int(seq))
	}
	query += " ORDER BY id DESC LIMIT " + strconv.Itoa(util.MaxListSize)
	log.Printf("query string:%s", query)
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("query failed:%v", err)
		return infos
	}

	for rows.Next() {
		var img [3]string
		var info hot.HotsInfo
		err = rows.Scan(&info.Seq, &info.Title, &img[0], &img[1], &img[2], &info.Source, &info.Dst, &info.Ctime)
		if err != nil {
			log.Printf("scan rows failed: %v", err)
			return infos
		}
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
	query := "SELECT vid, title, img, source, dst, ctime FROM youku_video WHERE 1 = 1 "
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

	for rows.Next() {
		var img [3]string
		var info hot.HotsInfo
		err = rows.Scan(&info.Seq, &info.Title, &img[0], &info.Source, &info.Dst, &info.Ctime)
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
	db, err := util.InitDB(true)
	if err != nil {
		log.Printf("connect mysql failed:%v", err)
		return &hot.HotsReply{Head: &common.Head{Retcode: 1}}, err
	}
	defer db.Close()
	log.Printf("request uid:%d, sid:%s ctype:%d, seq:%d", in.Head.Uid, in.Head.Sid, in.Type, in.Seq)
	var infos []*hot.HotsInfo
	if in.Type == 0 {
		infos = getNews(db, in.Seq)
	} else {
		infos = getVideos(db, in.Seq)
	}
	return &hot.HotsReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid}, Infos: infos}, nil
}

func getTops(db *sql.DB) ([]*hot.ServiceInfo, error) {
	var infos []*hot.ServiceInfo
	rows, err := db.Query("SELECT title, icon, dst FROM service WHERE category = 0")
	if err != nil {
		log.Printf("query failed:%v", err)
		return infos, err
	}

	for rows.Next() {
		var info hot.ServiceInfo
		err := rows.Scan(&info.Title, &info.Icon, &info.Dst)
		if err != nil {
			continue
		}
		infos = append(infos, &info)
	}

	return infos, nil
}

func getCategoryTitle(category int) string {
	switch category {
	default:
		return "智慧政务"
	case 2:
		return "交通出行"
	case 3:
		return "医疗服务"
	case 4:
		return "网上充值"
	}
}

func getService(db *sql.DB) ([]*hot.ServiceCategory, error) {
	var infos []*hot.ServiceCategory
	rows, err := db.Query("SELECT title, icon, dst, category FROM service WHERE category != 0 ORDER BY category")
	if err != nil {
		log.Printf("query failed:%v", err)
		return infos, err
	}

	category := 0
	var srvs []*hot.ServiceInfo
	for rows.Next() {
		var info hot.ServiceInfo
		var cate int
		err := rows.Scan(&info.Title, &info.Icon, &info.Dst, &cate)
		if err != nil {
			continue
		}

		if cate != category {
			if len(srvs) > 0 {
				var cateinfo hot.ServiceCategory
				cateinfo.Title = getCategoryTitle(category)
				cateinfo.Infos = srvs[:]
				infos = append(infos, &cateinfo)
				srvs = srvs[len(srvs):]
			}
			category = cate
		}
		srvs = append(srvs, &info)
	}

	if len(srvs) > 0 {
		var cateinfo hot.ServiceCategory
		cateinfo.Title = getCategoryTitle(category)
		cateinfo.Infos = srvs[:]
		infos = append(infos, &cateinfo)
	}

	return infos, nil
}

func (s *server) GetServices(ctx context.Context, in *hot.ServiceRequest) (*hot.ServiceReply, error) {
	db, err := util.InitDB(true)
	if err != nil {
		log.Printf("connect mysql failed:%v", err)
		return &hot.ServiceReply{Head: &common.Head{Retcode: 1}}, err
	}
	defer db.Close()
	infos, err := getTops(db)
	if err != nil {
		log.Printf("getTops failed:%v", err)
		return &hot.ServiceReply{Head: &common.Head{Retcode: 1}}, err
	}

	categories, err := getService(db)
	if err != nil {
		log.Printf("getServie failed:%v", err)
		return &hot.ServiceReply{Head: &common.Head{Retcode: 1}}, err
	}

	return &hot.ServiceReply{Head: &common.Head{Retcode: 0}, Tops: infos, Services: categories}, nil
}

func main() {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	hot.RegisterHotServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
