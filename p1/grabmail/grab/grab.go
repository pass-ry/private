package grab

import (
	"context"
	"fmt"
	"grabmail/grab/client"
	"grabmail/grab/mail"
	"grabmail/models/account"
	"grabmail/register"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/beiping96/grace"
	"github.com/pkg/errors"
	"gitlab.ifchange.com/data/cordwood/log"
	"gitlab.ifchange.com/data/cordwood/redis"
	"gitlab.ifchange.com/data/cordwood/util/date"
)

const (
	constGroupMailFetch = 50
)

var (
	constMinWorkerDuration = time.Duration(10) * time.Second
)

type Grab struct {
	Config
	workerLimit chan struct{}
}

type Config struct {
	NAME           string
	CTX            context.Context
	Register       register.Register
	AccountFilter  func(ac *account.Account) bool
	WorkerNum      uint16
	WorkerDuration time.Duration
}

func (cfg *Config) check() {
	if cfg == nil {
		panic("grabmail/grab config check error nil cfg")
	}
	if cfg.CTX == nil {
		panic("grabmail/grab config check error nil cfg.CTX")
	}
	if cfg.Register == nil {
		panic("grabmail/grab config check error nil cfg.Register")
	}
	if cfg.AccountFilter == nil {
		panic("grabmail/grab config check error nil cfg.AccountFilter")
	}
	if cfg.WorkerNum == 0 || cfg.WorkerNum > 2000 {
		panic(fmt.Sprintf("grabmail/grab config check error cfg.WorkerNum not allowed %d",
			cfg.WorkerNum))
	}
	if cfg.WorkerDuration <= constMinWorkerDuration {
		panic(fmt.Sprintf("grabmail/grab config check error cfg.WorkerDuration not allowed %d",
			cfg.WorkerDuration))
	}
}

func (cfg *Config) constQueue() string {
	return "grabmail_grab_queue_" + strings.ToLower(cfg.NAME)
}

func New(cfg *Config) *Grab {
	cfg.check()
	return &Grab{
		Config:      *cfg,
		workerLimit: make(chan struct{}, cfg.WorkerNum),
	}
}

func (g *Grab) SyncRun() {
	for i := uint16(0); i < g.WorkerNum; i++ {
		var workerID = i
		grace.Go(func(ctx context.Context) { g.worker(ctx, workerID) })
	}

	for {
		if date.IsForceStop() {
			time.Sleep(time.Hour)
			log.Infof("date control is force stop")
			continue
		}

		duration := 2 * g.WorkerDuration
		if date.IsHoliday() {
			duration = 6 * g.WorkerDuration
		}
		timer := time.After(duration)

		allAccount, err := account.GetCanUsedAll()
		if err != nil {
			log.Errorf("grabmail/grab %s account.GetAll error %v",
				g.NAME, err)
			time.Sleep(time.Duration(3) * time.Second)
			continue
		}
		if len(allAccount) == 0 {
			log.Errorf("grabmail/grab %s account.GetAll error NO Accounts",
				g.NAME)
			time.Sleep(time.Duration(5) * time.Second)
			continue
		}
		sort.Slice(allAccount, func(i, j int) bool { return allAccount[i].ID > allAccount[j].ID })

		for _, ac := range allAccount {
			if g.AccountFilter(ac) == false {
				continue
			}
			g.syncRun(ac)
			log.Infof("grabmail/grab %s Account: %+v Grab Pub",
				g.NAME, ac)
		}

		select {
		case <-g.CTX.Done():
			return
		case <-timer:
		}
	}
}

