package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"Server/util"

	"Server/proto/common"
	"Server/proto/modify"
	"Server/zte"

	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/net/context"
)

const (
	feedInterval = 60
)

const (
	videoClickType = iota
	newsClickType
	adShowType
	adClickType
	serviceClickType
	jokeHeartType
	downloadType
	tabSwitchType
	portalServiceType
	jokeBadType
	bannerClickType
	recommendClickType
	urbanServiceClickType
	hospitalIntroType
	hospitalServiceType
	onlineServiceType
	educationVideoType
)

type server struct{}

var db *sql.DB

func (s *server) ReviewNews(ctx context.Context, in *modify.NewsRequest) (*common.CommReply, error) {
	if in.Reject {
		db.Exec("UPDATE news SET review = 1, deleted = 1, rtime = NOW(), ruid = ? WHERE id = ?",
			in.Head.Uid, in.Id)
	} else {
		query := "UPDATE news SET review = 1, rtime = NOW(), deleted = 0, ruid = " +
			strconv.Itoa(int(in.Head.Uid))
		if in.Modify && in.Title != "" {
			query += ", title = '" + in.Title + "' "
		}
		query += " WHERE id = " + strconv.Itoa(int(in.Id))
		db.Exec(query)
		if len(in.Tags) > 0 {
			for i := 0; i < len(in.Tags); i++ {
				db.Exec("INSERT INTO news_tags(nid, tid, ruid, ctime) VALUES (?, ?, ?, NOW())",
					in.Id, in.Tags[i], in.Head.Uid)
			}
		}
	}

	return &common.CommReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
}

func (s *server) ReviewVideo(ctx context.Context, in *modify.VideoRequest) (*common.CommReply, error) {
	if in.Reject {
		db.Exec("UPDATE youku_video SET review = 1, deleted = 1, rtime = NOW(), ruid = ? WHERE vid = ?",
			in.Head.Uid, in.Id)
	} else {
		query := "UPDATE youku_video SET review = 1, rtime = NOW(), ruid = " +
			strconv.Itoa(int(in.Head.Uid))
		if in.Modify && in.Title != "" {
			query += ", title = '" + in.Title + "' "
		}
		query += " WHERE vid = " + strconv.Itoa(int(in.Id))
		db.Exec(query)
	}

	return &common.CommReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
}

func (s *server) AddTemplate(ctx context.Context, in *modify.AddTempRequest) (*common.CommReply, error) {
	res, err := db.Exec("INSERT INTO template(title, content, ruid, ctime, mtime) VALUES (?, ?, ?, NOW(), NOW())",
		in.Info.Title, in.Info.Content, in.Head.Uid)
	if err != nil {
		log.Printf("query failed:%v", err)
		return &common.CommReply{Head: &common.Head{Retcode: 1}}, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		log.Printf("query failed:%v", err)
		return &common.CommReply{Head: &common.Head{Retcode: 1}}, err
	}

	return &common.CommReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}, Id: id}, nil
}

func (s *server) AddWifi(ctx context.Context, in *modify.WifiRequest) (*common.CommReply, error) {
	_, err := db.Exec("INSERT INTO wifi(ssid, password, longitude, latitude, uid, ctime) VALUES (?, ?, ?, ?,?, NOW())",
		in.Info.Ssid, in.Info.Password, in.Info.Longitude, in.Info.Latitude, in.Head.Uid)
	if err != nil {
		log.Printf("query failed:%v", err)
		return &common.CommReply{Head: &common.Head{Retcode: 1}}, err
	}

	return &common.CommReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
}

func (s *server) ModTemplate(ctx context.Context, in *modify.ModTempRequest) (*common.CommReply, error) {
	query := "UPDATE template SET "
	if in.Info.Title != "" {
		query += " title = '" + in.Info.Title + "', "
	}
	if in.Info.Content != "" {
		query += " content = '" + in.Info.Content + "', "
	}
	online := 0
	if in.Info.Online {
		online = 1
	}
	query += " mtime = NOW(), ruid = " + strconv.Itoa(int(in.Head.Uid)) +
		", online = " + strconv.Itoa(online) + " WHERE id = " +
		strconv.Itoa(int(in.Info.Id))
	_, err := db.Exec(query)

	if err != nil {
		log.Printf("query failed:%v", err)
		return &common.CommReply{Head: &common.Head{Retcode: 1}}, err
	}

	return &common.CommReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
}

