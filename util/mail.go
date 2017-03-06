package util

import (
	"fmt"
	"net/smtp"
)

const (
	mailAddress = "dingrong@yunxingzh.com"
	mailPasswd  = "RyDc926ala"
	smtpAddress = "smtp.yunxingzh.com"
)

//SendAlertMail send alert mail
func SendAlertMail(content string) error {
	auth := smtp.PlainAuth(
		"",
		mailAddress,
		mailPasswd,
		smtpAddress,
	)
	var mailList []string
	mailList = append(mailList, mailAddress)
	subject := "【告警邮件】"
	sub := fmt.Sprintf("subject: %s\r\n\r\n", subject)

	err := smtp.SendMail(
		smtpAddress+":25",
		auth,
		mailAddress,
		mailList,
		[]byte(sub+content),
	)
	return err
}