func (g *Grab) syncRun(ac *account.Account) {
	defer ac.Close()

	conn, err := redis.GetConstClient()
	if err != nil {
		log.Errorf("grabmail/grab %s GetConstClient error %v",
			g.NAME, err)
		return
	}
	defer conn.Close()

	var (
		sleep time.Duration
	)

	for {
		select {
		case <-g.CTX.Done():
			return
		case <-time.After(sleep):
			sleep = 0
		}
		size, err := conn.DoInt("LLEN", g.constQueue())
		if err != nil {
			log.Errorf("grabmail/grab %s %+v Redis LLEN error %v",
				g.NAME, ac, err)
			return
		}
		if size >= int(g.WorkerNum) {
			sleep = time.Duration(5) * time.Second
			log.Debugf("grabmail/grab %s %+v is Full queue",
				g.NAME, ac)
			continue
		}

		if _, err := conn.Do("LPUSH", g.constQueue(), ac.UserName); err != nil {
			log.Errorf("grabmail/grab %s %+v LPUSH %v",
				g.NAME, ac, err)
		}
		return
	}
}

func (g *Grab) worker(ctx context.Context, workerID uint16) {
	conn, err := redis.GetConstClient()
	if err != nil {
		log.Errorf("grabmail/grab %s worker%d GetConstClient error %v",
			g.NAME, workerID, err)
		return
	}
	defer conn.Close()

	var (
		sleep time.Duration
	)
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(sleep):
			sleep = 0
		}
		username, err := conn.DoString("RPOP", g.constQueue())
		if err == conn.ErrNil() {
			// log.Debugf("grabmail/grab %s worker%d is sleeping",
			// 	g.NAME, workerID)
			sleep = time.Duration(5) * time.Second
			continue
		}
		if err != nil {
			log.Errorf("grabmail/grab %s worker%d RPOP error %v",
				g.NAME, workerID, err)
			sleep = time.Second
			continue
		}

		err = g.run(ctx, username)
		if err == nil {
			log.Infof("grabmail/grab %s worker%d Grab %s stop",
				g.NAME, workerID, username)
		} else {
			log.Errorf("grabmail/grab %s worker%d Grab %s error %v",
				g.NAME, workerID, username, err)
		}
	}
}

func Pin(ctx context.Context, g *Grab) {
	conn, err := redis.GetConstClient()
	if err != nil {
		panic(errors.Wrap(err, "redis.GetConstClient"))
	}
	defer conn.Close()

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(10) * time.Second):
		}
		username, err := conn.DoString("GET", "pin_email_account")
		if err == conn.ErrNil() {
			continue
		}
		err = g.run(ctx, username)
		if err == nil {
			log.Infof("grabmail/grab %s PIN Grab %s stop",
				g.NAME, username)
		} else {
			log.Errorf("grabmail/grab %s PIN Grab %s error %v",
				g.NAME, username, err)
		}
	}
}

func (g *Grab) run(ctx context.Context, username string) error {
	selfCtx, cancel := context.WithTimeout(ctx, g.WorkerDuration)
	defer cancel()

	ac, err := account.GetCanUsedAccountByUsername(username)
	if err != nil {
		return errors.Wrap(err, "GetAccountByUsername")
	}
	defer ac.Close()

	defer func() {
		err := recover()
		if err == nil {
			return
		}
		buf := make([]byte, 1<<10)
		num := runtime.Stack(buf, false)
		log.Errorf("grabmail/grab %s PANIC %+v %v %v %v",
			g.NAME, ac, err, num, string(buf))
	}()

	if !g.Register.LogIn(ac.UserName) {
		log.Infof("grabmail/grab %s Account: %+v Register Failed",
			g.NAME, ac)
		return nil
	}

	defer g.Register.LogOut(ac.UserName)

	log.Infof("grabmail/grab %s Account: %+v Grab Start",
		g.NAME, ac)
	start := time.Now()

	allSize, notExistSize, successSize, err := run(selfCtx, g.NAME, ac, false)
	if err != nil {
		return err
	}

	err = ac.UpdateLastCrawlerTime(ctx)
	if err != nil {
		log.Warnf("grabmail/grab %s Account: %+v UpdateLastCrawlerTime error %v",
			g.NAME, ac, err)
	}
	log.Infof("grabmail/grab %s Account: %+v Grab Stop COST: %s All/NotExist/Success: %d / %d / %d",
		g.NAME, ac, time.Now().Sub(start), allSize, notExistSize, successSize)

	return nil
}

