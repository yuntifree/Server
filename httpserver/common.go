package httpserver

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	"google.golang.org/grpc"

	"Server/proto/advertise"
	"Server/proto/common"
	"Server/proto/config"
	"Server/proto/discover"
	"Server/proto/fetch"
	"Server/proto/hot"
	"Server/proto/modify"
	"Server/proto/monitor"
	"Server/proto/push"
	"Server/proto/userinfo"
	"Server/proto/verify"
	"Server/util"

	simplejson "github.com/bitly/go-simplejson"
	nsq "github.com/nsqio/go-nsq"
)

const (
	ErrOk = iota
	ErrMissParam
	ErrInvalidParam
	ErrDatabase
	ErrInner
	ErrPanic
)
const (
	ErrToken = iota + 101
	ErrCode
	ErrGetCode
	ErrUsedPhone
	ErrWxMpLogin
	ErrUnionID
	ErrWxTicket
	ErrNotFound
	ErrIllegalPhone
	ErrZteLogin
	ErrZteRemove
	ErrNoNewVersion
	ErrHasPunch
	ErrIllegalCode
	ErrFrequencyLimit
)

var w *nsq.Producer

func init() {
	w = util.NewNsqProducer()
}

func extractAPIName(uri string) string {
	pos := strings.Index(uri, "?")
	path := uri
	if pos != -1 {
		path = uri[0:pos]
	}
	lpos := strings.LastIndex(path, "/")
	method := path
	if lpos != -1 {
		method = path[lpos+1:]
	}
	return method
}

//ReportRequest report request
func ReportRequest(uri string) {
	method := extractAPIName(uri)
	err := util.PubRequest(w, method)
	if err != nil {
		log.Printf("report request api:%s failed:%v", err)
	}
	return
}

//ReportSuccResp report success response
func ReportSuccResp(uri string) {
	method := extractAPIName(uri)
	err := util.PubResponse(w, method, 0)
	if err != nil {
		log.Printf("report response api:%s failed:%v", err)
	}
	return
}

func genParamErr(key string) string {
	return "get param:" + key + " failed"
}

func getJSONString(js *simplejson.Json, key string) string {
	if val, err := js.Get(key).String(); err == nil {
		return val
	}

	if val, err := js.Get("data").Get(key).String(); err == nil {
		return val
	}
	panic(util.AppError{Code: ErrMissParam, Msg: genParamErr(key)})
}

func getJSONStringDef(js *simplejson.Json, key, def string) string {
	if val, err := js.Get(key).String(); err == nil {
		return val
	}

	if val, err := js.Get("data").Get(key).String(); err == nil {
		return val
	}
	return def
}

func getJSONInt(js *simplejson.Json, key string) int64 {
	if val, err := js.Get(key).Int64(); err == nil {
		return val
	}

	if val, err := js.Get("data").Get(key).Int64(); err == nil {
		return val
	}
	panic(util.AppError{Code: ErrMissParam, Msg: genParamErr(key)})
}

func getJSONIntDef(js *simplejson.Json, key string, def int64) int64 {
	if val, err := js.Get(key).Int64(); err == nil {
		return val
	}

	if val, err := js.Get("data").Get(key).Int64(); err == nil {
		return val
	}
	return def
}

func getJSONBool(js *simplejson.Json, key string) bool {
	if val, err := js.Get(key).Bool(); err == nil {
		return val
	}

	if val, err := js.Get("data").Get(key).Bool(); err == nil {
		return val
	}
	panic(util.AppError{Code: ErrMissParam, Msg: genParamErr(key)})
}

func getJSONBoolDef(js *simplejson.Json, key string, def bool) bool {
	if val, err := js.Get(key).Bool(); err == nil {
		return val
	}

	if val, err := js.Get("data").Get(key).Bool(); err == nil {
		return val
	}
	return def
}

func getJSONFloat(js *simplejson.Json, key string) float64 {
	if val, err := js.Get(key).Float64(); err == nil {
		return val
	}

	if val, err := js.Get("data").Get(key).Float64(); err == nil {
		return val
	}
	panic(util.AppError{Code: ErrMissParam, Msg: genParamErr(key)})
}

