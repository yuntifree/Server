package util

import (
	"math/rand"
	"net"
	"strconv"
	"strings"
	"time"
)

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
