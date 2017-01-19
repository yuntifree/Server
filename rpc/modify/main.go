package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"

	"../../util"

	common "../../proto/common"
	modify "../../proto/modify"
	zte "../../zte"
	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
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
)

type server struct{}

var db *sql.DB

func (s *server) ReviewNews(ctx context.Context, in *modify.NewsRequest) (*common.CommReply, error) {
	if in.Reject {
		db.Exec("UPDATE news SET review = 1, deleted = 1, rtime = NOW(), ruid = ? WHERE id = ?",
			in.Head.Uid, in.Id)
	} else {
		query := "UPDATE news SET review = 1, rtime = NOW(), ruid = " +
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
	res, err := db.Exec("INSERT INTO banner(img, dst, priority, title, type, ctime, etime) VALUES(?, ?, ?, ?, ?, NOW(), ?)",
		in.Info.Img, in.Info.Dst, in.Info.Priority, in.Info.Title, in.Info.Type,
		in.Info.Expire)
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
	var ids []int32
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
		ids = append(ids, int32(id))
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

func (s *server) AddAddress(ctx context.Context, in *modify.AddressRequest) (*common.CommReply, error) {
	log.Printf("AddAddress uid:%d detail:%s", in.Head.Uid, in.Info.Detail)
	res, err := db.Exec("INSERT INTO address(uid, consignee, phone, province, city, district, detail, zip, addr, ctime) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NOW())",
		in.Head.Uid, in.Info.User, in.Info.Mobile, in.Info.Province, in.Info.City,
		in.Info.Zone, in.Info.Detail, in.Info.Zip, in.Info.Addr)
	if err != nil {
		log.Printf("add address failed uid:%d detail:%s err:%v\n", in.Head.Uid,
			in.Info.Detail, err)
		return &common.CommReply{Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}},
			errors.New("add address failed")
	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Printf("add address get insert id failed:%v", err)
		return &common.CommReply{Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}},
			errors.New("add address failed")
	}
	if in.Info.Def {
		_, err = db.Exec("UPDATE user SET address = ? WHERE uid = ?", id, in.Head.Uid)
		if err != nil {
			log.Printf("update user address failed, uid:%d aid:%d", in.Head.Uid, id)
		}
	}
	return &common.CommReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}, Id: id}, nil
}

func (s *server) ModAddress(ctx context.Context, in *modify.AddressRequest) (*common.CommReply, error) {
	log.Printf("ModAddress uid:%d detail:%s", in.Head.Uid, in.Info.Detail)
	_, err := db.Exec("UPDATE address SET consignee = ?, phone = ?, province = ?, city = ?, district = ?, detail = ?, zip = ?, addr = ? WHERE uid = ? AND aid = ?",
		in.Info.User, in.Info.Mobile, in.Info.Province, in.Info.City, in.Info.Zone,
		in.Info.Detail, in.Info.Zip, in.Info.Addr, in.Head.Uid, in.Info.Aid)
	if err != nil {
		log.Printf("modify address failed uid:%d detail:%s err:%v\n", in.Head.Uid,
			in.Info.Detail, err)
		return &common.CommReply{
				Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}},
			errors.New("add address failed")
	}
	return &common.CommReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
}

func (s *server) DelAddress(ctx context.Context, in *modify.AddressRequest) (*common.CommReply, error) {
	log.Printf("DelAddress uid:%d aid:%d", in.Head.Uid, in.Info.Aid)
	_, err := db.Exec("UPDATE address SET deleted = 1 WHERE uid = ? AND aid = ?",
		in.Head.Uid, in.Info.Aid)
	if err != nil {
		log.Printf("del address failed uid:%d aid:%d err:%v\n", in.Head.Uid,
			in.Info.Aid, err)
		return &common.CommReply{
				Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}},
			errors.New("add address failed")
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

func (s *server) PurchaseSales(ctx context.Context, in *common.CommRequest) (*modify.PurchaseReply, error) {
	var info modify.PurchaseResult
	recordPurchaseAttempt(db, in.Head.Uid, in.Id, int64(in.Num))
	var phone string
	var balance int64
	err := db.QueryRow("SELECT phone, balance FROM user WHERE uid = ?",
		in.Head.Uid).Scan(&phone, &balance)
	if err != nil {
		log.Printf("PusrchaseSales query user info failed uid:%d %v",
			in.Head.Uid, err)
		return &modify.PurchaseReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, err
	}
	info.Phoneflag = phone == ""
	if balance < 100 {
		info.Nocoin = true
		return &modify.PurchaseReply{
			Head: &common.Head{Retcode: 0, Uid: in.Head.Uid},
			Info: &info}, nil
	}

	pid := recordPurchase(db, in.Head.Uid, in.Id, int64(in.Num))

	var ret, hid int64
	err = db.QueryRow("call purchase_sales(?, ?,100)", in.Id, in.Head.Uid).
		Scan(&ret, &hid)
	if ret == 2 {
		info.Nocoin = true
		return &modify.PurchaseReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid},
			Info: &info}, nil
	} else if ret == 3 {
		info.Notimes = true
		return &modify.PurchaseReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid},
			Info: &info}, nil
	}

	if hid == 0 {
		log.Printf("illegal hid, sid:%d uid:%d", in.Id, in.Head.Uid)
		return &modify.PurchaseReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid},
			Info: &info}, nil
	}
	ackPurchaseFlag(db, pid)
	info.Remain = getRemainSales(db, in.Id)
	info.Code = getPurchaseCode(db, hid)

	return &modify.PurchaseReply{Head: &common.Head{Retcode: 0, Uid: in.Head.Uid},
		Info: &info}, nil
}

