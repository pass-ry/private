package main

import (
	"context"
	"grabliepin/deliver"
	"grabliepin/puber"
	"grabliepin/rpc"
	"grabliepin/suber"
	"os"
	"time"

	"github.com/beiping96/grace"
	"gitlab.ifchange.com/data/cordwood/cfg"
	"gitlab.ifchange.com/data/cordwood/cfg/loader"
	"gitlab.ifchange.com/data/cordwood/crypto/des3"
	dfs "gitlab.ifchange.com/data/cordwood/fast-dfs"
	"gitlab.ifchange.com/data/cordwood/log"
	"gitlab.ifchange.com/data/cordwood/mysql"
	"gitlab.ifchange.com/data/cordwood/ps"
	"gitlab.ifchange.com/data/cordwood/redis"
)

func main() {
	// Basic
	// load config (required)
	loader.LoadCfgByEnv()
	// init log (required)
	log.Construct(cfg.GetCfgLog())
	// init grace (required)
	grace.Log(log.Infof)
	// init go-ps
	grace.Go(ps.Construct)

	// Useful driver
	// init MySQL (optional)
	var cfgMySQL mysql.Config
	if os.Getenv("ENV") == "prod" {
		cfgMySQL.Username = os.Getenv("MySQL_Username")
		cfgMySQL.Password = os.Getenv("MySQL_Password")
		cfgMySQL.Address = os.Getenv("MySQL_Address")
		cfgMySQL.Port = os.Getenv("MySQL_Port")
		cfgMySQL.DB = "grab_liepin"
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

	// init fast-dfs
	dfs.Construct(cfg.GetCfgFastDfs())

	// 3DES
	des3.Setup(cfg.GetCfgCustom().Get("DES3"),
		true /* open memory-cache */)

	// HTTPD
	grace.Go(func(ctx context.Context) { rpc.Run(ctx, cfg.GetCfgCustom().Get("rpc_port")) })

	// Puber
	grace.Go(puber.Run)

	// Suber
	grace.Go(suber.Run)

	// Deliver
	grace.Go(deliver.Run)

	// Run
	grace.Run(time.Duration(10) * time.Second)
}
