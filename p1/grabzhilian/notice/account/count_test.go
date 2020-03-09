package account

import (
	"context"
	"testing"

	"gitlab.ifchange.com/data/cordwood/cfg"
	"gitlab.ifchange.com/data/cordwood/cfg/loader"
	"gitlab.ifchange.com/data/cordwood/log"
	"gitlab.ifchange.com/data/cordwood/mysql"
)

func TestMain(m *testing.M) {
	loader.LoadCfgInDev("grabzhilian")
	log.Construct(cfg.GetCfgLog())
	mysql.Construct(cfg.GetCfgMySQL())
	if err := mysql.GetConstConn().Ping(); err != nil {
		panic(err)
	}

	m.Run()
}

func TestSyncRun(t *testing.T) {
	SyncRun(context.Background())
}