func (s *server) ReportClick(ctx context.Context, in *modify.ClickRequest) (*common.CommReply, error) {
	log.Printf("ReportClick uid:%d type:%d id:%d", in.Head.Uid, in.Type, in.Id)
	var res sql.Result
	var err error
	if in.Type != 4 {
		res, err = db.Exec("INSERT IGNORE INTO click_record(uid, type, id, ctime) VALUES(?, ?, ?, NOW())",
			in.Head.Uid, in.Type, in.Id)
	} else {
		res, err = db.Exec("INSERT INTO service_click_record(uid, sid, ctime) VALUES(?, ?, NOW())",
			in.Head.Uid, in.Id)
	}
	if err != nil {
		log.Printf("query failed:%v", err)
		return &common.CommReply{Head: &common.Head{Retcode: 1}}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Printf("get last insert id failed:%v", err)
		return &common.CommReply{Head: &common.Head{Retcode: 1}}, err
	}

	if id != 0 {
		switch in.Type {
		case videoClickType:
			_, err = db.Exec("UPDATE youku_video SET play = play + 1 WHERE vid = ?", in.Id)
		case newsClickType:
			_, err = db.Exec("UPDATE news SET click = click + 1 WHERE id = ?", in.Id)
		case adShowType:
			_, err = db.Exec("UPDATE ads SET display = display + 1 WHERE id = ?", in.Id)
		case adClickType:
			_, err = db.Exec("UPDATE ads SET click = click + 1 WHERE id = ?", in.Id)
		case serviceClickType:
			_, err = db.Exec("INSERT INTO service_click(sid, click, ctime) VALUES (?, 1, CURDATE()) ON DUPLICATE KEY UPDATE click = click + 1", in.Id)
		case jokeHeartType:
			_, err = db.Exec("UPDATE joke SET heart = heart + 1 WHERE id = ?", in.Id)
		case downloadType, tabSwitchType, portalServiceType:
			name := in.Name
			if in.Type == downloadType {
				name = "portal"
			}
			_, err = db.Exec("INSERT INTO click_stat(type, name, ctime, total) VALUES (?,?,NOW(),1) ON DUPLICATE KEY UPDATE total = total + 1",
				in.Type, name)
		case jokeBadType:
			_, err = db.Exec("UPDATE joke SET bad = bad + 1 WHERE id = ?", in.Id)
		case bannerClickType:
			_, err = db.Exec("UPDATE banner SET click = click + 1 WHERE id = ?", in.Id)
		case recommendClickType:
			_, err = db.Exec("UPDATE recommend SET click = click + 1 WHERE id = ?", in.Id)
		case urbanServiceClickType:
			_, err = db.Exec("UPDATE urban_service SET click = click + 1 WHERE id = ?", in.Id)
		case educationVideoType:
			_, err = db.Exec("UPDATE education_video SET click = click + 1 WHERE id = ?",
				in.Id)
		default:
			log.Printf("illegal type:%d, id:%d uid:%d", in.Type, in.Id, in.Head.Uid)

		}
		if err != nil {
			log.Printf("update click count failed type:%d id:%d:%v", in.Type, in.Id, err)
			return &common.CommReply{Head: &common.Head{Retcode: 1}}, err
		}
	}

	return &common.CommReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
}

func (s *server) ReportApmac(ctx context.Context, in *modify.ApmacRequest) (*common.CommReply, error) {
	util.RefreshUserAp(db, in.Head.Uid, in.Apmac)
	return &common.CommReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
}

