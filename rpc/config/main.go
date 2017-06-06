package main

import (
	"fmt"
	"log"
	"net"
	"strings"

	"database/sql"

	"Server/proto/common"
	"Server/proto/config"
	"Server/util"

	_ "github.com/go-sql-driver/mysql"
	nsq "github.com/nsqio/go-nsq"
	"golang.org/x/net/context"
	redis "gopkg.in/redis.v5"
)

const (
	menuType           = 0
	tabType            = 1
	menuV2Type         = 2
	tabV2Type          = 3
	newsNum            = 10
	hospitalIntro      = 0
	hospitalService    = 1
	appBannerType      = 0
	portalBannerType   = 4
	portalBannerV2Type = 6
)

type server struct{}

var db *sql.DB
var kv *redis.Client
var w *nsq.Producer

func getPortalMenu(db *sql.DB, stype int64, flag bool) []*config.PortalMenuInfo {
	query := fmt.Sprintf("SELECT icon, text, name, routername, url, subtype FROM portal_menu WHERE type = %d AND deleted = 0 ", stype)
	if !flag {
		query += " AND dbg = 0 "
	}
	query += " ORDER BY priority DESC"
	rows, err := db.Query(query)
	var infos []*config.PortalMenuInfo
	if err != nil {
		log.Printf("getPortalMenu query failed:%v", err)
		return infos
	}

	defer rows.Close()
	for rows.Next() {
		var info config.PortalMenuInfo
		err := rows.Scan(&info.Icon, &info.Text, &info.Name, &info.Routername,
			&info.Url, &info.Type)
		if err != nil {
			log.Printf("getPortalMenu scan failed:%v", err)
			continue
		}
		infos = append(infos, &info)
	}
	return infos
}

func (s *server) GetPortalMenu(ctx context.Context, in *common.CommRequest) (*config.PortalMenuReply, error) {
	util.PubRPCRequest(w, "config", "GetPortalMenu")
	flag := util.IsWhiteUser(db, in.Head.Uid, util.PortalMenuDbgType)
	menulist := getPortalMenu(db, menuType, flag)
	tablist := getPortalMenu(db, tabType, flag)
	util.PubRPCSuccRsp(w, "config", "GetPortalMenu")
	return &config.PortalMenuReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}, Menulist: menulist,
		Tablist: tablist}, nil
}

func getOnlineService(db *sql.DB) []*config.PortalService {
	var infos []*config.PortalService
	var info config.PortalService
	info.Name = "上网服务"
	var items []*config.MediaInfo
	rows, err := db.Query("SELECT id, img, dst, title FROM online_service WHERE deleted = 0 ORDER BY priority DESC")
	if err != nil {
		log.Printf("getOnlineService query failed:%v", err)
		return infos
	}

	defer rows.Close()
	for rows.Next() {
		var item config.MediaInfo
		err := rows.Scan(&item.Id, &item.Img, &item.Dst, &item.Title)
		if err != nil {
			log.Printf("getOnlineService scan failed:%v", err)
			continue
		}
		item.Type = 15
		items = append(items, &item)
	}
	if len(items) > 0 {
		info.Items = items
		infos = append(infos, &info)
	}
	return infos
}

func getHospitalInfos(db *sql.DB, hid, stype int64) []*config.MediaInfo {
	var items []*config.MediaInfo
	rows, err := db.Query("SELECT id, img, dst, title, routername FROM hospital_info WHERE deleted = 0 AND hid = ? AND type = ? ORDER BY priority DESC", hid, stype)
	if err != nil {
		log.Printf("getHospitalInfos query failed:%v", err)
		return items
	}

	defer rows.Close()
	for rows.Next() {
		var item config.MediaInfo
		err := rows.Scan(&item.Id, &item.Img, &item.Dst, &item.Title,
			&item.Routername)
		if err != nil {
			log.Printf("getHospitalInfos scan failed:%v", err)
			continue
		}
		item.Type = 13 + stype
		items = append(items, &item)
	}
	return items
}

