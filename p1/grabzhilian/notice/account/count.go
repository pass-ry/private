package account

import (
	"context"
	"fmt"
	account "grabzhilian/account"
	"runtime"

	"gitlab.ifchange.com/data/cordwood/log"
)

func SyncRun(ctx context.Context) {
	log.Infof("ZhilianAccountCount Activating")
	defer func() {
		log.Infof("ZhilianAccountCount Hiddening")
		err := recover()
		if err == nil {
			return
		}
		buf := make([]byte, 1<<10)
		num := runtime.Stack(buf, false)
		log.Errorf("ZhilianAccountCount PANIC %v %v %v",
			err, num, string(buf))
	}()

	all, err := account.GetAll()
	if err != nil {
		log.Errorf("ZhilianAccountCount Try GetAll Error %v",
			err)
		return
	}
	logAC := []string{}
	for _, ac := range all {
		if len(ac.Msg) == 0 {
			continue
		}
		if ac.Msg == "帐号和密码不匹配" {
			continue
		}
		logAC = append(logAC, fmt.Sprintf("%+v", *ac))
	}
	if len(logAC) < 3 {
		log.Infof("ZhilianAccountCount Skip")
		return
	}

	if err := sendMail(style(logAC)); err != nil {
		log.Errorf("ZhilianAccountCount SendMail Error %v",
			err)
	}

}
