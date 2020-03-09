package mail

import (
	"context"
	registerPackage "grabmail/register"
	"runtime"

	"gitlab.ifchange.com/data/cordwood/cfg/loader"
	"gitlab.ifchange.com/data/cordwood/log"
	"gitlab.ifchange.com/data/cordwood/notice/email"
)

func SyncRun(ctx context.Context, register registerPackage.Register) {
	if loader.ENV != loader.PROD {
		log.Infof("grabmail/admin skipped in %s", loader.ENV)
		return
	}
	log.Infof("grabmail/admin Activating")
	defer func() {
		log.Infof("grabmail/admin Hiddening")
		err := recover()
		if err == nil {
			return
		}
		buf := make([]byte, 1<<10)
		num := runtime.Stack(buf, false)
		log.Errorf("grabmail/admin PANIC %v %v %v",
			err, num, string(buf))
	}()
	if !register.LogIn() {
		log.Infof("grabmail/admin Register Failed")
		return
	}
	// un logout
	// wait expire only
	// defer register.LogOut()
	body := count()
	if len(body) == 0 {
		return
	}
	if err := sendMail(body); err != nil {
		log.Errorf("grabmail/admin send mail error %v", err)
		return
	}
	log.Infof("grabmail/admin send mail success")
}

func sendMail(body string) error {
	return email.Send(email.DefaultFilter,
		"GrabMail", "Daily GrabMail", body,
		"chenwei@ifchange.com",
		"xin.yang@ifchange.com",
		"dejie.guo@ifchange.com",
		"yuan.ren@ifchange.com",
		"xingrui.du@ifchange.com")
}
