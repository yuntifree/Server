package util

import (
	"log"
	"math/rand"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	simplejson "github.com/bitly/go-simplejson"
)

const (
	paramErr = 2
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

func genParamErr(key string) string {
	return "get param:" + key + " failed"
}

//GetJSONString get json value of string
func GetJSONString(js *simplejson.Json, key string) string {
	if val, err := js.Get(key).String(); err == nil {
		return val
	}

	if val, err := js.Get("data").Get(key).String(); err == nil {
		return val
	}
	panic(AppError{Code: paramErr, Msg: genParamErr(key)})
}

//GetJSONStringDef get json value of string with default value
func GetJSONStringDef(js *simplejson.Json, key, def string) string {
	if val, err := js.Get(key).String(); err == nil {
		return val
	}

	if val, err := js.Get("data").Get(key).String(); err == nil {
		return val
	}
	return def
}

//GetJSONInt get json value of int
func GetJSONInt(js *simplejson.Json, key string) int64 {
	if val, err := js.Get(key).Int64(); err == nil {
		return val
	}

	if val, err := js.Get("data").Get(key).Int64(); err == nil {
		return val
	}
	panic(AppError{Code: paramErr, Msg: genParamErr(key)})
}

//GetJSONIntDef get json value of int with default value
func GetJSONIntDef(js *simplejson.Json, key string, def int64) int64 {
	if val, err := js.Get(key).Int64(); err == nil {
		return val
	}

	if val, err := js.Get("data").Get(key).Int64(); err == nil {
		return val
	}
	return def
}

//GetJSONBool get json value of bool
func GetJSONBool(js *simplejson.Json, key string) bool {
	if val, err := js.Get(key).Bool(); err == nil {
		return val
	}

	if val, err := js.Get("data").Get(key).Bool(); err == nil {
		return val
	}
	panic(AppError{Code: paramErr, Msg: genParamErr(key)})
}

//GetJSONBoolDef get json value of bool with default value
func GetJSONBoolDef(js *simplejson.Json, key string, def bool) bool {
	if val, err := js.Get(key).Bool(); err == nil {
		return val
	}

	if val, err := js.Get("data").Get(key).Bool(); err == nil {
		return val
	}
	return def
}

//GetJSONFloat get json value of float
func GetJSONFloat(js *simplejson.Json, key string) float64 {
	if val, err := js.Get(key).Float64(); err == nil {
		return val
	}

	if val, err := js.Get("data").Get(key).Float64(); err == nil {
		return val
	}
	panic(AppError{Code: paramErr, Msg: genParamErr(key)})
}

//GetJSONFloatDef get json value of float with default value
func GetJSONFloatDef(js *simplejson.Json, key string, def float64) float64 {
	if val, err := js.Get(key).Float64(); err == nil {
		return val
	}

	if val, err := js.Get("data").Get(key).Float64(); err == nil {
		return val
	}
	return def
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
