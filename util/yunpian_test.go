package util

import "testing"

func Test_SendYPSMS(t *testing.T) {
	err := SendYPSMS("18682313472", "123456", "6月2号14:00~16:00", tplID)
	if err != nil {
		t.Errorf("SendYPSMS failed:%v", err)
	}
}

func Test_SendPaySMS(t *testing.T) {
	err := SendPaySMS("18682313472")
	if err != nil {
		t.Errorf("SendPaySMS failed:%v", err)
	}
}
