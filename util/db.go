package util

import (
	"database/sql"
	"log"
	"strconv"
	"strings"
	"time"

	redis "gopkg.in/redis.v5"
)

const (
	//MaxListSize for page
	MaxListSize = 20
	masterRds   = "rm-wz9sb2613092ki9xn.mysql.rds.aliyuncs.com"
	readRds     = "rm-wz9sb2613092ki9xn.mysql.rds.aliyuncs.com"
)

const (
	FlashAdWhiteType = iota
	BannerWhiteType
	ActivityWhiteType
	FlashAdDbgType
	LiveDbgType
)

//UserInfo user base information
type UserInfo struct {
	UID, Sex                    int
	NickName, HeadURL, UserName string
}

//ExistPhone return whether phone exist
func ExistPhone(db *sql.DB, phone string) bool {
	var cnt int
	err := db.QueryRow("SELECT COUNT(uid) FROM user WHERE phone = ?", phone).Scan(&cnt)
	if err != nil {
		log.Printf("query failed:%v", err)
		return false
	}
	if cnt > 0 {
		return true
	}
	return false
}

//CheckToken verify user's token
func CheckToken(db *sql.DB, uid int64, token string, ctype int64) bool {
	var tk string
	var expire bool
	var err error
	if ctype == 0 {
		err = db.QueryRow("SELECT token, IF(etime > NOW(), false, true) FROM user WHERE uid = ?", uid).Scan(&tk, &expire)
	} else {
		err = db.QueryRow("SELECT skey, IF(expire_time > NOW(), false, true) FROM back_login WHERE uid = ?", uid).Scan(&tk, &expire)
	}
	if err != nil {
		log.Printf("query failed:%v", err)
		return false
	}
	if expire {
		log.Print("token expire")
		return false
	}
	if tk != token {
		log.Printf("token not match token:%s real:%s", token, tk)
		return false
	}
	return true
}

//ClearToken set token expire
func ClearToken(db *sql.DB, uid int64) {
	_, err := db.Exec("UPDATE user SET etime = NOW() WHERE uid = ?", uid)
	if err != nil {
		log.Printf("query failed:%v", err)
	}
}

func genDsn(readonly bool) string {
	host := masterRds
	if readonly {
		host = readRds
	}
	return "access:^yunti9df3b01c$@tcp(" + host + ":3306)/yunxing?charset=utf8"
}

func genMonitorDsn() string {
	return "access:^yunti9df3b01c$@tcp(" + masterRds + ":3306)/monitor?charset=utf8"
}

//InitDB connect to rds
func InitDB(readonly bool) (*sql.DB, error) {
	dsn := genDsn(readonly)
	return sql.Open("mysql", dsn)
}

//InitMonitorDB connect to monitor rds
func InitMonitorDB() (*sql.DB, error) {
	dsn := genMonitorDsn()
	return sql.Open("mysql", dsn)
}

