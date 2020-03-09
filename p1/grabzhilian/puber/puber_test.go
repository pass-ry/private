package puber

import (
	"testing"
	"time"
	//	"context"
	"github.com/beiping96/grace"

	"gitlab.ifchange.com/data/cordwood/cfg"
	"gitlab.ifchange.com/data/cordwood/cfg/loader"
	"gitlab.ifchange.com/data/cordwood/mysql"
	"gitlab.ifchange.com/data/cordwood/redis"
)

func TestMain(m *testing.M) {
	loader.LoadCfgInDev("grabzhilian")
	redis.Construct(cfg.GetCfgRedis())
	mysql.Construct(cfg.GetCfgMySQL())
	m.Run()
}

func TestRun(t *testing.T) {
	t.Run("Puber", func(t *testing.T) {
		t.Log("action")
		grace.Go(Run)
	})

	grace.Run(time.Duration(10) * time.Second)
}
