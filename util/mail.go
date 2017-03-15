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

func sendMail(title, content string) error {
	auth := smtp.PlainAuth(
		"",
		mailAddress,
		mailPasswd,
		smtpAddress,
	)
	var mailList []string
	mailList = append(mailList, mailAddress)
	sub := fmt.Sprintf("subject: %s\r\n\r\n", title)

	err := smtp.SendMail(
		smtpAddress+":25",
		auth,
		mailAddress,
		mailList,
		[]byte(sub+content),
	)
	return err
}

//SendAlertMail send alert mail
func SendAlertMail(content string) error {
	title := "【告警邮件】"
	return sendMail(title, content)
}

//SendCronMail send cron mail
func SendCronMail(content string) error {
	title := "【定时任务】"
	return sendMail(title, content)
}