func getSalesGid(db *sql.DB, sid int64) int64 {
	var id int64
	err := db.QueryRow("SELECT gid FROM sales WHERE sid = ?", sid).
		Scan(&id)
	if err != nil {
		log.Printf("getSalesGid failed:%v", err)
	}
	return id
}

func checkShare(db *sql.DB, uid, sid int64) bool {
	var status, euid, share int64
	err := db.QueryRow("SELECT status, uid, share FROM logistics WHERE sid = ?", sid).
		Scan(&status, &euid, &share)
	if err != nil {
		log.Printf("checkShare failed sid:%v", err)
		return false
	}
	if euid != uid || status != util.ReceiptStatus || share != 0 {
		return false
	}
	return true
}

func (s *server) AddShare(ctx context.Context, in *modify.ShareRequest) (*common.CommReply, error) {
	if !checkShare(db, in.Head.Uid, in.Bid) {
		log.Printf("AddShare check failed uid:%d bid:%d", in.Head.Uid, in.Bid)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	gid := getSalesGid(db, in.Bid)
	res, err := db.Exec("INSERT INTo share_history(sid, uid, gid, title, content, image_num, ctime) VALUES (?, ?, ?, ?, ?, ?, NOW())",
		in.Bid, in.Head.Uid, gid, in.Title, in.Text, len(in.Images))
	if err != nil {
		log.Printf("AddShare insert failed uid:%d bid:%d", in.Head.Uid, in.Bid)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	hid, err := res.LastInsertId()
	if err != nil {
		log.Printf("AddShare get last insert id failed uid:%d bid:%d", in.Head.Uid, in.Bid)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	for _, v := range in.Images {
		_, err := db.Exec("INSERT INTO share_image(sid, hid, url, ctime) VALUES (?, ?, ?, NOW())",
			in.Bid, hid, v)
		if err != nil {
			log.Printf("AddShare insert image failed:%v", err)
		}
	}
	_, err = db.Exec("UPDATE logistics SET share = 1 WHERE sid = ?", in.Bid)
	if err != nil {
		log.Printf("AddShare update share failed uid:%d bid:%d", in.Head.Uid,
			in.Bid)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	return &common.CommReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
}

func checkAddress(db *sql.DB, uid, aid int64) bool {
	var id int64
	err := db.QueryRow("SELECT uid FROM address WHERE aid = ?", aid).Scan(&id)
	if err != nil {
		log.Printf("checkAddress failed aid:%d %v", aid, err)
		return false
	}
	return id == uid
}

func (s *server) SetWinStatus(ctx context.Context, in *modify.WinStatusRequest) (*common.CommReply, error) {
	var uid, s1, s2, status int64
	err := db.QueryRow("SELECT s.uid, s.status, l.status FROM sales s LEFT JOIN logistics l ON s.sid = l.sid WHERE s.sid = ?", in.Bid).
		Scan(&uid, &s1, &s2)
	if err != nil {
		log.Printf("SetWinStatus failed:%v", err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, err
	}
	if s2 > s1 {
		status = s2
	} else {
		status = s1
	}
	if uid != in.Head.Uid || status+1 != in.Status {
		log.Printf("SetWinStatus check failed uid:%d|%d status:%d|%d",
			in.Head.Uid, uid, in.Status, status)
		return &common.CommReply{
				Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}},
			errors.New("illegal param")
	}
	if in.Status == util.AddressStatus && (in.Aid > 0 || in.Account != "") {
		if in.Aid > 0 && !checkAddress(db, in.Head.Uid, in.Aid) {
			log.Printf("SetWinStatus checkAddress failed uid:%d aid:%d",
				in.Head.Uid, in.Aid)
			return &common.CommReply{
					Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}},
				errors.New("illegal param")
		}
		_, err := db.Exec("UPDATE sales SET status = 4 WHERE sid = ?", in.Bid)
		if err != nil {
			log.Printf("SetWinStatus failed:%v", err)
			return &common.CommReply{
				Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, err
		}
		_, err = db.Exec("INSERT INTO logistics(sid, status, uid, aid, account, ctime) VALUES (?, 4, ?, ?, ?, NOW())",
			in.Bid, in.Head.Uid, in.Aid)
		if err != nil {
			log.Printf("SetWinStatus failed:%v", err)
			return &common.CommReply{
				Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, err
		}
	} else if in.Status == util.ReceiptStatus {
		_, err := db.Exec("UPDATE logistics SET status = 6, rtime = NOW() WHERE sid = ?", in.Bid)
		if err != nil {
			log.Printf("SetWinStatus failed:%v", err)
			return &common.CommReply{
				Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, err
		}
	}
	return &common.CommReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
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

	s := grpc.NewServer()
	modify.RegisterModifyServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
