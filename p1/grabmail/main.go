package main

import (
	"context"
	"grabmail/contact"
	"grabmail/controller"
	"grabmail/grab"
	"grabmail/models/account"
	noticeMail "grabmail/notice/mail"
	pushAccount "grabmail/push/account"
	pushMail "grabmail/push/mail"
	"grabmail/register"
	"os"
	"strings"
	"time"

	"github.com/beiping96/grace"
	"github.com/rk/go-cron"
	"gitlab.ifchange.com/data/cordwood/cfg"
	"gitlab.ifchange.com/data/cordwood/cfg/loader"
	"gitlab.ifchange.com/data/cordwood/crypto/des3"
	"gitlab.ifchange.com/data/cordwood/dfs"
	"gitlab.ifchange.com/data/cordwood/log"
	"gitlab.ifchange.com/data/cordwood/mysql"
	"gitlab.ifchange.com/data/cordwood/redis"
	router "gitlab.ifchange.com/data/cordwood/rpc/rpc-router"
	server "gitlab.ifchange.com/data/cordwood/rpc/rpc-server"
)

func main() {
	// Basic
	// load config (required)
	loader.LoadCfgByEnv()
	// init log (required)
	log.Construct(cfg.GetCfgLog())
	// init grace (required)
	grace.Log(log.Infof)
	// pid file
	grace.PID("pid/")

	// Useful driver
	// init MySQL (optional)
	var cfgMySQL mysql.Config
	if os.Getenv("ENV") == "prod" {
		cfgMySQL.Username = os.Getenv("MySQL_Username")
		cfgMySQL.Password = os.Getenv("MySQL_Password")
		cfgMySQL.Address = os.Getenv("MySQL_Address")
		cfgMySQL.Port = os.Getenv("MySQL_Port")
		cfgMySQL.DB = "grab_receive_mail"
		cfgMySQL.KeepAlive = 300
		cfgMySQL.MaxOpenConns = 200
		cfgMySQL.MaxIdleConns = 10
	} else {
		cfgMySQL = cfg.GetCfgMySQL()
	}
	mysql.Construct(cfgMySQL)
	if err := mysql.GetConstConn().Ping(); err != nil {
		panic(err)
	}
	// init Redis (optional)
	redis.Construct(cfg.GetCfgRedis())

	// init DFS (optional)
	dfs.Construct(cfg.GetCfgDfs())

	// 3DES
	des3.Setup(cfg.GetCfgCustom().Get("DES3"),
		true /* open memory-cache */)

	// httpd bind
	grace.Go(controller.BindHandler)

	// httpd
	grace.Go(func(ctx context.Context) {
		server.NewServer(ctx,
			cfg.GetCfgCustom().Get("rpc"),
			router.NewRouter().
				WithPPROF().WithMetrics().
				Handler("/tob", controller.ToB).
				Handler("/grab_mail_tob", controller.MailToB).
				Handler("/mail_exist", controller.MailExist).
				Handler("/pin_account", controller.PinAccount),
		).GraceRun()
	})

	isVanke := func(ac *account.Account) bool {
		return strings.Contains(ac.UserName, "@vanke.com")
	}

	// Start Vanke
	grace.Go(func(ctx context.Context) {
		grab.New(&grab.Config{
			NAME: "VankeOnly",
			CTX:  ctx,

			Register: register.New(register.ROLE_GRAB,
				time.Duration(2)*time.Minute),
			AccountFilter:  isVanke,
			WorkerNum:      1,
			WorkerDuration: time.Duration(5) * time.Minute,
		}).SyncRun()
	})

	// Start Default
	grace.Go(func(ctx context.Context) {
		grab.New(&grab.Config{
			NAME: "Default",
			CTX:  ctx,

			Register: register.New(register.ROLE_GRAB,
				time.Duration(10)*time.Minute),
			AccountFilter: func(ac *account.Account) bool {
				return !isVanke(ac)
			},
			WorkerNum:      50,
			WorkerDuration: time.Duration(15) * time.Minute,
		}).SyncRun()
	})

	// Start PIN
	grace.Go(func(ctx context.Context) {
		grab.Pin(ctx, grab.New(&grab.Config{
			NAME: "PIN",
			CTX:  ctx,

			Register: register.New(register.ROLE_GRAB,
				time.Duration(1)*time.Hour),
			AccountFilter: func(ac *account.Account) bool {
				return true
			},
			WorkerNum:      1,
			WorkerDuration: time.Duration(1) * time.Hour,
		}))
	})

	isLagou := func(siteID int) bool { return siteID == 11 }
	// start default mail push
	grace.Go(func(ctx context.Context) {
		pushMail.New(&pushMail.Config{
			CTX:  ctx,
			NAME: "Default",

			Register: register.New(register.ROLE_PUSH_MAIL,
				time.Duration(1)*time.Minute),
			AccountFilter:  func(ac *account.Account) bool { return true },
			InboxFilter:    func(siteID int) bool { return !isLagou(siteID) },
			Status:         0,
			WorkerDuration: time.Duration(5) * time.Minute,
		}).SyncRun()
	})

	// start contact suber
	grace.Go(contact.Run)

	// start account push
	cron.NewDailyJob(cron.ANY, 0, 0, func(time.Time) {
		grace.Go(func(ctx context.Context) {
			pushAccount.SyncRun(ctx,
				register.New(register.ROLE_PUSH_ACCOUNT,
					time.Duration(10)*time.Minute))
		})
	})

	// start count email
	cron.NewDailyJob(cron.ANY, 0, 0, func(time.Time) {
		grace.Go(func(ctx context.Context) {
			noticeMail.SyncRun(ctx,
				register.New(register.ROLE_COUNT_MAIL,
					time.Duration(30)*time.Minute))
		})
	})

	grace.Run(time.Duration(10) * time.Second)
}