func getHospitalService(db *sql.DB, stype int64) []*config.PortalService {
	var infos []*config.PortalService
	var info config.PortalService
	info.Name = "患者服务"
	info.Items = getHospitalInfos(db, stype, hospitalService)
	if len(info.Items) > 0 {
		infos = append(infos, &info)
	}
	return infos
}

func getHospitalIntro(db *sql.DB, hid int64) []*config.MediaInfo {
	return getHospitalInfos(db, hid, hospitalIntro)
}

func getPortalService(db *sql.DB, stype int64) []*config.PortalService {
	if stype == 0 {
		return nil
		//return getOnlineService(db)
	}
	_, tid := getCustomPortal(db, stype)
	//online := getOnlineService(db)
	hospital := getHospitalService(db, tid)
	//infos := append(online, hospital...)
	return hospital
}

func getHospital(db *sql.DB, hid int64) []*config.MediaInfo {
	var infos []*config.MediaInfo
	var info config.MediaInfo
	err := db.QueryRow("SELECT img, title, dst FROM hospital WHERE id = ?", hid).Scan(&info.Img, &info.Title, &info.Dst)
	if err != nil {
		log.Printf("getHospital query failed:%v", err)
		return infos
	}
	infos = append(infos, &info)
	return infos
}

func getAdvertiseBanner(db *sql.DB, adtype int64) []*config.MediaInfo {
	var infos []*config.MediaInfo
	rows, err := db.Query("SELECT img, dst, id FROM advertise WHERE areaid = ? AND type = 0 AND online = 1 AND deleted = 0", adtype)
	if err != nil {
		log.Printf("getAdvertiseBanner query failed:%v", err)
		return infos
	}
	defer rows.Close()
	for rows.Next() {
		var info config.MediaInfo
		err := rows.Scan(&info.Img, &info.Dst, &info.Id)
		if err != nil {
			log.Printf("getAdvertiseBanner scan failed:%v", err)
			continue
		}
		info.Type = 1
		infos = append(infos, &info)
	}
	return infos
}

func getAdvertise(db *sql.DB, adtype int64) []*config.MediaInfo {
	var infos []*config.MediaInfo
	rows, err := db.Query("SELECT img, dst, id, name FROM advertise WHERE areaid = ? AND type = 1 AND online = 1 AND deleted = 0", adtype)
	if err != nil {
		log.Printf("getAdvertiseBanner query failed:%v", err)
		return infos
	}
	defer rows.Close()
	for rows.Next() {
		var info config.MediaInfo
		err := rows.Scan(&info.Img, &info.Dst, &info.Id, &info.Title)
		if err != nil {
			log.Printf("getAdvertiseBanner scan failed:%v", err)
			continue
		}
		info.Type = 1
		infos = append(infos, &info)
	}
	return infos
}

func getCustomPortal(db *sql.DB, id int64) (stype, tid int64) {
	err := db.QueryRow("SELECT type, tid FROM custom_portal WHERE id = ?", id).
		Scan(&stype, &tid)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("getCustomPortal failed:%v", err)
	}
	return
}

func getUnitTitle(db *sql.DB, cid int64) string {
	var title string
	err := db.QueryRow("SELECT u.name FROM custom_portal c, unit u WHERE c.unid = u.id AND c.id = ?", cid).Scan(&title)
	if err != nil {
		log.Printf("getUnitTitle failed:%v", err)
	}
	return title
}