func getJSONFloatDef(js *simplejson.Json, key string, def float64) float64 {
	if val, err := js.Get(key).Float64(); err == nil {
		return val
	}

	if val, err := js.Get("data").Get(key).Float64(); err == nil {
		return val
	}
	return def
}

func getFormInt(v url.Values, key, callback string) int64 {
	vals := v[key]
	if len(vals) == 0 {
		panic(util.AppError{Code: ErrMissParam, Msg: genParamErr(key),
			Callback: callback})
	}
	val, err := strconv.ParseInt(vals[0], 10, 64)
	if err != nil {
		panic(util.AppError{Code: ErrMissParam, Msg: genParamErr(key),
			Callback: callback})
	}
	return val
}

func getFormIntDef(v url.Values, key string, def int64) int64 {
	vals := v[key]
	if len(vals) == 0 {
		return def
	}
	val, err := strconv.ParseInt(vals[0], 10, 64)
	if err != nil {
		return def
	}
	return val
}

func getFormFloat(v url.Values, key, callback string) float64 {
	vals := v[key]
	if len(vals) == 0 {
		panic(util.AppError{Code: ErrMissParam, Msg: genParamErr(key),
			Callback: callback})
	}
	val, err := strconv.ParseFloat(vals[0], 64)
	if err != nil {
		panic(util.AppError{Code: ErrMissParam, Msg: genParamErr(key),
			Callback: callback})
	}
	return val
}

func getFormFloatDef(v url.Values, key string, def float64) float64 {
	vals := v[key]
	if len(vals) == 0 {
		return def
	}
	val, err := strconv.ParseFloat(vals[0], 64)
	if err != nil {
		return def
	}
	return val
}

func getFormBool(v url.Values, key, callback string) bool {
	vals := v[key]
	if len(vals) == 0 {
		panic(util.AppError{Code: ErrMissParam, Msg: genParamErr(key),
			Callback: callback})
	}
	val, err := strconv.ParseBool(vals[0])
	if err != nil {
		panic(util.AppError{Code: ErrMissParam, Msg: genParamErr(key),
			Callback: callback})
	}
	return val
}

func getFormBoolDef(v url.Values, key string, def bool) bool {
	vals := v[key]
	if len(vals) == 0 {
		return def
	}
	val, err := strconv.ParseBool(vals[0])
	if err != nil {
		return def
	}
	return val
}

func getFormString(v url.Values, key, callback string) string {
	vals := v[key]
	if len(vals) == 0 {
		panic(util.AppError{Code: ErrMissParam, Msg: genParamErr(key),
			Callback: callback})
	}
	return vals[0]
}

func getFormStringDef(v url.Values, key string, def string) string {
	vals := v[key]
	if len(vals) == 0 {
		return def
	}
	return vals[0]
}

//Request request infos
type Request struct {
	Post     *simplejson.Json
	Form     url.Values
	debug    bool
	Callback string
}

func writeRsp(w http.ResponseWriter, body []byte, callback string) {
	if callback != "" {
		var buf bytes.Buffer
		buf.Write([]byte(callback))
		buf.Write([]byte("("))
		buf.Write(body)
		buf.Write([]byte(")"))
		w.Write(buf.Bytes())
		return
	}
	w.Write(body)
	return
}

//WriteRsp support for callback
func (r *Request) WriteRsp(w http.ResponseWriter, body []byte) {
	writeRsp(w, body, r.Callback)
}

//Init init request
func (r *Request) Init(req *http.Request) {
	ReportRequest(req.RequestURI)
	var err error
	r.Post, err = simplejson.NewFromReader(req.Body)
	if err == io.EOF {
		req.ParseForm()
		r.Form = req.Form
		r.debug = true
		r.Callback = getFormString(r.Form, "callback", "")
		return
	}
	if err != nil {
		log.Printf("parse reqbody failed:%v", err)
		panic(util.AppError{ErrInvalidParam, "invalid param", r.Callback})
	}
}

//InitCheck init request and check token
func (r *Request) InitCheck(req *http.Request, back bool) {
	r.Init(req)
	uid := r.GetParamInt("uid")
	token := r.GetParamString("token")

	var ctype int64
	if back {
		ctype = 1
	}

	flag := checkToken(uid, token, ctype)
	if !flag {
		log.Printf("checkToken failed, uid:%d token:%s\n", uid, token)
		panic(util.AppError{ErrToken, "token验证失败", r.Callback})
	}
}

