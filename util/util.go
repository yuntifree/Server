package util

import (
	"math/rand"
	"strconv"
	"time"
)

//GenWifiPass gen 4-digit password
func GenWifiPass() string {
	r := rand.New(rand.NewSource(time.Now().Unix()))
	var pass string
	for i := 0; i < 4; i++ {
		pass += strconv.Itoa(r.Intn(10))
	}

	return pass
}