func (s *server) GetPortalConf(ctx context.Context, in *common.CommRequest) (*config.PortalConfReply, error) {
	util.PubRPCRequest(w, "config", "GetPortalConf")
	log.Printf("GetPortalConf uid:%d type:%d subtype:%d id:%d", in.Head.Uid, in.Type,
		in.Subtype, in.Id)
	var banners []*config.MediaInfo
	var portaltype, adtype int64
	if in.Id != 0 {
		adtype = util.GetUnitArea(db, in.Id)
		portaltype = util.GetUnitPortal(db, in.Id)
	} else {
		portaltype = in.Type
		adtype = in.Subtype
	}
	if portaltype == 0 {
		banners = getBanners(db, portalBannerType, false, false)
	} else {
		stype, tid := getCustomPortal(db, portaltype)
		if stype == 0 {
			banners = getHospital(db, tid)
		}
	}
	if adtype != 0 {
		ads := getAdvertiseBanner(db, adtype)
		log.Printf("ads:%v", ads)
		banners = append(ads, banners...)

	}
	if portaltype == 0 {
		urbanservices := getUrbanServices(db, in.Head.Term)
		services := getPortalService(db, portaltype)
		util.PubRPCSuccRsp(w, "config", "GetPortalMenu")
		return &config.PortalConfReply{
			Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}, Banners: banners,
			Urbanservices: urbanservices, Services: services}, nil
	}
	_, tid := getCustomPortal(db, portaltype)
	hospitalintros := getHospitalIntro(db, tid)
	services := getPortalService(db, portaltype)
	unit := getUnitTitle(db, portaltype)
	util.PubRPCSuccRsp(w, "config", "GetPortalConf")
	return &config.PortalConfReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}, Banners: banners,
		Hospitalintros: hospitalintros, Services: services, Unit: unit}, nil
}

func isDstType(term, version int64) bool {
	if (term == 0 && version < 10) ||
		(term == 1 && version < 9) {
		return false
	}
	return true
}

func getBanners(db *sql.DB, btype int64, dstflag, dbgflag bool) []*config.MediaInfo {
	var infos []*config.MediaInfo
	query := fmt.Sprintf("SELECT img, dst, id, dsttype FROM banner WHERE deleted = 0 AND type = %d ", btype)
	if dbgflag {
		query += " AND (online = 1 OR dbg = 1) "
	} else {
		query += " AND online = 1 "
	}
	if !dstflag {
		query += " AND dsttype = 0 "
	}
	query += " ORDER BY priority DESC LIMIT 20"
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("select banner info failed:%v", err)
		return infos
	}
	for rows.Next() {
		var info config.MediaInfo
		err := rows.Scan(&info.Img, &info.Dst, &info.Id, &info.Type)
		if err != nil {
			log.Printf("scan failed:%v", err)
			continue
		}

		infos = append(infos, &info)

	}
	return infos
}

func getUrbanServices(db *sql.DB, term int64) []*config.MediaInfo {
	var infos []*config.MediaInfo
	rows, err := db.Query("SELECT title, img, dst, id FROM urban_service WHERE type = ?", term)
	if err != nil {
		log.Printf("getUrbanServices query failed:%v", err)
		return infos
	}
	defer rows.Close()
	for rows.Next() {
		var info config.MediaInfo
		err := rows.Scan(&info.Title, &info.Img, &info.Dst, &info.Id)
		if err != nil {
			log.Printf("getUrbanServices scan failed:%v", err)
			continue
		}
		infos = append(infos, &info)
	}
	return infos
}

func getRecommends(db *sql.DB) []*config.MediaInfo {
	var infos []*config.MediaInfo
	rows, err := db.Query("SELECT img, dst, id FROM recommend WHERE deleted = 0 ORDER BY priority DESC")
	if err != nil {
		log.Printf("getRecommends query failed:%v", err)
		return infos
	}
	defer rows.Close()
	for rows.Next() {
		var info config.MediaInfo
		err := rows.Scan(&info.Img, &info.Dst, &info.Id)
		if err != nil {
			log.Printf("getRecommends scan failed:%v", err)
			continue
		}
		infos = append(infos, &info)
	}
	return infos
}

