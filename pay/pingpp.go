package pay

import (
	"encoding/json"

	pingpp "github.com/pingplusplus/pingpp-go/pingpp"
	"github.com/pingplusplus/pingpp-go/pingpp/charge"
)

const (
	pingppAppid = "app_z90KC0fzT0SCS8un"
)

//GetPingPPCharge generate ping++ charge object for client
func GetPingPPCharge(amount int, channel string) string {
	pingpp.Key = "sk_test_ibbTe5jLGCi5rzfH4OqPW9KC"
	params := &pingpp.ChargeParams{
		Order_no:  "123456789",
		App:       pingpp.App{Id: "app_1Gqj58ynP0mHeX1q"},
		Amount:    uint64(amount),
		Channel:   channel,
		Currency:  "cny",
		Client_ip: "127.0.0.1",
		Subject:   "Your Subject",
		Body:      "Your Body",
	}

	ch, err := charge.New(params)
	if err != nil {
		errs, _ := json.Marshal(err)
		return string(errs)
	}
	chs, _ := json.Marshal(ch)
	return string(chs)
}