//GetUserInfo select user info
func GetUserInfo(db *sql.DB, uinfo *UserInfo) (err error) {
	query := "SELECT uid, username, nickname, headurl, sex FROM user WHERE "
	if uinfo.UID != 0 {
		query += " uid = " + strconv.Itoa(uinfo.UID)
	} else if uinfo.UserName != "" {
		query += " username = '" + uinfo.UserName + "'"
	}
	err = db.QueryRow(query).Scan(&uinfo.UID, &uinfo.UserName, &uinfo.NickName, &uinfo.HeadURL, &uinfo.Sex)
	return
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

//HasReddot check reddot
func HasReddot(db *sql.DB, uid int64) bool {
	if uid == 0 {
		return false
	}

	if !hasPhone(db, uid) || hasReceipt(db, uid) || hasShare(db, uid) {
		return true
	}

	return false
}

//GetSalesCount return purchase sales count
func GetSalesCount(db *sql.DB, sid, uid int64) int64 {
	var num int64
	err := db.QueryRow("SELECT COUNT(*) FROM sales_history WHERE uid = ? AND sid = ?", uid, sid).
		Scan(&num)
	if err != nil {
		log.Printf("getSalesCount query failed:%v", err)
	}
	return num
}

//GetSalesCodes return sales codes
func GetSalesCodes(db *sql.DB, sid, uid int64) []int64 {
	var codes []int64
	rows, err := db.Query("SELECT num FROM sales_history WHERE sid = ? AND uid = ?",
		sid, uid)
	if err != nil {
		log.Printf("getSalesCodes failed:%v", err)
		return codes
	}
	defer rows.Close()

	for rows.Next() {
		var code int64
		err := rows.Scan(&code)
		if err != nil {
			log.Printf("getSalesCodes scan failed:%v", err)
			continue
		}
		codes = append(codes, code)

	}
	return codes
}

//GetRemainSeconds return remain seconds for next award
func GetRemainSeconds(now int64) int64 {
	tt := time.Unix(now, 0)
	award := GetNextCqssc(tt)
	award = award.Add(120 * time.Second)
	return award.Unix() - tt.Unix()
}

//IsWhiteUser check white list for uid
func IsWhiteUser(db *sql.DB, uid, wtype int64) bool {
	var num int
	err := db.QueryRow("SELECT COUNT(id) FROM white_list WHERE type = ? AND deleted = 0 AND uid = ?",
		wtype, uid).Scan(&num)
	if err != nil {
		log.Printf("isWhiteUser query failed uid:%d %v", uid, err)
		return false
	}
	if num > 0 {
		return true
	}
	return false
}

//RefreshUserAp refresh user ap info
func RefreshUserAp(db *sql.DB, uid int64, apmac string) {
	var aid int
	mac := strings.Replace(strings.ToLower(apmac), ":", "", -1)
	log.Printf("ap mac origin:%s convert:%s\n", apmac, mac)
	err := db.QueryRow("SELECT id FROM ap WHERE mac = ? OR mac = ?", apmac, mac).Scan(&aid)
	if err != nil {
		log.Printf("select aid from ap failed uid:%d mac:%s err:%v\n", uid, apmac, err)
		db.Exec("UPDATE user SET aptime = NOW() WHERE uid = ?", uid)
		return
	}
	_, err = db.Exec("UPDATE user SET aid = ?, aptime = NOW() WHERE uid = ?", aid, uid)
	if err != nil {
		log.Printf("update user ap info failed uid:%d aid:%d\n", uid, aid)
		return
	}
	return
}

//GetPortalDir return portal dir
func GetPortalDir(db *sql.DB, ptype int64) (string, error) {
	var dir string
	err := db.QueryRow("SELECT dir FROM portal_page WHERE type = ? AND online = 1 ORDER BY id DESC LIMIT 1", ptype).Scan(&dir)
	return dir, err
}

func backupToken(db *sql.DB, uid int64, token, privdata string) {
	_, err := db.Exec("INSERT INTO token_backup(uid, token, private, ctime) VALUES (?, ?, ?, NOW()) ON DUPLICATE KEY UPDATE token = ?, private = ?",
		uid, token, privdata, token, privdata)
	if err != nil {
		log.Printf("backupToken failed uid:%d token:%s privdata:%s",
			uid, token, privdata)
	}
}

//RefreshTokenPrivdata refresh user token and privdata
func RefreshTokenPrivdata(db *sql.DB, kv *redis.Client, uid int64, expiretime int64) (string, string, int64, error) {
	var token, privdata string
	var expire int64
	err := db.QueryRow("SELECT token, private, UNIX_TIMESTAMP(etime) FROM user WHERE uid = ?", uid).
		Scan(&token, &privdata, &expire)
	if err != nil {
		log.Printf("refreshTokenPrivdata query failed uid:%d %v", uid, err)
		return token, privdata, expire, err
	}
	log.Printf("expire:%d, now:%d", expire, time.Now().Unix())
	if expire > time.Now().Unix() { //not expire
		return token, privdata, expire - time.Now().Unix(), nil
	}
	backupToken(db, uid, token, privdata)
	//token expire, update token
	token = GenSalt()
	privdata = GenSalt()
	_, err = db.Exec("UPDATE user SET token = ?, private = ?, etime = DATE_ADD(NOW(), INTERVAL 30 DAY) WHERE uid = ?",
		token, privdata, uid)
	if err != nil {
		log.Printf("refreshTokenPrivdata update failed uid:%d %v", uid, err)
		return token, privdata, expire, err
	}
	SetCachedToken(kv, uid, token)
	expire = expiretime
	return token, privdata, expire, nil
}