func getCategoryTitleIcon(category int) (string, string) {
	switch category {
	default:
		return "智慧政务", "http://file.yunxingzh.com/ico_government_xxxh.png"
	case 2:
		return "交通出行", "http://file.yunxingzh.com/ico_traffic_xxxh.png"
	case 3:
		return "医疗服务", "http://file.yunxingzh.com/ico_medical_xxxh.png"
	case 4:
		return "网上充值", "http://file.yunxingzh.com/ico_recharge.png"
	}
}

func getServices(db *sql.DB) []*config.ServiceCategory {
	var infos []*config.ServiceCategory
	rows, err := db.Query("SELECT title, dst, category, sid, icon FROM service WHERE category != 0 AND deleted = 0 AND dst != '' ORDER BY category")
	if err != nil {
		log.Printf("query failed:%v", err)
		return infos
	}
	defer rows.Close()

	category := 0
	var srvs []*config.ServiceInfo
	for rows.Next() {
		var info config.ServiceInfo
		var cate int
		err := rows.Scan(&info.Title, &info.Dst, &cate, &info.Sid, &info.Icon)
		if err != nil {
			continue
		}

		if cate != category {
			if len(srvs) > 0 {
				var cateinfo config.ServiceCategory
				cateinfo.Title, cateinfo.Icon = getCategoryTitleIcon(category)
				cateinfo.Items = srvs[:]
				infos = append(infos, &cateinfo)
				srvs = srvs[len(srvs):]
			}
			category = cate
		}
		srvs = append(srvs, &info)
	}

	if len(srvs) > 0 {
		var cateinfo config.ServiceCategory
		cateinfo.Title, cateinfo.Icon = getCategoryTitleIcon(category)
		cateinfo.Items = srvs[:]
		infos = append(infos, &cateinfo)
	}

	return infos
}

func getEducationVideo(db *sql.DB) []*config.MediaInfo {
	var infos []*config.MediaInfo
	rows, err := db.Query("SELECT id, title, dst, click, img, source FROM education_video WHERE deleted = 0 ORDER BY id DESC")
	if err != nil {
		log.Printf("getEducationVideo query:%v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var info config.MediaInfo
		err := rows.Scan(&info.Id, &info.Title, &info.Dst, &info.Click,
			&info.Img, &info.Source)
		if err != nil {
			log.Printf("err:%v", err)
			continue
		}
		infos = append(infos, &info)
	}
	return infos
}

func (s *server) GetEducationVideo(ctx context.Context, in *common.CommRequest) (*config.EducationVideoReply, error) {
	util.PubRPCRequest(w, "config", "GetEducationVideo")
	infos := getEducationVideo(db)
	util.PubRPCSuccRsp(w, "config", "GetEducationVideo")
	return &config.EducationVideoReply{
		Head:  &common.Head{Retcode: 0, Uid: in.Head.Uid},
		Infos: infos}, nil
}

func getHospitalDepartment(db *sql.DB, hid int64) []*config.DepartmentCategoryInfo {
	var infos []*config.DepartmentCategoryInfo
	rows, err := db.Query("SELECT i.id, i.cid i.name, i.stime, i.click, c.name FROM hospital_department_info i, hospital_department_category c WHERE i.cid = c.id AND i.hid = ?", hid)
	if err != nil {
		log.Printf("getHospitalDepartment failed:%v", err)
		return infos
	}
	defer rows.Close()
	var departs []*config.DepartmentInfo
	var category int64
	var catename string
	for rows.Next() {
		var dinfo config.DepartmentInfo
		var cid int64
		var name string
		err = rows.Scan(&dinfo.Id, &cid, &dinfo.Name, &dinfo.Stime, &dinfo.Click,
			&name)
		if err != nil {
			continue
		}
		if cid != category {
			if len(departs) > 0 {
				var cateinfo config.DepartmentCategoryInfo
				cateinfo.Name = name
				cateinfo.Infos = departs[:]
				infos = append(infos, &cateinfo)
				departs = departs[len(departs):]
			}
			category = cid
			catename = name
		} else {
			departs = append(departs, &dinfo)
		}
	}
	if len(departs) > 0 {
		var cateinfo config.DepartmentCategoryInfo
		cateinfo.Name = catename
		cateinfo.Infos = departs
		infos = append(infos, &cateinfo)
	}
	return infos
}

func (s *server) GetHospitalDepartment(ctx context.Context, in *common.CommRequest) (*config.HospitalDepartmentReply, error) {
	util.PubRPCRequest(w, "config", "GetHospitalDepartment")
	infos := getHospitalDepartment(db, in.Type)
	util.PubRPCSuccRsp(w, "config", "GetHospitalDepartment")
	return &config.HospitalDepartmentReply{
		Head:  &common.Head{Retcode: 0, Uid: in.Head.Uid},
		Infos: infos}, nil
}

func getPortalDir(db *sql.DB, ptype int64, acname, apmac string) string {
	portaltype := util.GetPortalType(db, apmac)
	if ptype == util.PortalType {
		return util.GetPortalPath(db, acname, portaltype)
	}
	return util.GetLoginPath(db, acname, portaltype)
}

func (s *server) GetPortalDir(ctx context.Context, in *config.PortalDirRequest) (*config.PortalDirReply, error) {
	util.PubRPCRequest(w, "config", "GetPortalDir")
	dir := getPortalDir(db, in.Type, in.Acname, in.Apmac)
	util.PubRPCSuccRsp(w, "config", "GetPortalDir")
	log.Printf("GetPortalDir request:%v dir:%s", in, dir)
	return &config.PortalDirReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid},
		Dir:  dir}, nil
}

