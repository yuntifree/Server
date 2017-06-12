package weixin

import (
	"fmt"
	"time"
)

const (
	wxType = 101
)

//GenOrderID generate order id
func GenOrderID(id int64) string {
	now := time.Now()
	return fmt.Sprintf("%03d%04d%02d%02d%02d%02d%02d%06d%11d",
		wxType, now.Year(), now.Month(), now.Day(), now.Hour(),
		now.Minute(), now.Second(), (now.UnixNano()%1e9)/1000,
		id)

}
