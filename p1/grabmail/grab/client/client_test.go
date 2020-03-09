package client

import (
	"testing"

	"gitlab.ifchange.com/data/cordwood/cfg"
	"gitlab.ifchange.com/data/cordwood/cfg/loader"
	"gitlab.ifchange.com/data/cordwood/crypto/des3"
	"gitlab.ifchange.com/data/cordwood/mysql"
)

func TestMain(m *testing.M) {
	loader.LoadCfgInDev("grabmail")
	mysql.Construct(cfg.GetCfgMySQL())
	des3.Setup(cfg.GetCfgCustom().Get("DES3"),
		true /* open memory-cache */)

	m.Run()
}

func Test_isReceiveMailBox(t *testing.T) {
	got, got1 := isReceiveMailBox("a", "", []string{"Deleted Messages"})
	t.Log(got, got1)
}