func extractTermContent(menulist []*config.PortalMenuInfo, flag bool) []*config.PortalMenuInfo {
	for i := 0; i < len(menulist); i++ {
		arr := strings.Split(menulist[i].Url, ";")
		if len(arr) >= 2 {
			if flag {
				menulist[i].Url = arr[0]
			} else {
				menulist[i].Url = arr[1]
			}
		}
	}
	return menulist
}

func (s *server) GetPortalContent(ctx context.Context, in *common.CommRequest) (*config.PortalContentReply, error) {
	util.PubRPCRequest(w, "config", "GetPortalContent")
	banners := getBanners(db, portalBannerV2Type, false, false)
	if in.Type != 0 {
		ads := getAdvertiseBanner(db, in.Type)
		banners = append(ads, banners...)
	}
	flag := util.IsWhiteUser(db, in.Head.Uid, util.PortalMenuDbgType)
	menulist := getPortalMenu(db, menuV2Type, flag)
	var termflag bool
	if in.Head.Term > 0 {
		termflag = true
	}
	menulist = extractTermContent(menulist, termflag)
	tablist := getPortalMenu(db, tabV2Type, flag)
	ads := getAdvertise(db, in.Type)
	util.PubRPCSuccRsp(w, "config", "GetPortalContent")
	return &config.PortalContentReply{
		Head:    &common.Head{Retcode: 0, Uid: in.Head.Uid},
		Banners: banners, Menulist: menulist, Tablist: tablist, Ads: ads}, nil
}

func (s *server) GetDiscovery(ctx context.Context, in *common.CommRequest) (*config.DiscoveryReply, error) {
	util.PubRPCRequest(w, "config", "GetDiscovery")
	dbgflag := util.IsWhiteUser(db, in.Head.Uid, util.BannerWhiteType)
	dstflag := isDstType(in.Head.Term, in.Head.Version)
	banners := getBanners(db, appBannerType, dstflag, dbgflag)
	urbanservices := getUrbanServices(db, in.Head.Term)
	recommends := getRecommends(db)
	services := getServices(db)
	util.PubRPCSuccRsp(w, "config", "GetDiscovery")
	return &config.DiscoveryReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}, Banners: banners,
		Urbanservices: urbanservices, Recommends: recommends,
		Services: services}, nil
}

