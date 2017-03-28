package util

import (
	"database/sql"
	"log"
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

//IsTestUsermac check test user mac
func IsTestUsermac(usermac string) bool {
	for i := 0; i < len(testUsermacs); i++ {
		if testUsermacs[i] == usermac {
			return true
		}
	}
	return false
}

func getPortalHost(acname string) string {
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
	if IsTestAcname(acname) {
		if portaltype == 0 {
			ptype = PortalTestType
		} else {
			ptype = SceneTestType
		}
	} else {
		if portaltype == 0 {
			ptype = PortalType
		} else {
			ptype = SceneType
		}
	}
	dir, err := GetPortalDir(db, ptype)
	if err != nil {
		log.Printf("getPortalPath failed:%v", err)
	}
	host := getPortalHost(acname)
	return host + dir
}
