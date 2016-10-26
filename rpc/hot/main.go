package main

import (
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

func (s *server) GetHots(ctx context.Context, in *hot.HotsRequest) (*hot.HotsReply, error) {
	db, err := util.InitDB()
	if err != nil {
		log.Printf("connect mysql failed:%v", err)
		return &hot.HotsReply{Head: &common.Head{Retcode: 1}}, err
	}
	log.Printf("request uid:%d, sid:%s ctype:%d, seq:%d", in.Head.Uid, in.Head.Sid, in.Type, in.Seq)
	var table string
	if in.Type == 0 {
		table = "news"
	} else {
		table = "video"
	}
	query := "SELECT id, title, img1, img2, img3, vid, source, dst, ctime FROM " + table + " WHERE 1 = 1 "
	if in.Seq != 0 {
		query += " AND id < " + strconv.Itoa(int(in.Seq))
	}
	query += " ORDER BY id DESC LIMIT " + strconv.Itoa(util.MaxListSize)
	log.Printf("query string:%s", query)
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("query failed:%v", err)
		return &hot.HotsReply{Head: &common.Head{Retcode: 1}}, err
	}

	infos := make([]*hot.HotsInfo, 20)
	i := 0
	for rows.Next() {
		var img [3]string
		var info hot.HotsInfo
		err = rows.Scan(&info.Seq, &info.Title, &img[0], &img[1], &img[2], &info.Video, &info.Source, &info.Dst, &info.Ctime)
		if err != nil {
			log.Printf("scan rows failed: %v", err)
			return &hot.HotsReply{Head: &common.Head{Retcode: 1}}, err
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
		infos[i] = &info
		i++
		log.Printf("title:%s source:%s", info.Title, info.Source)
	}
	return &hot.HotsReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid, Sid: in.Head.Sid}, Infos: infos[:i]}, nil
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