//InitCheckApp init request and check token for app
func (r *Request) InitCheckApp(req *http.Request) {
	r.InitCheck(req, false)
}

//InitCheckOss init request and check token for oss
func (r *Request) InitCheckOss(req *http.Request) {
	r.InitCheck(req, true)
}

func (r *Request) GetParamInt(key string) int64 {
	if r.debug {
		return getFormInt(r.Form, key, r.Callback)
	}
	return getJSONInt(r.Post, key)
}

func (r *Request) GetParamIntDef(key string, def int64) int64 {
	if r.debug {
		return getFormIntDef(r.Form, key, def)
	}
	return getJSONIntDef(r.Post, key, def)
}

func (r *Request) GetParamBool(key string) bool {
	if r.debug {
		return getFormBool(r.Form, key, r.Callback)
	}
	return getJSONBool(r.Post, key)
}

func (r *Request) GetParamBoolDef(key string, def bool) bool {
	if r.debug {
		return getFormBoolDef(r.Form, key, def)
	}
	return getJSONBoolDef(r.Post, key, def)
}

func (r *Request) GetParamString(key string) string {
	if r.debug {
		return getFormString(r.Form, key, r.Callback)
	}
	return getJSONString(r.Post, key)
}
func (r *Request) GetParamStringDef(key string, def string) string {
	if r.debug {
		return getFormStringDef(r.Form, key, def)
	}
	return getJSONStringDef(r.Post, key, def)
}

func (r *Request) GetParamFloat(key string) float64 {
	if r.debug {
		return getFormFloat(r.Form, key, r.Callback)
	}
	return getJSONFloat(r.Post, key)
}
func (r *Request) GetParamFloatDef(key string, def float64) float64 {
	if r.debug {
		return getFormFloatDef(r.Form, key, def)
	}
	return getJSONFloatDef(r.Post, key, def)
}

func extractError(r interface{}) *util.AppError {
	if k, ok := r.(util.AppError); ok {
		return &k
	}
	log.Printf("unexpected panic:%v", r)
	return &util.AppError{ErrPanic, r.(error).Error(), ""}
}

func handleError(w http.ResponseWriter, e *util.AppError) {
	log.Printf("error code:%d msg:%s callback:%s", e.Code, e.Msg,
		e.Callback)

	js, _ := simplejson.NewJson([]byte(`{}`))
	js.Set("errno", e.Code)
	if e.Code == ErrInvalidParam || e.Code == ErrMissParam {
		js.Set("errno", ErrToken)
		js.Set("desc", "服务器又傲娇了~")
	} else if e.Code < ErrToken {
		js.Set("desc", "服务器又傲娇了~")
	} else {
		js.Set("desc", e.Msg)
	}
	body, err := js.MarshalJSON()
	if err != nil {
		log.Printf("MarshalJSON failed: %v", err)
		writeRsp(w, []byte(`{"errno":2,"desc":"invalid param"}`), e.Callback)
		return
	}
	writeRsp(w, body, e.Callback)
}

type AppHandler func(http.ResponseWriter, *http.Request) *util.AppError

func (fn AppHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			apperr := extractError(r)
			handleError(w, apperr)
		}
	}()
	if e := fn(w, r); e != nil {
		handleError(w, e)
	}
}

func checkToken(uid int64, token string, ctype int64) bool {
	address := GetNameServer(uid, util.VerifyServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Printf("did not connect: %v", err)
		return false
	}
	defer conn.Close()
	c := verify.NewVerifyClient(conn)

	uuid := util.GenUUID()
	res, err := c.CheckToken(context.Background(),
		&verify.TokenRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Token: token, Type: ctype})
	if err != nil {
		log.Printf("failed: %v", err)
		return false
	}

	if res.Head.Retcode != 0 {
		log.Printf("check token failed")
		return false
	}

	return true
}