func (s *server) AddImage(ctx context.Context, in *modify.AddImageRequest) (*common.CommReply, error) {
	for i := 0; i < len(in.Fnames); i++ {
		_, err := db.Exec("INSERT IGNORE INTO image(uid, name, ctime) VALUES(?, ?, NOW())",
			in.Head.Uid, in.Fnames[i])
		if err != nil {
			log.Printf("insert into image failed uid:%d name:%s err:%v\n",
				in.Head.Uid, in.Fnames[i], err)
		}
	}
	return &common.CommReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
}

func (s *server) FinImage(ctx context.Context, in *modify.ImageRequest) (*common.CommReply, error) {
	_, err := db.Exec("UPDATE image SET filesize = ?, height = ?, width = ?, ftime = NOW(), status = 1 WHERE name = ?",
		in.Info.Size, in.Info.Height, in.Info.Width, in.Info.Name)
	if err != nil {
		log.Printf("update image failed name:%s err:%v\n", in.Info.Name, err)
	}
	return &common.CommReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
}

func (s *server) AddBanner(ctx context.Context, in *modify.BannerRequest) (*common.CommReply, error) {
	res, err := db.Exec("INSERT INTO banner(img, dst, priority, title, type, ctime, etime, dbg) VALUES(?, ?, ?, ?, ?, NOW(), ?, ?)",
		in.Info.Img, in.Info.Dst, in.Info.Priority, in.Info.Title, in.Info.Type,
		in.Info.Expire, in.Info.Dbg)
	if err != nil {
		log.Printf("insert into banner failed img:%s dst:%s err:%v\n",
			in.Info.Img, in.Info.Dst, err)
		return &common.CommReply{Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Printf("AddBanner get LastInsertId failed:%v\n", err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	return &common.CommReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}, Id: id}, nil
}

func (s *server) ModBanner(ctx context.Context, in *modify.BannerRequest) (*common.CommReply, error) {
	log.Printf("ModBanner info:%v", in.Info)
	query := fmt.Sprintf("UPDATE banner SET priority = %d, online = %d, deleted = %d, dbg = %d ",
		in.Info.Priority, in.Info.Online, in.Info.Deleted, in.Info.Dbg)
	if in.Info.Img != "" {
		query += ", img = '" + in.Info.Img + "' "
	}
	if in.Info.Dst != "" {
		query += ", dst = '" + in.Info.Dst + "' "
	}
	if in.Info.Title != "" {
		query += ", title = '" + in.Info.Title + "' "
	}
	if in.Info.Expire != "" {
		query += ", etime = '" + in.Info.Expire + "' "
	}
	query += fmt.Sprintf(" WHERE id = %d", in.Info.Id)
	_, err := db.Exec(query)
	if err != nil {
		log.Printf("insert into banner failed img:%s dst:%s err:%v\n",
			in.Info.Img, in.Info.Dst, err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	return &common.CommReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
}

func (s *server) AddTags(ctx context.Context, in *modify.AddTagRequest) (*modify.AddTagReply, error) {
	var ids []int64
	for i := 0; i < len(in.Tags); i++ {
		res, err := db.Exec("INSERT INTO tags(content, ctime) VALUES (?, NOW())", in.Tags[i])
		if err != nil {
			log.Printf("add tag failed tag:%s err:%v\n", in.Tags[i], err)
			continue
		}
		id, err := res.LastInsertId()
		if err != nil {
			log.Printf("get tag insert id failed:%v", err)
			continue
		}
		ids = append(ids, id)
	}
	return &modify.AddTagReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}, Ids: ids}, nil
}

func genIDStr(ids []int64) string {
	var str string
	for i := 0; i < len(ids); i++ {
		str += strconv.Itoa(int(ids[i]))
		if i < len(ids)-1 {
			str += ","
		}
	}
	return str
}

func (s *server) DelTags(ctx context.Context, in *modify.DelTagRequest) (*common.CommReply, error) {
	str := genIDStr(in.Ids)
	query := "UPDATE tags SET deleted = 1 WHERE id IN (" + str + ")"
	_, err := db.Exec(query)
	if err != nil {
		log.Printf("DelTags failed:%v", err)
	}
	return &common.CommReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
}

func genConfStr(names []string) string {
	var str string
	for i := 0; i < len(names); i++ {
		str += "'" + names[i] + "'"
		if i < len(names)-1 {
			str += ","
		}
	}
	return str
}

func (s *server) DelConf(ctx context.Context, in *modify.DelConfRequest) (*common.CommReply, error) {
	str := genConfStr(in.Names)
	query := "UPDATE kv_config SET deleted = 1 WHERE name IN (" + str + ")"
	_, err := db.Exec(query)
	if err != nil {
		log.Printf("DelTags failed:%v", err)
	}
	return &common.CommReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
}

func (s *server) AddConf(ctx context.Context, in *modify.ConfRequest) (*common.CommReply, error) {
	log.Printf("AddConf uid:%d key:%s", in.Head.Uid, in.Info.Key)
	_, err := db.Exec("INSERT INTO kv_config(name, val, ctime) VALUES (?, ?, NOW()) ON DUPLICATE KEY UPDATE val = ?",
		in.Info.Key, in.Info.Val, in.Info.Val)
	if err != nil {
		log.Printf("add config failed uid:%d name:%s err:%v\n", in.Head.Uid,
			in.Info.Key, err)
		return &common.CommReply{
				Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}},
			errors.New("add conf failed")
	}
	return &common.CommReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
}

