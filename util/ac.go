package util

import (
	"database/sql"
	"log"
	"time"
)

const (
	portalDir    = "http://api.yunxingzh.com/"
	testDir      = "http://120.76.236.185/"
	innerSshHost = "http://192.168.100.4:8080/"
	innerWjjHost = "http://192.168.200.4:8080/"
)

var sshAcnames = []string{
	"0110.0001.001.01",
	"2013.0769.200.00",
	"2043.0769.200.00",
	"AC_SSH_A_01",
	"AC_SSH_A_02",
	"AC_SSH_A_03",
	"AC_SSH_A_04",
	"AC_SSH_A_05",
	"AC_SSH_A_06",
	"AC_SSH_A_07",
	"AC_SSH_A_08",
	"AC_SSH_A_09",
	"AC_SSH_B_10",
	"AC-SSH-02-11",
	"AC_JYJ_01",
	"JYJ_RJ01",
	"AC_JYJ_02",
	"AC_JYJ_03",
	"AC_JYJ_04",
}

var wjjAcnames = []string{
	"AC_120_A_01",
	"AC_120_A_02",
	"AC_120_A_03",
	"AC_120_A_04",
	"AC_120_A_05",
	"AC_120_A_06",
	"AC_120_A_07",
	"AC_120_A_08",
	"AC_120_A_09",
	"AC_120_A_10",
	"TRX1",
	"TRX2",
	"TRX3",
}

var testUsermacs = []string{
	"F45C89987347",
	"14F65A9F590C",
	"0C51015B928B",
	"20AB37909A39",
	"60F81D405892",
	"D065CA2F5BC6",
}

var wjjKongguAcnames = []string{
	"AC_120_A_04",
	"AC_120_A_05",
}

//IsSshAcname check ssh acname
func IsSshAcname(acname string) bool {
	for i := 0; i < len(sshAcnames); i++ {
		if sshAcnames[i] == acname {
			return true
		}
	}
	return false
}

//IsWjjAcname check ssh acname
func IsWjjAcname(acname string) bool {
	for i := 0; i < len(wjjAcnames); i++ {
		if wjjAcnames[i] == acname {
			return true
		}
	}
	return false
}

//IsTestAcname check test acname
func IsTestAcname(acname string) bool {
	if acname == "AC_SSH_A_04" {
		return true
	}
	return false
}

//IsKongguAcname check konggu acname
func IsKongguAcname(acname string) bool {
	if acname == "AC_SSH_B_10" {
		return true
	}
	return false
}

//IsWjjKongguAcname check konggu acname
func IsWjjKongguAcname(acname string) bool {
	for i := 0; i < len(wjjKongguAcnames); i++ {
		if wjjKongguAcnames[i] == acname {
			return true
		}
	}
	return false
}

//IsLzfAcname chec lianzufang acname
func IsLzfAcname(acname string) bool {
	if acname == "AC_SSH_A_06" || acname == "AC_SSH_A_07" ||
		acname == "AC_SSH_A_08" || acname == "AC_SSH_A_09" ||
		acname == "AC_SSH_A_10" {
		return true
	}
	return false
}

//IsTestUsermac check test user mac
func IsTestUsermac(usermac string) bool {
	for i := 0; i < len(testUsermacs); i++ {
		if testUsermacs[i] == usermac {
			return true
		}
	}
	return false
}

//GetWjjHost get wjj host
func GetWjjHost() string {
	return innerWjjHost
}

//GetSshHost get ssh host
func GetSshHost() string {
	return innerSshHost
}

//GetPortalHost get portal host
func GetPortalHost(acname string) string {
	host := portalDir
	if IsTestAcname(acname) {
		host = testDir
	} else {
		if IsSshAcname(acname) {
			host = innerSshHost
		} else if IsWjjAcname(acname) {
			host = innerWjjHost
		}
	}
	return host
}

//GetPortalPath get portal path
func GetPortalPath(db *sql.DB, acname string, portaltype int64) string {
	var dir string
	var ptype int64
	host := portalDir
	if IsTestAcname(acname) {
		host = testDir
		if portaltype == 0 {
			ptype = PortalTestType
		} else {
			ptype = SceneTestType
		}
	} else {
		if portaltype == 0 {
			if IsWjjAcname(acname) {
				ptype = WjjPortalType
			} else {
				ptype = PortalType
			}
		} else {
			ptype = SceneType
		}
	}
	var err error
	if IsKongguAcname(acname) {
		dir = "portal201703212030/"
	} else {
		dir, err = GetPortalDir(db, ptype)
		if err != nil {
			log.Printf("getPortalPath failed:%v", err)
		}
	}
	return host + dir
}

//GetLoginPath get login path
func GetLoginPath(db *sql.DB, acname string, portaltype int64) string {
	var ptype int64
	if IsTestAcname(acname) {
		ptype = LoginTestType
	} else {
		if IsWjjAcname(acname) {
			ptype = WjjLoginType
		} else {
			ptype = LoginType
		}
	}
	host := GetPortalHost(acname)
	dir, err := GetPortalDir(db, ptype)
	if err != nil {
		log.Printf("GetLoginPath failed:%v", err)
	}
	return host + dir
}

func getApUnit(db *sql.DB, apmac string) int64 {
	if apmac == "" {
		return 0
	}
	var unit int64
	err := db.QueryRow("SELECT unid FROM ap_info WHERE mac = ?", apmac).Scan(&unit)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("getApUnit query failed:%v", err)
	}
	return unit
}

//GetUnitArea get unit area
func GetUnitArea(db *sql.DB, unit int64) int64 {
	if unit == 0 {
		return 0
	}
	var area int64
	err := db.QueryRow("SELECT aid FROM area_unit WHERE deleted = 0 AND unid = ?", unit).Scan(&area)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("getUnitArea query failed:%v", err)
	}
	return area
}

func getAreaAd(db *sql.DB, area int64) int64 {
	if area == 0 {
		return 0
	}
	var aid int64
	var start, end int
	err := db.QueryRow("SELECT a.id, ts.start, ts.end FROM advertise a, timeslot ts WHERE a.tsid = ts.id AND a.areaid = ? AND a.online = 1", area).Scan(&aid, start, end)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("getAreaAd query failed:%v", err)
		return aid
	}
	now := time.Now()
	hour := now.Hour()
	min := now.Minute()
	ts := hour*100 + min
	if ts >= start && ts <= end {
		return aid
	}
	return 0
}

//GetAdType get ad type
func GetAdType(db *sql.DB, apmac string) int64 {
	unit := getApUnit(db, apmac)
	area := GetUnitArea(db, unit)
	return area
}

//GetUnitPortal get unit portal
func GetUnitPortal(db *sql.DB, unit int64) int64 {
	var ptype int64
	err := db.QueryRow("SELECT id FROM custom_portal WHERE deleted = 0 AND unid = ?", unit).
		Scan(&ptype)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("getUnitPortal query failed:%v", err)
	}
	return ptype
}

//GetPortalType get portal type
func GetPortalType(db *sql.DB, apmac string) int64 {
	unit := getApUnit(db, apmac)
	ptype := GetUnitPortal(db, unit)
	return ptype
}
