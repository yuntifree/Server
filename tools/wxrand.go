package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"runtime/trace"
	"strings"

	"github.com/PuerkitoBio/goquery"
	simplejson "github.com/bitly/go-simplejson"
)

const (
	baseUrl    = "http://www.gsdata.cn/rank/wxdetail?wxname="
	contentUrl = "http://www.gsdata.cn/rank/toparc?sort=-1&wxname="
	imgUrl     = "http://img1.gsdata.cn/index.php/rank/getimageurl?callback=_&hash="
)

//WxData information for wxData
type WxInfo struct {
	Id, // wxid
	Name, // 微信名称
	Desc, // 公众号描述
	HeadUrl string // 公众号头像
}

type WxContent struct {
	Title, // 文章标题
	Desc, // 文章描述
	Image, // 文章图片：aliyun
	PostTime, // 发布时间
	PostUrl, // 文章地址
	HomeUrl string // 公众号主页，点击进入公众号用
}

var ids = []string{
	"America_hq          ",
	"apptoday            ",
	"BAIKE0769           ",
	"bamaying            ",
	"bfaner              ",
	"bixiakaifanle       ",
	"bmsh_dg             ",
	"Buyerkey            ",
	"cbn_tglj            ",
	"cctvnewscenter      ",
	"cctvyscj            ",
	"chaping321          ",
	"chetuteng-com       ",
	"chuangyezuiqianxian ",
	"cmm445              ",
	"cmo1967             ",
	"coollabs            ",
	"cypuzi              ",
	"dazhengjing         ",
	"dg2050              ",
	"dgdalaba            ",
	"dgrb22008278        ",
	"dgzhcs              ",
	"dianyingbake        ",
	"DingXiangYiSheng    ",
	"DJ00123987          ",
	"DouguoCom           ",
	"dsmovie             ",
	"duhaoshu            ",
	"duliyumovie         ",
	"dxloveplay          ",
	"dyp833              ",
	"eemovie             ",
	"fengyuhuangshan     ",
	"fgzadmin            ",
	"foodvideo           ",
	"Food_Lab            ",
	"gaichezhi           ",
	"gqtzy2014           ",
	"guanzhutvb          ",
	"gudianshucheng      ",
	"Guokr42             ",
	"gushequ             ",
	"gxjhshys            ",
	"gzwcjs              ",
	"heimagongshe        ",
	"hereinuk            ",
	"hibetterme          ",
	"hkstocks            ",
	"huazhuangshimk      ",
	"huxiu_com           ",
	"Iamasinger_hntv     ",
	"ibaoman             ",
	"ichuangyebang       ",
	"icuiyutao           ",
	"idongche            ",
	"ifeng-news          ",
	"iiiher              ",
	"ilianyue            ",
	"iModifiedCar        ",
	"iphone-apple-ipad   ",
	"iyourcar            ",
	"jiaosushi           ",
	"jiedawang-zhi       ",
	"kawa01              ",
	"kejimx              ",
	"kidsfood            ",
	"kongfuf             ",
	"lang-club           ",
	"lengtoo             ",
	"lengxiaohua2012     ",
	"LinkedIn-China      ",
	"liuxb0929           ",
	"lol_helper          ",
	"lwwuwuwu            ",
	"m-a-dmen            ",
	"mh4565              ",
	"miaofafoyin520      ",
	"microhugo           ",
	"mimeng7             ",
	"mofzpy              ",
	"movieiii            ",
	"movpuzi             ",
	"MusicClassic        ",
	"mymoney888          ",
	"newcaimi            ",
	"newWhatYouNeed      ",
	"OurDongguan         ",
	"popdgwx             ",
	"popland100          ",
	"Pydp888             ",
	"qqmusic             ",
	"QQ_shijuezhi        ",
	"rmrbwx              ",
	"Rockerfm            ",
	"rosemarytv          ",
	"rzt317              ",
	"sdimov              ",
	"shejizone           ",
	"shen1dian           ",
	"shenyefachi         ",
	"shenyeshitang521    ",
	"shicishijie         ",
	"shicitiandi         ",
	"shiguangmm01        ",
	"shudanlaile         ",
	"sisterinlaw         ",
	"sun0769-com         ",
	"super_misse         ",
	"SZLife0755          ",
	"tancaijing          ",
	"taolumusic          ",
	"tcdy007             ",
	"thepoemforyou       ",
	"timedg              ",
	"vipidy              ",
	"vistaweek           ",
	"v_movier            ",
	"wangyixinwen163     ",
	"wanzilove1218       ",
	"webthinking         ",
	"weixinlukuang       ",
	"weloveuk            ",
	"wenyijcc            ",
	"whbzh520            ",
	"Win_in_Japan        ",
	"witheating          ",
	"wonderful_picture   ",
	"woshitongdao        ",
	"wow36kr             ",
	"wudaoone            ",
	"wuxiaobopd          ",
	"xfd0769             ",
	"xiachufang          ",
	"xiami_music         ",
	"xiaogdnw            ",
	"xinli01             ",
	"xxbmm123            ",
	"yesdg0769           ",
	"yingdanlaile        ",
	"yixuejiezazhi       ",
	"youshucc            ",
	"youthmba            ",
	"yuedu58             ",
	"yummydg             ",
	"yunyinyue163        ",
	"zcdq520             ",
	"zg5201949           ",
	"zhangzhaozhong45    ",
	"zhanhao668          ",
	"zimeiti-sogou       ",
	"zmscook             ",
	"zsnc-ok             ",
	"zuiheikeji          ",
	"zuofaniii           ",
}

