package util

import (
	"log"
	"math/rand"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func init() {
	log.SetFlags(log.Ldate | log.Lmicroseconds | log.Llongfile)
}

//GenWifiPass gen 4-digit password
func GenWifiPass() string {
	r := rand.New(rand.NewSource(time.Now().Unix()))
	var pass string
	for i := 0; i < 4; i++ {
		pass += strconv.Itoa(r.Intn(10))
	}

	return pass
}

//IsIntranet check intranet ip
func IsIntranet(ip string) bool {
	arr := strings.Split(ip, ".")
	if len(arr) != 4 {
		return false
	}

	if strings.HasPrefix(ip, "10.") || strings.HasPrefix(ip, "192.168.") {
		return true
	}

	//172.16.0.0 -- 172.31.255.255
	if strings.HasPrefix(ip, "172.") {
		second, err := strconv.ParseInt(arr[1], 10, 64)
		if err != nil {
			return false
		}

		if second >= 16 && second <= 31 {
			return true
		}
	}

	return false
}

//GetInnerIP return inner ip of host
func GetInnerIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}

	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}

		if ip == nil || ip.IsLoopback() {
			continue
		}

		ip = ip.To4()
		if ip == nil {
			continue
		}
		ipstr := ip.String()
		if IsIntranet(ipstr) {
			return ipstr
		}
	}

	return ""
}

//GetNextCqssc return next cqssc time
func GetNextCqssc(tt time.Time) time.Time {
	year, month, day := tt.Date()
	local := tt.Location()
	hour, min, _ := tt.Clock()
	if hour >= 10 && hour < 22 {
		min = (min/10 + 1) * 10
	} else if hour >= 22 && hour < 2 {
		min = (min/5 + 1) * 5
	} else {
		hour = 10
		min = 0
	}

	return time.Date(year, month, day, hour, min, 0, 0, local)
}

//IsIllegalPhone check phone format 11-number begin with 1
func IsIllegalPhone(phone string) bool {
	flag, err := regexp.MatchString(`^1\d{10}$`, phone)
	if err != nil {
		log.Printf("IsIllegalPhone MatchString failed:%v", err)
	}
	return flag
}

//CheckTermVersion check for hot news compatibility
func CheckTermVersion(term, version int64) bool {
	if (term == 0 && version < 6) || (term == 1 && version < 4) {
		return false
	}
	return true
}