//GetAps get ap infos
func GetAps(w http.ResponseWriter, r *http.Request, back bool) (apperr *util.AppError) {
	defer func() {
		if r := recover(); r != nil {
			apperr = extractError(r)
		}
	}()

	var req Request
	if back {
		req.InitCheckOss(r)
	} else {
		req.InitCheckApp(r)
	}
	uid := req.GetParamInt("uid")
	longitude := req.GetParamFloat("longitude")
	latitude := req.GetParamFloat("latitude")

	address := GetNameServer(uid, util.FetchServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return &util.AppError{ErrInner, err.Error(), req.Callback}
	}
	defer conn.Close()
	c := fetch.NewFetchClient(conn)
	uuid := util.GenUUID()
	res, err := c.FetchAps(context.Background(),
		&fetch.ApRequest{Head: &common.Head{Uid: uid, Sid: uuid},
			Longitude: longitude, Latitude: latitude})
	if err != nil {
		return &util.AppError{ErrInner, err.Error(), req.Callback}
	}

	if res.Head.Retcode != 0 {
		return &util.AppError{ErrInner, "服务器又傲娇了", req.Callback}
	}

	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		return &util.AppError{ErrInner, "init json failed", req.Callback}
	}
	infos := make([]interface{}, len(res.Infos))
	for i := 0; i < len(res.Infos); i++ {
		json, _ := simplejson.NewJson([]byte(`{}`))
		json.Set("id", res.Infos[i].Id)
		json.Set("longitude", res.Infos[i].Longitude)
		json.Set("latitude", res.Infos[i].Latitude)
		json.Set("address", res.Infos[i].Address)
		infos[i] = json
	}
	js.SetPath([]string{"data", "infos"}, infos)

	body, err := js.MarshalJSON()
	if err != nil {
		return &util.AppError{ErrInner, "marshal json failed", req.Callback}
	}
	RspGzip(w, body)
	ReportSuccResp(r.RequestURI)
	return nil
}

func getDiscoverAddress() string {
	ip := util.GetInnerIP()
	if ip != util.DebugHost {
		hosts := strings.Split(util.APIHosts, ",")
		if len(hosts) > 0 {
			idx := util.Randn(int32(len(hosts)))
			return hosts[idx] + util.DiscoverServerPort
		}
	}
	return "localhost" + util.DiscoverServerPort
}

//GetNameServer
func GetNameServer(uid int64, name string) string {
	return GetNameServerCallback(uid, name, "")
}

//GetNameServerCallback get server from name service with callback
func GetNameServerCallback(uid int64, name, callback string) string {
	address := getDiscoverAddress()
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Printf("did not connect %s: %v", address, err)
		panic(util.AppError{ErrInner, err.Error(), callback})
	}
	defer conn.Close()
	c := discover.NewDiscoverClient(conn)

	ip := util.GetInnerIP()
	if ip == util.DebugHost {
		name += ":debug"
	}
	uuid := util.GenUUID()
	res, err := c.Resolve(context.Background(),
		&discover.ServerRequest{Head: &common.Head{Uid: uid, Sid: uuid}, Sname: name})
	if err != nil {
		log.Printf("Resolve failed %s: %v", name, err)
		panic(util.AppError{ErrInner, err.Error(), callback})
	}

	if res.Head.Retcode != 0 {
		log.Printf("Resolve failed  name:%s errcode:%d\n", name, res.Head.Retcode)
		panic(util.AppError{ErrInner,
			fmt.Sprintf("Resolve failed  name:%s errcode:%d\n", name, res.Head.Retcode), callback})
	}

	return res.Host
}

//RspGzip response with gzip
func RspGzip(w http.ResponseWriter, body []byte) {
	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Set("Content-Type", "application/json")
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	gw.Write(body)
	gw.Close()
	w.Write(buf.Bytes())
}

//AddImages add images
func AddImages(uid int64, names []string) error {
	address := GetNameServer(uid, util.ModifyServerName)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return err
	}
	defer conn.Close()
	c := modify.NewModifyClient(conn)

	uuid := util.GenUUID()
	res, err := c.AddImage(context.Background(),
		&modify.AddImageRequest{Head: &common.Head{Sid: uuid, Uid: uid},
			Fnames: names})
	if err != nil {
		return err
	}
	if res.Head.Retcode != 0 {
		return errors.New("添加图片失败")
	}
	return nil
}

