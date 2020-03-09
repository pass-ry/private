package account

import (
	"gitlab.ifchange.com/data/cordwood/notice/email"
)

func sendMail(body string) error {
	return email.Send(email.DefaultFilter,
		"ZhiLianAccount", "Hourly ZhiLian Account Reports", body,
		"chenwei@ifchange.com",
		"xin.yang@ifchange.com",
		"dejie.guo@ifchange.com",
		"yuan.ren@ifchange.com",
		"xinrui.du@ifchange.com")
}
