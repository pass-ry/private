package suber

import (
	"encoding/json"
	"grabzhilian/account"
	result "grabzhilian/grab-result"
	"runtime"
	"time"

	"github.com/pkg/errors"
	"gitlab.ifchange.com/data/cordwood/log"
	"gitlab.ifchange.com/data/cordwood/redis"
)

var (
	constMaxLoginFail = 200
)

func (r *SubResult) handleBind(ac *account.Account, source int, accountIsExist bool) (gErr error) {
	if r.Username != ac.Username || r.Password != ac.Password {
		return errors.Errorf("HandleBind Error account:%+v != result:%+v",
			ac, r)
	}
	defer func() {
		if err := ac.Push(r.Code, source); err != nil {
			gErr = errors.Wrap(err, "HandleBind Login Success Account Push")
		}
	}()
	switch r.Code {
	case 1:
		// login success
		ac.Msg = ""
		ac.Cookie = r.Cookie
		ac.IsDeleted = false
		ac.Errcount = 0
		switch {
		case accountIsExist:
			if err := ac.SyncC(); err != nil {
				return errors.Wrap(err, "HandleBind Login Success Account SyncC")
			}
		case !accountIsExist:
			if err := ac.Insert(); err != nil {
				return errors.Wrap(err, "HandleBind Login Success Account Insert")
			}
		}

		// 绑定成功直接开启抓取简历
		if err := firstBindSuccess(ac, 3, "pyspider_zhaopin_queue",
			ConstSubQueue, false, ""); err != nil {
			return errors.Wrap(err, "HandleBind bindSuccessGrasp err")
		}
	default:
		ac.Msg = r.Msg
		ac.Cookie = r.Cookie
		ac.IsDeleted = true
		switch {
		case accountIsExist:
			if err := ac.SyncC(); err != nil {
				return errors.Wrap(err, "HandleBind Login Success Account SyncC")
			}
		}
	}
	return
}

func (r *SubResult) handle() error {
	defer func() {
		err := recover()
		if err == nil {
			return
		}
		buf := make([]byte, 1<<10)
		num := runtime.Stack(buf, false)
		log.Errorf("Suber-Handle %+v PANIC %v %v %v",
			r, err, num, string(buf))
	}()

	ac, err := account.GetByUsername(r.Username)
	if err != nil {
		return errors.Wrapf(err,
			"Handle Account GetByUsername %s", r.Username)
	}
	if r.Resume == nil {
		return r.handleAccount(ac)
	} else {
		return r.handleResume(ac)
	}
}

func (r *SubResult) handleAccount(ac *account.Account) error {
	source := 0
	switch r.Code {
	case 1: // login success
		if ac.Errcount == 0 {
			return nil
		}
		ac.Errcount = 0
		ac.Msg = ""
	case 3: // password error
		ac.Errcount += constMaxLoginFail
		ac.Msg = r.Msg
		ac.IsDeleted = true
		source = -1
	default:
		ac.Errcount += 1
		ac.Msg = r.Msg
		ac.IsDeleted = true
		source = -1
	}
	if err := ac.Sync(); err != nil {
		return errors.Wrapf(err,
			"HandleAccount Account Sync")
	}

	if err := ac.Push(r.Code, source); err != nil {
		return errors.Wrapf(err,
			"HandleAccount Account Push")
	}

	return nil
}

func (r *SubResult) handleResume(ac *account.Account) (err error) {
	dbResult := new(result.Result)
	dbResult.AccountID = ac.ID
	resume := r.Resume

	dbResult.CV_ID, err = resume.ResumeID()
	if err != nil {
		return errors.Wrap(err, "Get resumeID")
	}
	dbResult.JD_ID, err = resume.PositionID()
	if err != nil {
		return errors.Wrap(err, "Find PositionID")
	}
	dbResult.ReceiveTime, err = resume.ReceiveTime()
	if err != nil {
		return errors.Wrap(err, "Find ReceiveTime")
	}
	dbResult.CV_Name, err = resume.CVName()
	if err != nil {
		return errors.Wrap(err, "Find CVName")
	}
	dbResult.JD_Name, err = resume.JDName()
	if err != nil {
		return errors.Wrap(err, "Find JDName")
	}

	exist, err := dbResult.CheckDuplicated()
	if err != nil {
		return errors.Wrap(err, "Check Duplicated")
	}
	if exist {
		return errors.New("resume is exist")
	}

	dbResult.ContentDfs, err = resume.Content()
	if err != nil {
		return errors.Wrap(err, "Payload Content")
	}

	err = dbResult.Insert()
	if err != nil {
		return errors.Wrap(err, "Insert Grab Result")
	}
	if dbResult.ReceiveTime.After(ac.ReceiveTime) {
		ac.ReceiveTime = dbResult.ReceiveTime
		err = ac.Sync()
		if err != nil {
			return errors.Wrap(err, "Account Sync")
		}
	}
	return nil
}

func firstBindSuccess(ac *account.Account,
	retryTimes int, pubQueue, subQueue string,
	isCheckPassWord bool, phone string) error {
	params := struct {
		CheckPassword bool   `json:"check_password"`
		Phone         string `json:"phone"`

		Username      string `json:"username"`
		Password      string `json:"password"`
		Time          string `json:"time"`
		RetryTimes    int    `json:"retry_times"`
		CallBackQueue string `json:"callback_queue"`
	}{
		Username:   ac.Username,
		Password:   ac.Password,
		RetryTimes: retryTimes,
		Time: ac.ReceiveTime.Add(time.Duration(-5) * time.Second).
			Format("2006-01-02 15:04:05"),
		CallBackQueue: subQueue,

		CheckPassword: isCheckPassWord,
		Phone:         phone,
	}
	jsonParams, err := json.Marshal(params)
	if err != nil {
		return errors.Wrapf(err, "Json Marshal CheckPassword: %v Phone: %s Username: %s Password: %d Time: %v RetryTimes: %d CallBackQueue: %s",
			params.CheckPassword, params.Phone, params.Username, len(params.Password), params.Time, params.RetryTimes, params.CallBackQueue)
	}
	conn, err := redis.GetConstClient()
	if err != nil {
		return errors.Wrap(err, "Get Redis Const Client")
	}
	defer conn.Close()
	_, err = conn.Do("LPUSH", pubQueue, string(jsonParams))
	if err != nil {
		return errors.Wrap(err, "Redis Exec LPUSH")
	}
	return nil
}