func getMpArticle(db *sql.DB, id int64) *config.Article {
	var art config.Article
	err := db.QueryRow("SELECT title, img, dst, ctime FROM wx_mp_article WHERE wid = ? ORDER BY id DESC LIMIT 1", id).Scan(&art.Title, &art.Img, &art.Dst, &art.Ctime)
	if err != nil {
		return nil
	}
	return &art
}

func getMpwxInfo(db *sql.DB, wtype int64) []*config.MpwxInfo {
	var infos []*config.MpwxInfo
	rows, err := db.Query("SELECT id, name, icon, abstract, dst FROM wx_mp_info WHERE deleted = 0 AND type = ?", wtype)
	if err != nil {
		return infos
	}

	defer rows.Close()
	for rows.Next() {
		var info config.MpwxInfo
		err := rows.Scan(&info.Id, &info.Name, &info.Icon, &info.Abstract, &info.Dst)
		if err != nil {
			log.Printf("getMpwxInfo scan failed:%v", err)
			continue
		}
		info.Article = getMpArticle(db, info.Id)
		infos = append(infos, &info)
	}
	return infos
}

func getLocalMpwx(db *sql.DB) []*config.MpwxInfo {
	return getMpwxInfo(db, 0)
}
func getHotMpwx(db *sql.DB) []*config.MpwxInfo {
	return getMpwxInfo(db, 1)
}

func (s *server) GetMpwxInfo(ctx context.Context, in *common.CommRequest) (*config.MpwxInfoReply, error) {
	util.PubRPCRequest(w, "config", "GetDiscovery")
	local := getLocalMpwx(db)
	hot := getHotMpwx(db)
	util.PubRPCSuccRsp(w, "config", "GetDiscovery")
	return &config.MpwxInfoReply{
		Head:  &common.Head{Retcode: 0, Uid: in.Head.Uid},
		Local: local, Hot: hot,
	}, nil
}

func getMpwxArticle(db *sql.DB, stype, seq, num int64) []*config.Article {
	var infos []*config.Article
	query := fmt.Sprintf("SELECT id, title, img, dst, ctime, wid FROM wx_mp_article WHERE type = %d ", stype)
	if seq != 0 {
		query += fmt.Sprintf(" AND id < %d ", seq)
	}
	query += fmt.Sprintf(" ORDER BY id DESC LIMIT %d", num)
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("getMpwxArticle query failed:%v", err)
		return infos
	}

	defer rows.Close()
	for rows.Next() {
		var info config.Article
		var wid int64
		err := rows.Scan(&info.Id, &info.Title, &info.Img, &info.Dst, &info.Ctime,
			&wid)
		if err != nil {
			log.Printf("getMpwxArticle scan failed:%v", err)
			continue
		}
		info.Seq = info.Id
		if !isInnerWxid(wid) {
			info.Dst = extractArticleDst(info.Dst)
		}
		infos = append(infos, &info)
	}
	return infos
}

func extractArticleDst(dst string) string {
	pos := strings.Index(dst, "#wechat_redirect")
	if pos != -1 {
		return dst[0:pos]
	}
	return dst
}

func isInnerWxid(wid int64) bool {
	if wid == 1 || wid == 1299 {
		return true
	}
	return false
}

func (s *server) GetMpwxArticle(ctx context.Context, in *common.CommRequest) (*config.MpwxArticleReply, error) {
	util.PubRPCRequest(w, "config", "GetMpwxArticle")
	infos := getMpwxArticle(db, in.Type, in.Seq, in.Num)
	var hasmore int64
	if len(infos) >= int(in.Num) {
		hasmore = 1
	}
	util.PubRPCSuccRsp(w, "config", "GetMpwxArticle")
	return &config.MpwxArticleReply{
		Head:  &common.Head{Retcode: 0, Uid: in.Head.Uid},
		Infos: infos, Hasmore: hasmore,
	}, nil
}