var limit chan int

func getWxInfo(id string, ch chan string) WxInfo {
	limit <- 1
	var wxData WxInfo
	client := &http.Client{}
	req, err := http.NewRequest("GET", baseUrl+id, nil)
	req.Header.Add("User-Agent", "Mozilla/5.0 (Linux; Android 6.0; Nexus 5 Build/MRA58N) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/46.0.2490.76 Mobile Safari/537.36")
	resp, err := client.Do(req)
	defer resp.Body.Close()

	d, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Printf("fetch url failed:%v", err)
		return wxData
	}

	div := d.Find(".wxDetail-top")
	wxData.Id = id
	wxData.Name = d.Find(".wxDetail-name label").Text()
	wxData.Desc = div.Find(".info-li p").Last().Text()
	wxData.HeadUrl = "http://open.weixin.qq.com/qr/code/?username=" + wxData.Id

	<-limit
	ch <- fmtContent(wxData, getWxContent(id))

	return wxData
}

func getWxContent(id string) []WxContent {
	client := &http.Client{}
	req, err := http.NewRequest("GET", contentUrl+id, nil)
	req.Header.Add("X-Requested-With", "XMLHttpRequest")

	if err != nil {
		log.Printf("fetch content failed:%v", err)
		return nil
	}

	resp, err := client.Do(req)
	defer resp.Body.Close()

	js, err := simplejson.NewFromReader(resp.Body)
	if err != nil {
		log.Printf("parse rspbody failed:%v", err)
		return nil
	}

	ret, _ := js.Get("error").Int()
	if ret != 0 {
		log.Printf("status not ok: %v", ret)
		return nil
	}
	// get article and images, 取得最近一篇文章
	// TODO: check empty items
	artlist, _ := js.Get("data").Array()
	artlen := len(artlist)
	items := make([]WxContent, artlen)

	for i := 0; i < artlen; i++ {
		article := js.Get("data").GetIndex(i)
		// post infos
		items[i].PostTime, _ = article.Get("posttime").String()
		postUrl, _ := article.Get("url").String()
		items[i].PostUrl = postUrl

		// 获取__biz 得到主页url
		start := strings.Index(postUrl, "__biz=") + 6
		end := strings.Index(postUrl, "&mid=")

		items[i].HomeUrl = "https://mp.weixin.qq.com/mp/profile_ext?action=home&scene=116&__biz=" + string(postUrl[start:end]) + "#wechat_redirect"
		items[i].Title, _ = article.Get("title").String()
		items[i].Desc, _ = article.Get("content").String()

		// fetch aliyun imageurl
		picurl, _ := article.Get("data-hash").String()

		items[i].Image = getImageUrl(url.QueryEscape(picurl))
		if items[i].Image == "" {
			items[i].Image, _ = article.Get("picurl").String()
		}
	}

	return items
}

func getImageUrl(qurl string) string {
	client := &http.Client{}
	req, err := http.NewRequest("GET", imgUrl+qurl, nil)

	if err != nil {
		log.Printf("fetch image url failed:%v", err)
		return ""
	}

	resp, err := client.Do(req)
	defer resp.Body.Close()

	str, err := ioutil.ReadAll(resp.Body)
	if checkerr(err) {
		return ""
	}

	t := string(str)
	pos := strings.Index(t, "(")
	if pos == -1 {
		pos = 2
	}
	js, err := simplejson.NewJson(str[pos+1 : len(str)-1])
	if err != nil {
		log.Printf("parse imageurl json failed:%v", err)
		log.Printf("%v", string(str))
		return ""
	}

	ret, _ := js.Get("url").String()

	return ret
}

func checkerr(e error) bool {
	if e != nil {
		panic(e)
		return true
	}
	return false
}

func fmtHeader() string {
	return fmt.Sprintln("wxid,wx_name,wx_desc,wx_head,wx_home,title,desc,img,url,posttime")
}

func fmtContent(wxInfo WxInfo, wxList []WxContent) string {
	ret := ""
	for _, item := range wxList {
		ret += fmt.Sprintf("%v||%v||%v||%v||%v||%v||%v||%v||%v||%v\n", wxInfo.Id, wxInfo.Name, wxInfo.Desc, wxInfo.HeadUrl,
			item.HomeUrl, item.Title, item.Desc, item.Image, item.PostUrl, item.PostTime)
	}
	return ret
}

func main() {
	g, err := os.Create("trace.out")
	if err != nil {
		panic(err)
	}
	defer g.Close()
	err = trace.Start(g)
	if err != nil {
		panic(err)
	}
	defer trace.Stop()

	f, err := os.Create("wxData.csv")
	checkerr(err)
	defer f.Close()

	fmt.Println(fmtHeader())
	w := bufio.NewWriter(f)

	limit = make(chan int, 8)
	chs := make([]chan string, len(ids))
	for i, v := range ids {
		chs[i] = make(chan string, 2)
		str := strings.Replace(v, " ", "", -1)
		go getWxInfo(str, chs[i])
	}
	content := ""
	for _, ch := range chs {
		content = <-ch
		w.WriteString(content)
		fmt.Println(content)
	}
	w.Flush()
}