func run(ctx context.Context, name string, ac *account.Account, skipSave bool) (allSize, notExistSize, successSize int, rErr error) {
	switch ac.TYPE() {
	case account.EXCHANGE:
		return runEXCHANGE(ctx, name, ac, skipSave)
	default:
		return runDefault(ctx, name, ac, skipSave)
	}
}

func runEXCHANGE(ctx context.Context, name string, ac *account.Account, skipSave bool) (allSize, notExistSize, successSize int, rErr error) {
	cli, _, err := client.NewClient(ctx, ac, false)
	if err != nil {
		rErr = errors.Wrap(err, "client.NewClient")
		return
	}
	defer cli.Close()

	inboxMail, err := cli.FetchInboxMail(ctx, nil)
	if err != nil {
		rErr = errors.Wrap(err, "cli.FetchInboxMail")
		return
	}
	allSize = len(inboxMail)
	notExistSize = len(inboxMail)
	successSize = len(inboxMail)
	log.Infof("grabmail/grab %s Account: %+v FetchInboxMail fetchLen:%d fetchSuccessLen:%d",
		name, ac, len(inboxMail), len(inboxMail))

	addGrabSuccessNum(len(inboxMail))
	if len(inboxMail) == 0 {
		return
	}
	if !skipSave {
		mail.InboxSaveMail(ctx, ac, inboxMail)
	}
	return
}

func runDefault(ctx context.Context, name string, ac *account.Account, skipSave bool) (allSize, notExistSize, successSize int, rErr error) {
	cli, _, err := client.NewClient(ctx, ac, false)
	if err != nil {
		rErr = errors.Wrap(err, "client.NewClient")
		return
	}
	defer cli.Close()
	indexMail, err := cli.GetIndexMail(ctx)
	if err != nil {
		rErr = errors.Wrap(err, "cli.GetIndexMail")
		return
	}
	allSize = len(indexMail)
	log.Infof("grabmail/grab %s Account: %+v GetIndexMail len:%d",
		name, ac, len(indexMail))
	if len(indexMail) == 0 {
		return
	}
	notExistIndexMail, err := mail.IndexFilterExistMail(ctx, ac, indexMail)
	if err != nil {
		rErr = errors.Wrap(err, "mail.IndexFilterExistMail")
		return
	}
	notExistIndexMail = mail.IndexSort(constGroupMailFetch, ac, notExistIndexMail)
	notExistSize = len(notExistIndexMail)

	if len(notExistIndexMail) > 0 {
		log.Infof("grabmail/grab %s Account: %+v GetIndexMailLen:%d notExistIndexMailLen:%d FirstInbox:%+v",
			name, ac, len(indexMail), len(notExistIndexMail), notExistIndexMail[0])
	} else {
		log.Infof("grabmail/grab %s Account: %+v GetIndexMailLen:%d notExistIndexMailLen:%d",
			name, ac, len(indexMail), len(notExistIndexMail))
	}

	// Grouping
	for len(notExistIndexMail) > 0 {
		select {
		case <-ctx.Done():
			rErr = errors.Errorf("Receive stop signal")
			return
		default:
		}
		count := len(notExistIndexMail)
		if count > constGroupMailFetch {
			count = constGroupMailFetch
		}
		execIndexMails := notExistIndexMail[:count]
		notExistIndexMail = notExistIndexMail[count:]
		inboxMail, err := cli.FetchInboxMail(ctx, execIndexMails)
		if err != nil {
			rErr = errors.Wrap(err, "cli.FetchInboxMail")
			return
		}
		successSize += len(inboxMail)
		log.Infof("grabmail/grab %s Account: %+v FetchInboxMail fetchLen:%d fetchSuccessLen:%d",
			name, ac, len(execIndexMails), len(inboxMail))

		addGrabSuccessNum(len(inboxMail))

		if len(inboxMail) == 0 {
			continue
		}
		if !skipSave {
			mail.InboxSaveMail(ctx, ac, inboxMail)
		}
	}
	return
}