func (s *server) AddAdBan(ctx context.Context, in *modify.AddBanRequest) (*common.CommReply, error) {
	log.Printf("AddAdBan uid:%d term:%s version", in.Head.Uid, in.Info.Term,
		in.Info.Version)
	res, err := db.Exec("INSERT INTO ad_ban(term, version, ctime) VALUES (?, ?, NOW()) ON DUPLICATE KEY UPDATE deleted = 0",
		in.Info.Term, in.Info.Version)
	if err != nil {
		log.Printf("add adban failed uid:%d term:%d version:%d err:%v\n",
			in.Head.Uid, in.Info.Term, in.Info.Version, err)
		return &common.CommReply{
				Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}},
			errors.New("add adban failed")
	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Printf("add adban get insert id failed uid:%d term:%d version:%d err:%v\n",
			in.Head.Uid, in.Info.Term, in.Info.Version, err)
		return &common.CommReply{
				Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}},
			errors.New("add adban failed")
	}
	return &common.CommReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}, Id: id}, nil
}

func (s *server) DelAdBan(ctx context.Context, in *modify.DelBanRequest) (*common.CommReply, error) {
	log.Printf("DelAdBan uid:%d", in.Head.Uid)
	idStr := genIDStr(in.Ids)
	query := fmt.Sprintf("UPDATE ad_ban SET deleted = 1 WHERE id IN (%s)", idStr)
	log.Printf("query :%s", query)
	_, err := db.Exec(query)
	if err != nil {
		log.Printf("DelAdBan query failed:%v", err)
	}
	return &common.CommReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
}

func (s *server) AddWhiteList(ctx context.Context, in *modify.WhiteRequest) (*common.CommReply, error) {
	for _, v := range in.Ids {
		_, err := db.Exec("INSERT INTO white_list(type, uid, ctime) VALUES (?, ?, NOW()) ON DUPLICATE KEY UPDATE deleted = 0", in.Type, v)
		if err != nil {
			log.Printf("AddWhiteList insert failed uid:%d %v", v, err)
			continue
		}
	}
	return &common.CommReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
}

func (s *server) DelWhiteList(ctx context.Context, in *modify.WhiteRequest) (*common.CommReply, error) {
	idStr := genIDStr(in.Ids)
	query := fmt.Sprintf("UPDATE white_list SET deleted = 1 WHERE type = 0 AND uid IN (%s)", idStr)
	log.Printf("DelWhiteList query:%s", query)
	_, err := db.Exec(query)
	if err != nil {
		log.Printf("DelWhiteList query failed:%v", err)
	}
	return &common.CommReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
}

