package suber

import (
	"grabliepin/account"
	result "grabliepin/grab-result"
	"runtime"

	"github.com/pkg/errors"
	"gitlab.ifchange.com/data/cordwood/log"
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
			gErr = errors.Wrap(err, "HandleBind Login Fail Account Push")
		}
	}()
	switch r.Code {
	case 1:
		// login success
		ac.Cookie = r.Cookie
		ac.Msg = ""
		ac.IsDeleted = false
		switch {
		case accountIsExist:
			if err := ac.Sync(); err != nil {
				return errors.Wrap(err, "HandleBind Login Success Account Sync")
			}
		case !accountIsExist:
			if err := ac.Insert(); err != nil {
				return errors.Wrap(err, "HandleBind Login Success Account Insert")
			}
		}
	default:
		ac.Msg = r.Msg
		ac.IsDeleted = true
		switch {
		case accountIsExist:
			if err := ac.Sync(); err != nil {
				return errors.Wrap(err, "HandleBind Login Success Account Sync")
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
		ac.IsDeleted = false
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
		// if ac.Errcount > constMaxLoginFail {
		// 	ac.IsDeleted = true
		// 	if err := ac.Push(); err != nil {
		// 		return errors.Wrap(err, "HandleAccount Account Push")
		// 	}
		// }
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
	log.Infof("SubResult....%v", r.Resume)
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