func fetchPortalMenu(db *sql.DB, stype int64) []*config.PortalMenuInfo {
	var infos []*config.PortalMenuInfo
	rows, err := db.Query("SELECT id, icon, text, name, routername, url, priority, dbg, deleted FROM portal_menu WHERE type = ? ORDER BY priority DESC", stype)
	if err != nil {
		log.Printf("fetchPortalMenu query failed:%v", err)
		return infos
	}

	defer rows.Close()
	for rows.Next() {
		var info config.PortalMenuInfo
		err := rows.Scan(&info.Id, &info.Icon, &info.Text, &info.Name, &info.Routername,
			&info.Url, &info.Priority, &info.Dbg, &info.Deleted)
		if err != nil {
			log.Printf("fetchPortalMenu scan failed:%v", err)
			continue
		}
		infos = append(infos, &info)
	}
	return infos
}

func (s *server) FetchPortalMenu(ctx context.Context, in *common.CommRequest) (*config.MenuReply, error) {
	util.PubRPCRequest(w, "config", "FetchPortalMenu")
	infos := fetchPortalMenu(db, in.Type)
	util.PubRPCSuccRsp(w, "config", "FetchPortalMenu")
	return &config.MenuReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}, Infos: infos}, nil
}

func getRedirect(db *sql.DB, id int64) string {
	var dst string
	err := db.QueryRow("SELECT dst FROM redirect WHERE type = ?", id).Scan(&dst)
	if err != nil {
		log.Printf("getRedirect scan failed:%v", err)
		return dst
	}
	_, err = db.Exec("INSERT INTO redirect_cnt(type, ctime, cnt) VALUES (?, NOW(), 1) ON DUPLICATE KEY UPDATE cnt = cnt + 1", id)
	if err != nil {
		log.Printf("getRedirect add count failed:%v", err)
	}
	return dst
}

func (s *server) Redirect(ctx context.Context, in *common.CommRequest) (*config.RedirectReply, error) {
	util.PubRPCRequest(w, "config", "Redirect")
	dst := getRedirect(db, in.Id)
	util.PubRPCSuccRsp(w, "config", "Redirect")
	return &config.RedirectReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}, Dst: dst}, nil
}

func genReserveCode() int64 {
	r := util.Rand()
	n := (r % 9) + 1
	m := r % 100000
	return int64(n*100000 + m)
}

func getCodeCnt(db *sql.DB, code int64) int64 {
	var cnt int64
	err := db.QueryRow("SELECT COUNT(id) FROM reserve_info WHERE code = ?", code).
		Scan(&cnt)
	if err != nil {
		log.Printf("getCodeCnt failed:%v", err)
	}
	return cnt
}

func genCode(db *sql.DB) int64 {
	for i := 0; i < 10; i++ {
		code := genReserveCode()
		cnt := getCodeCnt(db, code)
		if cnt == 0 {
			return code
		}
	}
	return 0
}

func (s *server) GetReserveInfo(ctx context.Context, in *config.GetReserveRequest) (*config.ReserveInfoReply, error) {
	util.PubRPCRequest(w, "config", "GetReserveInfo")
	var name, phone string
	err := db.QueryRow("SELECT name, phone FROM reserve_info WHERE code = ?", in.Code)
	if err != nil {
		log.Printf("GetReserveInfo query failed:%d %v", in.Code, err)
		return &config.ReserveInfoReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	util.PubRPCSuccRsp(w, "config", "GetReserveInfo")
	return &config.ReserveInfoReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}, Name: name,
		Phone: phone}, nil
}

