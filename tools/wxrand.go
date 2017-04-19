package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime/trace"
	"strings"

	"github.com/PuerkitoBio/goquery"
	simplejson "github.com/bitly/go-simplejson"
)

const (
	baseUrl    = "http://www.gsdata.cn/rank/single?id="
	contentUrl = "http://www.gsdata.cn/rank/recommendArticles?type=hot&id="
	imgUrl     = "http://img1.gsdata.cn/index.php/rank/getImageUrl?hash="
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

var ids = [][2]string{
	{"MxTuEex5N2T1Mtxa", "111531"},                  // OurDongguan
	{"MxTuQe25N2D1ctyaNnggO0O0O#O0O0O4", "1464726"}, // 东莞大喇叭
	{"MxTuYew5M2z1Qt5aNnwgO0O0O#O0O0O4", "1603497"}, // 东莞美食先锋队
	{"MxTuAe35O2T1ktxa", "107991"},                  // 东莞时间网
	{"MxTuUez5N2T1YtxaNnAgO0O0O#O0O0O4", "1535614"}, // 东莞生活君
	{"MxTuMe35M2z1ctza", "137373"},                  // 东莞美食大搜罗
	{"MxTuAe35O2T1gtxa", "107981"},                  // 东莞阳光网
	{"MxTuIe45M2j1gtO0O0Oa", "12828"},               // 东莞日报
	{"MxTuEex5N2T1Mtwa", "111530"},                  // 流行东莞
	{"MxTuUez5M2j1EtzaNnQgO0O0O#O0O0O4", "1532135"}, // 差评
	{"Mxjuce55O2A1O0O0OtO0O0Oa", "2798"},            // 虎嗅网
	{"MxTuUe05O2D1QtyaMnwgO0O0O#O0O0O4", "1548423"}, // 超高能E姐
	{"MxjuceO0O0O5", "27"},                          // Vista看天下
	{"MxzuYe35O2D1Ut5a", "367859"},                  // 同道大叔
	{"MxTuAe35N2T1Ytya", "107562"},                  // 人民日报
	{"MxTuce25M2z1gtO0O0Oa", "17638"},               // 深夜发媸
	{"NxTuQez5N2T1Itwa", "543520"},                  // 二更食堂
	{"MxTuQez5", "143"},                             // 十点读书
	{"MxTuUez5M2j1EtzaNnQgO0O0O#O0O0O4", "1532135"},
	{"MxzuMe25O2T1ItO0O0Oa", "33692"},
	{"Mxjukey5N2Q1O0O0OtO0O0Oa", "2925"},
	{"MxzuUe25M2A1O0O0OtO0O0Oa", "3560"},
	{"MxjuMe45M2Q1O0O0OtO0O0Oa", "2381"},
	{"MxTuYex5N2D1Ut0aMnwgO0O0O#O0O0O4", "1614543"},
	{"Mxjuce55O2A1O0O0OtO0O0Oa", "2798"},
	{"MxzuYex5O2A1O0O0OtO0O0Oa", "3618"},
	{"NxDuce35M2D1kt3aOnQgO0O0O#O0O0O4", "4770979"},
	{"MxTuEex5M2T1At3a", "111107"},
	{"MxjuQe45", "248"},
	{"MxTuUe15N2D1Qt0aNnwgO0O0O#O0O0O4", "1554447"},
	{"MxTuUe15M2T1EtxaOnQgO0O0O#O0O0O4", "1551119"},
	{"NxTuke55N2z1ctO0O0Oa", "59977"},
	{"NxjuAe55M2z1ctxaOnQgO0O0O#O0O0O4", "6093719"},
	{"MxTuUe25M2z1Et4aNnggO0O0O#O0O0O4", "1563186"},
	{"MxTuAe35N2T1Ytya", "107562"},
	{"NxjuYew5M2T1ktO0O0Oa", "66019"},
	{"MxjuIe35", "227"},
	{"MxTuEe35N2j1MtxaNnQgO0O0O#O0O0O4", "1176315"},
	{"MxjuQey5", "242"},
	{"MxzuYe35O2D1Ut5a", "367859"},
	{"MxTuUe15N2D1EtxaMnggO0O0O#O0O0O4", "1554112"},
	{"MxTuMez5", "133"},
	{"NxTuke55N2z1EtO0O0Oa", "59971"},
	{"MxTuUe15M2z1ktxaMnQgO0O0O#O0O0O4", "1553911"},
	{"MxTuMe35N2D1gtxa", "137481"},
	{"MxzuIe25M2T1ktO0O0Oa", "32619"},
	{"Nxjuce05N2j1UtO0O0Oa", "67465"},
	{"MxTuAex5M2D1Et1a", "101015"},
	{"NxjuQey5N2T1ctO0O0Oa", "64257"},
	{"MxTuQe25M2T1At3aMnAgO0O0O#O0O0O4", "1461070"},
	{"NxzuEe15N2j1gt2a", "715686"},
	{"MxTuUe15N2j1YtzaMnAgO0O0O#O0O0O4", "1556630"},
	{"NxTuMez5N2T1ctO0O0Oa", "53357"},
	{"NxTuMey5O2D1gtO0O0Oa", "53288"},
	{"MxTuIex5N2T1AtyaMnwgO0O0O#O0O0O4", "1215023"},
	{"OxDuMey5M2T1EtO0O0Oa", "83211"},
	{"MxTuUe25M2D1UtwaMnggO0O0O#O0O0O4", "1560502"},
	{"MxTuUe05M2z1gt2aNnggO0O0O#O0O0O4", "1543866"},
	{"NxTuMey5O2T1MtO0O0Oa", "53293"},
	{"NxTuMez5M2T1ItO0O0Oa", "53312"},
	{"NxjuYe25M2z1MtO0O0Oa", "66633"},
	{"MxTuUe15N2T1ct5aNnggO0O0O#O0O0O4", "1555796"},
	{"MxTuAey5M2T1Et1aOnDgQ#O0O0O4", "10211584"},
	{"MxTuQe35N2D1ItwaNnggO0O0O#O0O0O4", "1474206"},
	{"NxDuYex5M2z1gt2a", "461386"},
	{"Mxjucew5M2w1O0O0OtO0O0Oa", "2703"},
	{"MxTuUe15N2T1ct4aOnAgO0O0O#O0O0O4", "1555788"},
	{"OxDuMe45N2D1ktO0O0Oa", "83849"},
	{"MxTuEex5N2T1Mtxa", "111531"},
	{"NxzuMe35N2j1Mt1a", "737635"},
	{"MxTuUe15N2D1UtxaMnwgO0O0O#O0O0O4", "1554513"},
	{"NxTuMey5N2T1ItO0O0Oa", "53252"},
	{"NxTuke05N2z1AtO0O0Oa", "59470"},
	{"MxjuQez5O2Q1O0O0OtO0O0Oa", "2439"},
	{"MxzuEew5O2A1O0O0OtO0O0Oa", "3108"},
	{"Nxjuce45M2z1ItO0O0Oa", "67832"},
	{"MxTuUe55N2z1MtxaMnQgO0O0O#O0O0O4", "1597311"},
	{"MxTuUe05M2j1gtwaMnAgO0O0O#O0O0O4", "2542800"},
	{"MxTuQe25M2z1gt2aMnQgO0O0O#O0O0O4", "1463861"},
	{"MxTuQex5N2T1Ut5a", "141559"},
	{"OxDuge55N2w1O0O0OtO0O0Oa", "8897"},
	{"NxjuUez5M2D1gtO0O0Oa", "65308"},
	{"MxTuAe25M2D1Ut2a", "106056"},
	{"MxTukez5", "193"},
	{"MxTuMe25O2T1It3a", "136927"},
	{"OxDuMe45N2j1ktO0O0Oa", "83869"},
	{"NxjuUe45M2A1O0O0OtO0O0Oa", "6580"},
	{"NxjuIex5N2j1At4a", "621608"},
	{"NxjuYe35N2j1ctO0O0Oa", "66767"},
	{"MxTuUe15N2z1AtwaNnggO0O0O#O0O0O4", "1557006"},
	{"MxTuQez5", "143"},
	{"MxTuUe25N2D1ItwaNnggO0O0O#O0O0O4", "1564206"},
	{"MxTuce55N2T1gtO0O0Oa", "17958"},
	{"MxTuUez5O2D1gtzaNnQgO0O0O#O0O0O4", "1538835"},
	{"MxTuAeO0O0O5", "10"},
	{"NxDuEey5O2D1ct2a", "412876"},
	{"OxDuMe45N2D1MtO0O0Oa", "83843"},
	{"MxTuIe25N2z1kt1aOnQgO0O0O#O0O0O4", "1267959"},
	{"NxTuAe25N2T1Yt4a", "506568"},
	{"MxzuYex5M2w1O0O0OtO0O0Oa", "3613"},
	{"NxTuAez5O2D1YtyaOnAgO0O0O#O0O0O4", "5038628"},
	{"MxTuQe25N2T1Ut3aOnAgO0O0O#O0O0O4", "1465578"},
	{"OxTuQey5N2j1Atya", "942602"},
	{"Mxzugez5N2T1ctO0O0Oa", "38357"},
	{"MxTuUe25N2D1kt1aOnQgO0O0O#O0O0O4", "1564959"},
	{"MxTuMe15N2z1MtxaMnQgO0O0O#O0O0O4", "1357311"},
	{"MxTuUe35M2j1Et0aMnQgO0O0O#O0O0O4", "1572141"},
	{"MxTuYex5N2D1YtzaMnwgO0O0O#O0O0O4", "1614633"},
	{"MxTuQez5M2z1QtwaNnggO0O0O#O0O0O4", "1433406"},
	{"MxjuQew5N2g1O0O0OtO0O0Oa", "2406"},
	{"NxTuAe45O2A1O0O0OtO0O0Oa", "5088"},
	{"MxzuMew5N2T1ctO0O0Oa", "33057"},
	{"MxTuAe25O2T1ktwa", "106990"},
	{"MxTuUe15N2T1ktyaNnQgO0O0O#O0O0O4", "1555925"},
	{"MxTuUey5N2z1EtwaNnggO0O0O#O0O0O4", "1527106"},
	{"NxDucey5N2T1gt4aNnQgO0O0O#O0O0O4", "4725885"},
	{"NxjuAe15M2D1Et5aOnAgO0O0O#O0O0O4", "6050198"},
	{"MxjuUe35O2D1gtO0O0Oa", "25788"},
	{"MxTuIey5N2j1Yt3a", "122667"},
	{"MxTuAe05N2j1gtxa", "104681"},
	{"NxjuUe05N2z1QtO0O0Oa", "65474"},
	{"MxTuEe45M2z1gt1aNnggO0O0O#O0O0O4", "1183856"},
	{"OxTuIe55N2w1O0O0OtO0O0Oa", "9297"},
	{"MxjuUe35O2D1gtO0O0Oa", "25788"},
	{"MxTuce05O2D1YtO0O0Oa", "17486"},
	{"NxDuMe05M2z1UtyaMnQgO0O0O#O0O0O4", "4343521"},
	{"NxzuIe15M2j1kt0a", "725294"},
	{"Mxzuke15O2T1ItxaNnggO0O0O#O0O0O4", "3959216"},
	{"MxTuUe05N2T1Et1aMnwgO0O0O#O0O0O4", "1545153"},
	{"Nxjuce05N2j1EtO0O0Oa", "67461"},
	{"NxTuAe05N2T1ct2aMnQgO0O0O#O0O0O4", "5045761"},
	{"Mxzuge25M2D1gtya", "386082"},
	{"NxDuAe55N2T1Atxa", "409501"},
	{"MxTuUez5M2j1kt2aMnQgO0O0O#O0O0O4", "1532961"},
	{"MxTuUe55O2T1Mt5aMnAgO0O0O#O0O0O4", "1599390"},
	{"MxTuAe15M2j1kt1a", "105295"},
	{"NxjuAe45M2D1UtyaMnQgO0O0O#O0O0O4", "6080521"},
	{"Nxjuce25O2D1EtO0O0Oa", "67681"},
	{"MxTuIex5O2D1gtyaOnAgO0O0O#O0O0O4", "1218828"},
	{"MxTuYex5N2T1Yt5aMnggO0O0O#O0O0O4", "1615692"},
	{"MxTuIex5O2D1gtyaNnwgO0O0O#O0O0O4", "1218827"},
	{"MxTuUez5M2D1Ut2aMnQgO0O0O#O0O0O4", "1530561"},
	{"NxjuQe25M2z1YtwaMnAgO0O0O#O0O0O4", "6463600"},
	{"MxTuUe35M2j1Et2aMnwgO0O0O#O0O0O4", "1572163"},
	{"MxjuEey5N2T1QtwaMnjgc#O0O0O4", "21254027"},
	{"MxTuQe25O2D1Et3aOnAgO0O0O#O0O0O4", "1468178"},
	{"MxTuQe25O2D1Mt4aMnggO0O0O#O0O0O4", "1468382"},
	{"NxTuEez5M2z1Itza", "513323"},
	{"MxjuEez5O2D1gtya", "213882"},
	{"NxDuUe05O2T1gtO0O0Oa", "45498"},
	{"MxjuEez5O2D1Et0a", "213814"},
	{"OxTuIez5N2g1O0O0OtO0O0Oa", "9236"},
	{"NxDuEey5O2T1Mt4a", "412938"},
	{"MxTuIew5M2j1Ut5a", "120259"},
	{"MxTuQe35M2z1UtzaMnAgO0O0O#O0O0O4", "1473530"},
	{"Nxjuce45N2z1gtO0O0Oa", "67878"},
	{"Nxjuce45N2j1ktO0O0Oa", "67869"},
	{"OxDuge05O2T1Mt3a", "884937"},
	{"MxzuUey5M2D1Ut4aNnwgO0O0O#O0O0O4", "3520587"},
	{"MxTuce05M2T1ktO0O0Oa", "17419"},
	{"NxjuAe05M2j1ktyaNnggO0O0O#O0O0O4", "6042926"},
	{"Nxjuce45N2z1ctO0O0Oa", "67877"},
	{"MxTuAe35O2T1gtxa", "107981"},
	{"MxTuEe45M2z1Mt5aMnwgO0O0O#O0O0O4", "1183393"},
	{"MxTuEex5N2T1Mtwa", "111530"},
	{"MxTuMe35M2z1ctza", "137373"},
	{"MxTuIe45M2j1gtO0O0Oa", "12828"},
	{"MxTuUez5N2T1YtxaNnAgO0O0O#O0O0O4", "1535614"},
	{"MxTuQe25N2D1ctyaNnggO0O0O#O0O0O4", "1464726"},
	{"MxTuAe35O2T1ktxa", "107991"},
	{"MxjuYew5O2T1Ytza", "260963"},
	{"MxTuQex5M2j1ct4aNnQgO0O0O#O0O0O4", "1412785"},
	{"NxjuAe55N2z1kt1aNnggO0O0O#O0O0O4", "6097956"},
	{"NxjuEew5M2T1Et0aOnAgO0O0O#O0O0O4", "6101148"},
}

var limit chan int

func getWxInfo(id [2]string, ch chan string) WxInfo {
	limit <- 1
	var wxData WxInfo
	client := &http.Client{}
	req, err := http.NewRequest("GET", baseUrl+id[0], nil)
	req.Header.Add("User-Agent", "Mozilla/5.0 (Linux; Android 6.0; Nexus 5 Build/MRA58N) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/46.0.2490.76 Mobile Safari/537.36")
	resp, err := client.Do(req)
	defer resp.Body.Close()

	d, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Printf("fetch url failed:%v", err)
		return wxData
	}

	div := d.Find(".number-txt")
	wxData.Id = div.Find("#wx_name").Text()
	wxData.Name = d.Find(".number-title #wx_nickname").Text()
	wxData.Desc = div.Find(".wx-sp").Eq(1).Find(".sp-txt").Text()
	wxData.HeadUrl = "http://open.weixin.qq.com/qr/code/?username=" + wxData.Id

	ch <- fmtContent(wxData, getWxContent(id[1]))
	<-limit

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

	ret, _ := js.Get("status").String()
	if ret != "OK" {
		log.Printf("status not ok: %v", ret)
		return nil
	}
	// get article and images, 取得最近一篇文章
	// TODO: check empty items
	artlist, _ := js.Get("result").Get("items").Array()
	artlen := len(artlist)
	items := make([]WxContent, artlen)

	for i := 0; i < artlen; i++ {
		article := js.Get("result").Get("items").GetIndex(i)
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
		picurl, _ := article.Get("picurl").String()

		items[i].Image = getImageUrl(picurl)
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

	js, err := simplejson.NewFromReader(resp.Body)
	if err != nil {
		log.Printf("parse url json failed:%v", err)
		return ""
	}

	ret, _ := js.Get("url").String()

	return ret
}

func checkerr(e error) {
	if e != nil {
		panic(e)
	}
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

	limit = make(chan int, 2)
	chs := make([]chan string, len(ids))
	for i, v := range ids {
		chs[i] = make(chan string, 2)
		go getWxInfo(v, chs[i])
	}
	content := ""
	for _, ch := range chs {
		content = <-ch
		w.WriteString(content)
		fmt.Println(content)
	}
	w.Flush()
}
