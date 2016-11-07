package util

import (
	"fmt"
	"math"
)

const (
	earthRadius = 6378137
	pi          = 3.14159265358979324
	xPi         = 3.14159265358979324 * 3000.0 / 180.0
)

//Point point
type Point struct {
	Longitude, Latitude float64
}

func (p *Point) String() {
	fmt.Sprintf("{%f, %f}", p.Longitude, p.Latitude)
}

func rad(d float64) float64 {
	return pi * d / 180.0
}

func outOfChina(p Point) bool {
	if p.Longitude < 72.004 || p.Longitude > 137.8347 {
		return true
	}
	if p.Latitude < 0.8293 || p.Latitude > 55.8271 {
		return true
	}
	return false
}

func transLatitude(x, y float64) float64 {
	ret := -100.0 + 2.0*x + 3.0*y + 0.2*y*y + 0.1*x*y + 0.2*math.Sqrt(math.Abs(x))
	ret += (20.0*math.Sin(6.0*x*pi) + 20.0*math.Sin(2.0*x*pi)) * 2.0 / 3.0
	ret += (20.0*math.Sin(y*pi) + 40.0*math.Sin(y/3.0*pi)) * 2.0 / 3.0
	ret += (160.0*math.Sin(y/12.0*pi) + 320*math.Sin(y*pi/30.0)) * 2.0 / 3.0
	return ret
}

func transLongitude(x, y float64) float64 {
	ret := 300.0 + x + 2.0*y + 0.1*x*x + 0.1*x*y + 0.1*math.Sqrt(math.Abs(x))
	ret += (20.0*math.Sin(6.0*x*pi) + 20.0*math.Sin(2.0*x*pi)) * 2.0 / 3.0
	ret += (20.0*math.Sin(x*pi) + 40.0*math.Sin(x/3.0*pi)) * 2.0 / 3.0
	ret += (150.0*math.Sin(x/12.0*pi) + 300*math.Sin(x*pi/30.0)) * 2.0 / 3.0
	return ret
}

func delta(p1 Point) (p2 Point) {
	//Krasovsky 1940
	// a = 6378245.0 1/f = 298.3
	// b = a * （1 - f)
	// ee = (a^2 - b^2) / a^2
	a := 6378245.0
	ee := 0.00669342162296594323
	dLat := transLatitude(p1.Longitude-105.0, p1.Latitude-35.0)
	dLon := transLongitude(p1.Longitude-105.0, p1.Latitude-35.0)
	radLat := p1.Latitude / 180.0 * pi
	magic := math.Sin(radLat)
	magic = 1 - ee*magic*magic
	sqrtMagic := math.Sqrt(magic)
	p2.Latitude = (dLat * 180.0) / ((a * (1 - ee)) / (magic * sqrtMagic) * pi)
	p2.Longitude = (dLon * 180.0) / (a / sqrtMagic * math.Cos(radLat) * pi)
	return
}

//Gps2Tx convert gps to tencent map
func Gps2Tx(p1 Point) (p2 Point) {
	if outOfChina(p1) {
		return p1
	}
	p := delta(p1)
	p2.Longitude = p1.Longitude + p.Longitude
	p2.Latitude = p1.Latitude + p.Latitude
	return
}

//Tx2Gps convert tencent to gps map
func Tx2Gps(p1 Point) (p2 Point) {
	if outOfChina(p1) {
		return p1
	}
	p := delta(p1)
	p2.Longitude = p1.Longitude - p.Longitude
	p2.Latitude = p1.Latitude - p.Latitude
	return
}

//Tx2GpsExact convert tencent to gps map exactly
func Tx2GpsExact(p1 Point) (p2 Point) {
	initDelta := 0.01
	threshold := 0.000000001
	dLat := initDelta
	dLon := initDelta
	mLat := p1.Latitude - dLat
	mLon := p1.Longitude - dLon
	pLat := p1.Latitude + dLat
	pLon := p1.Longitude + dLon
	wgsLat, wgsLon, i := 0.0, 0.0, 0
	for {
		i++
		wgsLat = (mLat + pLat) / 2
		wgsLon = (mLon + pLon) / 2
		p := Point{Latitude: wgsLat, Longitude: wgsLon}
		tmp := Gps2Tx(p)
		dLat = tmp.Latitude - p1.Latitude
		dLon = tmp.Longitude - p1.Longitude
		if math.Abs(dLat) < threshold && math.Abs(dLon) < threshold {
			break
		}

		if dLat > 0 {
			pLat = wgsLat
		} else {
			mLat = wgsLat
		}

		if i > 1000 {
			break
		}
	}
	p2.Longitude = wgsLon
	p2.Latitude = wgsLat
	return
}

//Gps2Bd convert gps to baidu map
func Gps2Bd(p1 Point) (p2 Point) {
	p2 = Tx2Bd(Gps2Tx(p1))
	return
}

//GetDistance calc distance between two points
func GetDistance(p1, p2 Point) int {
	radLat1 := rad(p1.Latitude)
	radLat2 := rad(p2.Latitude)
	a := radLat1 - radLat2
	b := rad(p1.Longitude) - rad(p2.Longitude)
	s := 2 * math.Asin(math.Sqrt(math.Pow(math.Sin(a/2), 2)+math.Cos(radLat1)*math.Cos(radLat2)*math.Pow(math.Sin(b/2), 2)))
	return int(s * earthRadius)
}

//Bd2Tx convert baidu to tencent
func Bd2Tx(p Point) (q Point) {
	x := p.Longitude - 0.0065
	y := p.Latitude - 0.006
	z := math.Sqrt(x*x+y*y) - 0.00002*math.Sin(y*xPi)
	theta := math.Atan2(y, x) - 0.000003*math.Cos(x*xPi)
	q.Longitude = z * math.Cos(theta)
	q.Latitude = z * math.Sin(theta)
	return
}

//Tx2Bd convert tencent to baidu
func Tx2Bd(p Point) (q Point) {
	x := p.Longitude
	y := p.Latitude
	z := math.Sqrt(x*x+y*y) + 0.00002*math.Sin(y*xPi)
	theta := math.Atan2(y, x) + 0.000003*math.Cos(x*xPi)
	q.Longitude = z*math.Cos(theta) + 0.0065
	q.Latitude = z*math.Sin(theta) + 0.006
	return
}