func (s *server) AddFeedback(ctx context.Context, in *modify.FeedRequest) (*common.CommReply, error) {
	var last int64
	db.QueryRow("SELECT UNIX_TIMESTAMP(ctime) FROM feedback WHERE uid = ? ORDER BY id DESC LIMIT 1",
		in.Head.Uid).Scan(&last)
	if time.Now().Unix() > last+feedInterval {
		db.Exec("INSERT INTO feedback(uid, content, contact, ctime) VALUES(?, ?, ?, NOW())",
			in.Head.Uid, in.Content, in.Contact)
	} else {
		log.Printf("frequency exceed limit uid:%d", in.Head.Uid)
	}
	return &common.CommReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
}

func recordPurchaseAttempt(db *sql.DB, uid, sid, num int64) {
	_, err := db.Exec("INSERT INTO purchase_attempt_history(uid, sid, num, ctime) VALUES (?, ?, ?, NOW())",
		uid, sid, num)
	if err != nil {
		log.Printf("recordPurchaseAttempt failed, uid:%d sid:%d num:%d", uid,
			sid, num)
	}
}

func recordPurchase(db *sql.DB, uid, sid, num int64) int64 {
	res, err := db.Exec("INSERT INTO purchase_history(uid, sid, num, ctime) VALUES(?, ?, ?, NOW())",
		uid, sid, num)
	if err != nil {
		log.Printf("recordPurchase query failed, uid:%d sid:%d num:%d", uid,
			sid, num)
		return 0
	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Printf("recordPurchase query failed, uid:%d sid:%d num:%d", uid,
			sid, num)
		return 0
	}
	return id
}

func ackPurchaseFlag(db *sql.DB, hid int64) {
	_, err := db.Exec("UPDATE purchase_history SET ack_flag = 1 WHERE hid = ?", hid)
	if err != nil {
		log.Printf("ackPurchaseFlag failed hid:%d %v", hid, err)
	}
}

func getPurchaseCode(db *sql.DB, hid int64) int64 {
	var code int64
	err := db.QueryRow("SELECT num FROM sales_history WHERE hid = ?", hid).
		Scan(&code)
	if err != nil {
		log.Printf("getPurchaseCode query failed:%d %v", hid, err)
	}
	return code
}

func getRemainSales(db *sql.DB, sid int64) int64 {
	var remain int64
	err := db.QueryRow("SELECT remain FROM sales WHERE sid = ?", sid).
		Scan(&remain)
	if err != nil {
		log.Printf("getRemainSales query failed:%d %v", sid, err)
	}
	return remain
}

func (s *server) DelZteAccount(ctx context.Context, in *modify.ZteRequest) (*common.CommReply, error) {
	log.Printf("DelZteAccount uid:%d account:%s", in.Head.Uid, in.Phone)
	if !zte.Remove(in.Phone, zte.SshType) {
		log.Printf("DelZteAccount failed, account:%s", in.Phone)
		return &common.CommReply{
			Head: &common.Head{Retcode: common.ErrCode_ZTE_REMOVE,
				Uid: in.Head.Uid}}, nil
	}
	return &common.CommReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
}