//GenResponseBody generate response body
func GenResponseBody(res interface{}, flag bool) []byte {
	return GenResponseBodyCallback(res, "", flag)
}

//MergeResponseBody merge multiple rpc response
func MergeResponseBody(responses []interface{}) []byte {
	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		panic(util.AppError{ErrInner, err.Error(), ""})
	}

	for _, res := range responses {
		val := reflect.ValueOf(res).Elem()
		for i := 0; i < val.NumField(); i++ {
			valueField := val.Field(i)
			typeField := val.Type().Field(i)
			if typeField.Name == "Head" {
				continue
			} else {
				js.SetPath([]string{"data", strings.ToLower(typeField.Name)},
					valueField.Interface())
			}
		}
	}
	data, err := js.MarshalJSON()
	if err != nil {
		panic(util.AppError{ErrInner, err.Error(), ""})
	}

	return data
}

//GenResponseBodyCallback generate response body with callback
func GenResponseBodyCallback(res interface{}, callback string, flag bool) []byte {
	js, err := simplejson.NewJson([]byte(`{"errno":0}`))
	if err != nil {
		panic(util.AppError{ErrInner, err.Error(), callback})
	}
	val := reflect.ValueOf(res).Elem()
	for i := 0; i < val.NumField(); i++ {
		valueField := val.Field(i)
		typeField := val.Type().Field(i)
		if typeField.Name == "Head" {
			if flag {
				headVal := reflect.Indirect(valueField)
				uid := headVal.FieldByName("Uid")
				js.SetPath([]string{"data", "uid"}, uid.Interface())

			} else {
				continue
			}
		} else {
			js.SetPath([]string{"data", strings.ToLower(typeField.Name)},
				valueField.Interface())
		}
	}
	data, err := js.MarshalJSON()
	if err != nil {
		panic(util.AppError{ErrInner, err.Error(), callback})
	}

	return data
}

//CheckRPCErr check rpc response error
func CheckRPCErr(err reflect.Value, method string) {
	CheckRPCErrCallback(err, method, "")
	return
}

func CheckRPCErrCallback(err reflect.Value, method, callback string) {
	if err.Interface() != nil {
		log.Printf("RPC %s failed:%v", method, err)
		panic(util.AppError{ErrInner, "grpc failed " + method, callback})
	}
}

//CheckRPCCode check rpc response code
func CheckRPCCode(retcode common.ErrCode, method string) {
	CheckRPCCodeCallback(retcode, method, "")
	return
}

//CheckRPCCodeCallback check rpc response code with callback
func CheckRPCCodeCallback(retcode common.ErrCode, method, callback string) {
	if retcode != 0 {
		log.Printf("%s failed retcode:%d", method, retcode)
	}
	if retcode == common.ErrCode_INVALID_TOKEN {
		panic(util.AppError{ErrToken, "token验证失败", callback})
	} else if retcode == common.ErrCode_USED_PHONE {
		panic(util.AppError{ErrUsedPhone, "该账号已注册，请直接登录", callback})
	} else if retcode == common.ErrCode_CHECK_CODE {
		panic(util.AppError{ErrCode, "验证码错误", callback})
	} else if retcode == common.ErrCode_ZTE_LOGIN {
		panic(util.AppError{ErrZteLogin, "登录失败", callback})
	} else if retcode == common.ErrCode_ZTE_REMOVE {
		panic(util.AppError{ErrZteRemove, "删除中兴账号失败", callback})
	} else if retcode == common.ErrCode_NO_NEW_VERSION {
		panic(util.AppError{ErrNoNewVersion, "当前已是最新版本", callback})
	} else if retcode == common.ErrCode_HAS_PUNCH {
		panic(util.AppError{ErrHasPunch, "此地已经被别人打过卡", callback})
	} else if retcode == common.ErrCode_ILLEGAL_CODE {
		panic(util.AppError{ErrIllegalCode, "code已过期", callback})
	} else if retcode == common.ErrCode_FREQUENCY_LIMIT {
		panic(util.AppError{ErrFrequencyLimit, "请求太频繁", callback})
	} else if retcode != 0 {
		panic(util.AppError{int(retcode), "服务器又傲娇了~", callback})
	}
}