func (s *server) SubmitReserveInfo(ctx context.Context, in *config.ReserveRequest) (*config.ReserveReply, error) {
	util.PubRPCRequest(w, "config", "SubmitReserveInfo")
	code := genCode(db)
	if code == 0 {
		log.Printf("SubmitReserveInfo genCode failed phone:%s", in.Phone)
		return &config.ReserveReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil

	}
	res, err := db.Exec("INSERT IGNORE INTO reserve_info(name, phone, sid, reserve_date, btype, pillow, code, ctime) VALUES (?, ?, ?, ?, ?, ?, ?, NOW())",
		in.Name, in.Phone, in.Sid, in.Date, in.Btype, in.Pillow, code)
	if err != nil {
		log.Printf("SubmitReserveInfo insert failed:%v", err)
		return &config.ReserveReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	cnt, err := res.RowsAffected()
	if err != nil {
		log.Printf("SubmitReserveInfo get affected rows failed:%v", err)
		return &config.ReserveReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	if cnt == 0 {
		log.Printf("SubmitReserveInfo insert ignored phone:%s", in.Phone)
		return &config.ReserveReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	util.PubRPCSuccRsp(w, "config", "SubmitReserveInfo")
	return &config.ReserveReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}, Code: code}, nil
}

func (s *server) AddPortalMenu(ctx context.Context, in *config.MenuRequest) (*common.CommReply, error) {
	util.PubRPCRequest(w, "config", "AddPortalMenu")
	res, err := db.Exec("INSERT INTO portal_menu(type, icon, text, name, routername, url, priority, dbg, deleted, ctime) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NOW())",
		in.Info.Type, in.Info.Icon, in.Info.Text, in.Info.Name, in.Info.Routername,
		in.Info.Url, in.Info.Priority, in.Info.Dbg, in.Info.Deleted)
	if err != nil {
		log.Printf("AddPortalMenu query failed:%v", err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Printf("AddPortalMenu get id failed:%v", err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	util.PubRPCSuccRsp(w, "config", "AddPortalMenu")
	return &common.CommReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}, Id: id}, nil
}

func (s *server) ModPortalMenu(ctx context.Context, in *config.MenuRequest) (*common.CommReply, error) {
	util.PubRPCRequest(w, "config", "ModPortalMenu")
	query := fmt.Sprintf("UPDATE portal_menu SET mtime = NOW(), dbg = %d, deleted = %d ",
		in.Info.Dbg, in.Info.Deleted)
	if in.Info.Icon != "" {
		query += ", icon = '" + in.Info.Icon + "' "
	}
	if in.Info.Text != "" {
		query += ", text = '" + in.Info.Text + "' "
	}
	if in.Info.Name != "" {
		query += ", name = '" + in.Info.Name + "' "
	}
	if in.Info.Url != "" {
		query += ", url = '" + in.Info.Url + "' "
	}
	if in.Info.Priority != 0 {
		query += fmt.Sprintf(", priority = %d", in.Info.Priority)
	}
	if in.Info.Routername != "" {
		query += ", routername = '" + in.Info.Routername + "' "
	}
	query += fmt.Sprintf(" WHERE id = %d", in.Info.Id)
	_, err := db.Exec(query)
	if err != nil {
		log.Printf("ModPortalMenu query failed:%v", err)
		return &common.CommReply{
			Head: &common.Head{Retcode: 1, Uid: in.Head.Uid}}, nil
	}
	util.PubRPCSuccRsp(w, "config", "ModPortalMenu")
	return &common.CommReply{
		Head: &common.Head{Retcode: 0, Uid: in.Head.Uid}}, nil
}

func main() {
	lis, err := net.Listen("tcp", util.ConfigServerPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	w = util.NewNsqProducer()

	db, err = util.InitDB(false)
	if err != nil {
		log.Fatalf("failed to init db connection: %v", err)
	}
	db.SetMaxIdleConns(util.MaxIdleConns)
	kv = util.InitRedis()
	go util.ReportHandler(kv, util.ConfigServerName, util.ConfigServerPort)

	s := util.NewGrpcServer()
	config.RegisterConfigServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
