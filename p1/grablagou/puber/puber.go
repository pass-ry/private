package puber

import (
	"context"
	"encoding/json"
	"fmt"
	"grablagou/account"
	"grablagou/suber"
	"math"
	"runtime"
	"strings"
	"time"

	"gitlab.ifchange.com/data/cordwood/redis"

	"github.com/pkg/errors"
	"gitlab.ifchange.com/data/cordwood/log"
	"gitlab.ifchange.com/data/cordwood/util/date"
)

const (
	ConstPubQueue      = "pyspider_lagou_queue"
	ConstLoginPubQueue = "pyspider_lagou_login_queue"
)

var (
	constWorkDuration        = time.Duration(3) * time.Hour
	constWorkHolidayDuration = time.Duration(6) * time.Hour
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

	if len(acs) == 0 {
		return
	}

	// 小时除以账号总数,得到几秒发布一个账号
	lenf := int(math.Floor(float64(int(constWorkDuration.Minutes()) / len(acs))))
	for k, ac := range acs {
		select {
		case <-ctx.Done():
			return
		default:
		}
		// 添加一个中间函数，负责不同时间账号的发布
		go func(t int, ac *account.Account) {
			// time 必须大于0
			if t < 1 {
				t = 1
			}

			// 由于等待，ctx存活时间需加上等待时长
			_, cancel := context.WithTimeout(parentCtx, constWorkDuration+time.Duration(t)*time.Minute)
			defer cancel()

			ticker := time.After(time.Duration(t) * time.Minute)
			select {
			case <-ticker:
				err := acRun(ac)
				if err != nil {
					log.Warnf("Puber %+v Pub Error %v", ac, err)
					return
				}
				log.Infof("Puber %+v Pub Success", ac)
			}
		}(k*lenf, ac)
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
	workingKey := fmt.Sprintf("pyspider_lagou_account_%s_lock",
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
	params := struct {
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
