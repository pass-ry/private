package account

import (
	"context"
	"grabmail/models/account"
	registerPackage "grabmail/register"
	"runtime"

	"github.com/pkg/errors"
	"gitlab.ifchange.com/data/cordwood/crypto/des3"
	"gitlab.ifchange.com/data/cordwood/encoding/json"
	"gitlab.ifchange.com/data/cordwood/log"
	"gitlab.ifchange.com/data/cordwood/redis"
)

func SyncRun(ctx context.Context, register registerPackage.Register) {
	log.Infof("grabmail/push/account Activating")
	defer func() {
		log.Infof("grabmail/push/account Hiddening")
		err := recover()
		if err == nil {
			return
		}
		buf := make([]byte, 1<<10)
		num := runtime.Stack(buf, false)
		log.Errorf("grabmail/push/account PANIC %v %v %v",
			err, num, string(buf))
	}()
	if !register.LogIn() {
		log.Infof("grabmail/push/account Register Failed")
		return
	}
	// un logout
	// wait expire only
	// defer register.LogOut()

	failAccounts, err := account.GetLoginFailAccount()
	if err != nil {
		log.Errorf("grabmail/push/account GetLoginFailAccount error %v",
			err)
		return
	}
	log.Infof("grabmail/push/account GetLoginFailAccount len:%d",
		len(failAccounts))
	for _, failAccount := range failAccounts {
		select {
		case <-ctx.Done():
			return
		default:
		}
		err := Push(ctx, -1, failAccount, 2, failAccount.Msg)
		if err != nil {
			log.Errorf("grabmail/push/account Push %+v Fail",
				failAccount)
			continue
		}
		log.Infof("grabmail/push/account Push %+v Success",
			failAccount)
		failAccount.MarkAsDeleted()
	}
}

func Push(ctx context.Context, source int, ac *account.Account, status int, msg string) error {
	var msgPtr *string = nil
	if len(msg) > 0 {
		msgPtr = &msg
	}
	showPassword, err := des3.Decrypt(ac.Password)
	if err != nil {
		return errors.Wrap(err, "password Decrypt")
	}

	params := struct {
		Username      string  `json:"username"`
		Password      string  `json:"password"`
		Agreement     string  `json:"agreement"`
		ServerAddress string  `json:"server_address"`
		Port          int     `json:"port"`
		Ssl           int     `json:"ssl"`
		Status        int     `json:"status"`
		Msg           *string `json:"msg"`

		Source int `json:"source"`
	}{
		Username:      ac.UserName,
		Password:      showPassword,
		Agreement:     ac.TYPE().String(),
		ServerAddress: ac.MailServer,
		Port:          ac.Port,
		Ssl:           ac.Ssl,
		Status:        status,
		Msg:           msgPtr,
		Source:        source,
	}

	pushMsg, err := json.Marshal(params)
	if err != nil {
		return errors.Wrap(err, "Account-Push Marshal")
	}
	conn, err := redis.GetConstClient()
	if err != nil {
		return errors.Wrap(err, "Account-Push GetConstClient")
	}
	defer conn.Close()

	_, err = conn.Do("LPUSH", "email_account_forward_queue", pushMsg)
	if err != nil {
		return errors.Wrap(err, "Account-Push LPUSH")
	}
	return nil
}