func genServerName(rtype int64, callback string) string {
	switch rtype {
	case util.DiscoverServerType:
		return util.DiscoverServerName
	case util.VerifyServerType:
		return util.VerifyServerName
	case util.HotServerType:
		return util.HotServerName
	case util.FetchServerType:
		return util.FetchServerName
	case util.ModifyServerType:
		return util.ModifyServerName
	case util.PushServerType:
		return util.PushServerName
	case util.PunchServerType:
		return util.PunchServerName
	case util.UserinfoServerType:
		return util.UserinfoServerName
	case util.ConfigServerType:
		return util.ConfigServerName
	case util.MonitorServerType:
		return util.MonitorServerName
	case util.AdvertiseServerType:
		return util.AdvertiseServerName
	default:
		panic(util.AppError{ErrInvalidParam, "illegal server type", callback})
	}
}

func genClient(rtype int64, conn *grpc.ClientConn, callback string) interface{} {
	var cli interface{}
	switch rtype {
	case util.DiscoverServerType:
		cli = discover.NewDiscoverClient(conn)
	case util.VerifyServerType:
		cli = verify.NewVerifyClient(conn)
	case util.HotServerType:
		cli = hot.NewHotClient(conn)
	case util.FetchServerType:
		cli = fetch.NewFetchClient(conn)
	case util.ModifyServerType:
		cli = modify.NewModifyClient(conn)
	case util.PushServerType:
		cli = push.NewPushClient(conn)
	case util.UserinfoServerType:
		cli = userinfo.NewUserinfoClient(conn)
	case util.ConfigServerType:
		cli = config.NewConfigClient(conn)
	case util.MonitorServerType:
		cli = monitor.NewMonitorClient(conn)
	case util.AdvertiseServerType:
		cli = advertise.NewAdvertiseClient(conn)
	default:
		panic(util.AppError{ErrInvalidParam, "illegal server type", callback})
	}
	return cli
}

//CallRPC call rpc method
func CallRPC(rtype, uid int64, method string, request interface{}) (reflect.Value, reflect.Value) {
	return CallRPCCallback(rtype, uid, method, "", request)
}

//CallRPC call rpc method with callback
func CallRPCCallback(rtype, uid int64, method, callback string, request interface{}) (reflect.Value, reflect.Value) {
	var resp reflect.Value
	serverName := genServerName(rtype, callback)
	address := GetNameServerCallback(uid, serverName, callback)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return resp, reflect.ValueOf(err)
	}
	defer conn.Close()
	cli := genClient(rtype, conn, callback)
	ctx := context.Background()

	inputs := make([]reflect.Value, 2)
	inputs[0] = reflect.ValueOf(ctx)
	inputs[1] = reflect.ValueOf(request)
	arr := reflect.ValueOf(cli).MethodByName(method).Call(inputs)
	if len(arr) != 2 {
		log.Printf("callRPC arr len%d", len(arr))
		return resp, reflect.ValueOf(errors.New("illegal grpc call response"))
	}
	return arr[0], arr[1]
}

//GetConf get config
func GetConf(w http.ResponseWriter, r *http.Request, back bool) (apperr *util.AppError) {
	var req Request
	if back {
		req.InitCheckOss(r)
	} else {
		req.InitCheckApp(r)
	}
	uid := req.GetParamInt("uid")

	uuid := util.GenUUID()
	resp, rpcerr := CallRPC(util.FetchServerType, uid, "FetchConf",
		&common.CommRequest{Head: &common.Head{Sid: uuid, Uid: uid}})
	CheckRPCErr(rpcerr, "FetchConf")
	res := resp.Interface().(*fetch.ConfReply)
	CheckRPCCode(res.Head.Retcode, "FetchConf")

	body := GenResponseBody(res, false)
	w.Write(body)
	return nil
}

//FileHandler wrapper for FileServer
type FileHandler struct {
	Dir string
	h   http.Handler
}

//NewFileHandler return new FileHandler
func NewFileHandler(dir string) *FileHandler {
	return &FileHandler{
		Dir: dir,
		h:   http.FileServer(http.Dir(dir)),
	}
}

//ServeHTTP FileHandler implemention
func (f FileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("url:%s", r.URL)
	f.h.ServeHTTP(w, r)
}