func (s *server) AddPortalDir(ctx context.Context, in *modify.PortalDirRequest) (*common.CommReply, error) {
	log.Printf("AddPortalDir info:%v", in.Info)
	res, err := db.Exec("INSERT INTO portal_page(type, dir, description, ctime) VALUES (?,?,?, NOW())", in.Info.Type, in.Info.Dir, in.Info.Description)
	if err != nil {
		log.Printf("AddPortalDir query failed:%v", err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil

	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Printf("AddPortalDir get id failed:%v", err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	return &common.CommReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}, Id: id}, nil
}

func getPortalDirType(db *sql.DB, id int64) int64 {
	var ptype int64
	err := db.QueryRow("SELECT type FROM portal_page WHERE id = ?", id).Scan(&ptype)
	if err != nil {
		log.Printf("getPortalDirType query failed:%v", err)
	}
	return ptype
}

func (s *server) OnlinePortalDir(ctx context.Context, in *common.CommRequest) (*common.CommReply, error) {
	log.Printf("AddPortalDir info:%v", in)
	ptype := getPortalDirType(db, in.Id)
	_, err := db.Exec("UPDATE portal_page SET online = 1 WHERE id = ?", in.Id)
	if err != nil {
		log.Printf("OnlinePortalDir query update failed:%v", err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil

	}
	_, err = db.Exec("UPDATE portal_page SET online = 0 WHERE id != ? AND type = ?",
		in.Id, ptype)
	if err != nil {
		log.Printf("OnlinePortalDir drop failed:%v", err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	return &common.CommReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
}

func (s *server) AddChannelVersion(ctx context.Context, in *modify.ChannelVersionRequest) (*common.CommReply, error) {
	log.Printf("AddChannelVersion info:%v", in.Info)
	res, err := db.Exec("INSERT IGNORE INTO app_channel(channel, cname, version, vname, downurl, ctime) VALUES (?,?,?,?,?, NOW())",
		in.Info.Channel, in.Info.Cname, in.Info.Version, in.Info.Vname, in.Info.Downurl)
	if err != nil {
		log.Printf("AddChannelVersion query failed:%v", err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil

	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Printf("AddChannelVersion get id failed:%v", err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	return &common.CommReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}, Id: id}, nil
}

func (s *server) ModChannelVersion(ctx context.Context, in *modify.ChannelVersionRequest) (*common.CommReply, error) {
	log.Printf("ModChannelVersion info:%v", in.Info)
	query := fmt.Sprintf("UPDATE app_channel SET version = %d, vname = '%s'", in.Info.Version, in.Info.Vname)
	if in.Info.Channel != "" {
		query += ", channel = '" + in.Info.Channel + "' "
	}
	if in.Info.Cname != "" {
		query += ", cname = '" + in.Info.Cname + "' "
	}
	if in.Info.Downurl != "" {
		query += ", downurl = '" + in.Info.Downurl + "' "
	}
	query += fmt.Sprintf(" WHERE id = %d", in.Info.Id)
	log.Printf("ModChannelVersion query:%s", query)
	_, err := db.Exec(query)
	if err != nil {
		log.Printf("ModChannelVersion query failed:%v", err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil

	}
	return &common.CommReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
}

func recordIssue(db *sql.DB, ids string) {
	arr := strings.Split(ids, ",")
	for i := 0; i < len(arr); i++ {
		id, err := strconv.Atoi(arr[i])
		if err != nil {
			log.Printf("recordIssue failed:%d %s %v", i, arr[i], err)
			continue
		}
		db.Exec("UPDATE issue SET times = times + 1 WHERE id = ?", id)
	}
}

func (s *server) ReportIssue(ctx context.Context, in *modify.IssueRequest) (*common.CommReply, error) {
	log.Printf("ReportIssue request:%v", in)
	_, err := db.Exec("INSERT INTO issue_record(acname, usermac, apmac, contact, content, ids, ctime) VALUES (?, ?, ?, ?, ?, ?, NOW())",
		in.Acname, in.Usermac, in.Apmac, in.Contact, in.Content,
		in.Ids)
	if err != nil {
		log.Printf("ReportIssue query failed:%v", err)
	}
	recordIssue(db, in.Ids)
	return &common.CommReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
}

func main() {
	lis, err := net.Listen("tcp", util.ModifyServerPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	db, err = util.InitDB(false)
	if err != nil {
		log.Fatalf("failed to init db connection:%v", err)
	}
	db.SetMaxIdleConns(util.MaxIdleConns)

	kv := util.InitRedis()
	go util.ReportHandler(kv, util.ModifyServerName, util.ModifyServerPort)
	//cli := util.InitEtcdCli()
	//go util.ReportEtcd(cli, util.ModifyServerName, util.ModifyServerPort)

	s := util.NewGrpcServer()
	modify.RegisterModifyServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
