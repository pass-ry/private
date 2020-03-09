package puber

import (
	"context"
	"encoding/json"
	"fmt"
	"grabliepin/account"
	"grabliepin/suber"
	"runtime"
	"strings"
	"time"

	"gitlab.ifchange.com/data/cordwood/redis"

	"github.com/pkg/errors"
	"gitlab.ifchange.com/data/cordwood/log"
	"gitlab.ifchange.com/data/cordwood/util/date"
)

const (
	ConstPubQueue      = "pyspider_liepin_queue"
	ConstLoginPubQueue = "pyspider_liepin_login_queue"
)

var (
	constWorkDuration        = time.Duration(2) * time.Hour
	constWorkHolidayDuration = time.Duration(3) * time.Hour
)

func Run(ctx context.Context) {
	ticker := time.NewTimer(constWorkDuration)
	for {
		if date.IsForceStop() {
			time.Sleep(time.Hour)
			log.Info("Spring Festival")
			continue
		}

		if ok := date.IsHoliday(); ok {
			ticker.Reset(constWorkHolidayDuration)
		} else {
			ticker.Reset(constWorkDuration)
		}

		nowHour := time.Now().Hour()
		if nowHour >= 7 && nowHour < 23 {
			run(ctx)
		} else {
			log.Infof("Puber Is sleeping 23:00-7:00")
		}

		select {
		case <-ticker.C:
		case <-ctx.Done():
			log.Infof("Puber stopped")
			return
		}
	}
}

func run(parentCtx context.Context) {
	ctx, cancel := context.WithTimeout(parentCtx, constWorkDuration)
	defer cancel()
	acs, err := account.GetAllUsefulAccounts()
	if err != nil {
		log.Errorf("Puber GetAllUsefulAccounts %v", err)
		return
	}
	for _, ac := range acs {
		select {
		case <-ctx.Done():
			return
		default:
		}
		err := acRun(ac)
		if err != nil {
			log.Warnf("Puber %+v Pub Error %v",
				ac, err)
			continue
		}
		log.Infof("Puber %+v Pub Success", ac)
	}
}

func acRun(ac *account.Account) error {
	defer func() {
		err := recover()
		if err == nil {
			return
		}
		buf := make([]byte, 1<<10)
		num := runtime.Stack(buf, false)
		log.Errorf("Puber %+v PANIC %v %v %v",
			ac, err, num, string(buf))
	}()
	retryTimes := 3
	pubQueue := ConstPubQueue
	subQueue := suber.ConstSubQueue

	conn, err := redis.GetConstClient()
	if err != nil {
		return errors.Wrap(err, "Get Redis Const Client")
	}
	defer conn.Close()

	// check if is working
	workingKey := fmt.Sprintf("pyspider_liepin_account_%s_lock",
		ac.Username)
	lockResult, err := conn.DoInt("SETNX", workingKey, "is_working")
	if err != nil {
		return errors.Wrap(err, "Redis Cmd SETNX")
	}
	if lockResult != 1 {
		return errors.New("is working")
	}
	_, err = conn.Do("EXPIRE", workingKey,
		int(constWorkDuration.Seconds()))
	if err != nil {
		conn.Do("DEL", workingKey)
		return errors.Wrap(err, "Redis Cmd EXPIRE")
	}

	// check if in queue
	allInQueue, err := conn.DoStrings("LRANGE", pubQueue, 0, -1)
	if err != conn.ErrNil() && err != nil {
		return errors.Wrap(err, "Redis Cmd LRANGE")
	}
	for _, job := range allInQueue {
		if strings.Contains(job, ac.Username) {
			return errors.New("is in queue")
		}
	}

	if err := pub(ac, retryTimes, pubQueue, subQueue); err != nil {
		return errors.Wrap(err, "Pub")
	}
	return nil
}

func pub(ac *account.Account,
	retryTimes int, pubQueue, subQueue string) error {
	return pubWithPhone(ac, "", retryTimes, pubQueue, subQueue)
}

func pubWithPhone(ac *account.Account, phone string,
	retryTimes int, pubQueue, subQueue string) error {
	params := struct {
		Username      string `json:"username"`
		Password      string `json:"password"`
		Time          string `json:"time"`
		RetryTimes    int    `json:"retry_times"`
		CallBackQueue string `json:"callback_queue"`
		Phone         string `json:"phone,omitempty"`
	}{
		Username:   ac.Username,
		Password:   ac.Password,
		RetryTimes: retryTimes,
		Time: ac.ReceiveTime.Add(time.Duration(-5) * time.Second).
			Format("2006-01-02 15:04:05"),
		CallBackQueue: subQueue,
		Phone:         phone,
	}
	jsonParams, err := json.Marshal(params)
	if err != nil {
		return errors.Wrapf(err, "Json Marshal Username: %s Password: %d Time: %v RetryTimes: %d CallBackQueue: %s", params.Username,
			len(params.Password), params.Time, params.RetryTimes, params.CallBackQueue)
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
