package util

import "math"

const (
	earthRadius = 6378137
	pi          = 3.14159265358979324
)

//Point point
type Point struct {
	Longitude, Latitude float64
}

//Gps2Bd  convert gps to baidu map
func Gps2Bd(p Point) (q Point) {
	return
}

func rad(d float64) float64 {
	return pi * d / 180.0
}

func getDistance(p1, p2 Point) int {
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
	z := math.Sqrt(x*x+y*y) - 0.00002*math.Sin(y*pi)
	theta := math.Atan2(y, x) - 0.000003*math.Cos(x*pi)
	q.Longitude = z * math.Cos(theta)
	q.Latitude = z * math.Sin(theta)
	return
}

//Tx2Bd convert tencent to baidu
func Tx2Bd(p Point) (q Point) {
	x := p.Longitude
	y := p.Latitude
	z := math.Sqrt(x*x+y*y) + 0.00002*math.Sin(y*pi)
	theta := math.Atan2(y, x) + 0.000003*math.Cos(x*pi)
	q.Longitude = z*math.Cos(theta) + 0.0065
	q.Latitude = z*math.Sin(theta) + 0.006
	return
}
