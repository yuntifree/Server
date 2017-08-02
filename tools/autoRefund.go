package main

import (
	"Server/util"
	"Server/weixin"
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	db, err := util.InitInquiryDB()
	if err != nil {
		log.Printf("InitDB failed:%v", err)
		os.Exit(1)
	}

	//get refund list
	rows, err := db.Query(`SELECT i.id, r.id, i.doctor, i.patient FROM 
	inquiry_history i, refund_history r WHERE r.hid = i.id AND 
	i.status = 3 AND r.status = 0
	AND i.ptime > DATE_ADD(NOW(), INTERVAL -72 HOUR) 
	AND i.ptime < DATE_ADD(NOW(), INTERVAL -71 HOUR)`)
	if err != nil {
		log.Printf("query failed:%v", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var hid, rid, doctor, patient int64
		err = rows.Scan(&hid, &rid, &doctor, &patient)
		if err != nil {
			log.Printf("scan failed:%v", err)
			continue
		}
		refundInquiry(db, hid, rid, doctor, patient)
	}
}

func refundInquiry(db *sql.DB, hid, rid, doctor, patient int64) {
	var oid, fee int64
	var tradeno string
	err := db.QueryRow("SELECT id, oid, fee FROM orders WHERE item = ? AND status = 1", hid).
		Scan(&oid, &tradeno, &fee)
	if err != nil {
		log.Printf("refundInquiry get order info failed:%v", err)
		return
	}
	refundno := util.GenSalt()
	if !weixin.Refund(tradeno, refundno, fee, fee) {
		log.Printf("refundInquiry refund failed:%d %s %d", hid, tradeno, fee)
		util.SendAlertMail(fmt.Sprintf("refundInquiry refund failed:%d %s %d", hid, tradeno, fee))
		return
	}
	_, err = db.Exec("UPDATE orders SET status = 2, refundno = ?, rtime = NOW() WHERE id = ?",
		refundno, oid)
	if err != nil {
		log.Printf("refundInquiry update order status failed:%d", oid)
		return
	}
	_, err = db.Exec("UPDATE inquiry_history SET status = 4 WHERE id = ?", hid)
	if err != nil {
		log.Printf("refundInquiry update inquiry_history status failed:%d", oid)
		return
	}
	_, err = db.Exec("UPDATE refund_history SET status = 1 WHERE id = ?", rid)
	if err != nil {
		log.Printf("refundInquiry update refund_history status failed:%d", oid)
		return
	}
	_, err = db.Exec("UPDATE relations SET status = 4 WHERE doctor = ? AND patient = ?", doctor, patient)
	if err != nil {
		log.Printf("refundInquiry update relations status failed:%d", oid)
		return
	}

}
