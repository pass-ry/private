package resume

import (
	"gitlab.ifchange.com/data/cordwood/cfg"
	"gitlab.ifchange.com/data/cordwood/cfg/loader"
	"gitlab.ifchange.com/data/cordwood/mysql"
	"testing"
)

func TestMain(m *testing.M) {
	loader.LoadCfgInDev("grabzhilian")
	mysql.Construct(cfg.GetCfgMySQL())
	m.Run()
}

func TestGetUserName(t *testing.T) {
	t.Run("zhilian", func(t *testing.T) {
		name, id, err := getTableName("1uju44955362n")
		t.Log(name, id, err)
	})
}

func TestGetData(t *testing.T) {
	t.Run("zhilian", func(t *testing.T) {
		var testReq tobGetResumeListReq
		testReq.Username = "uju44955362n"
		testReq.Password = "gui know"
		testReq.SiteID = 1
		testReq.Type = "webkit"
		testReq.StartTimeDuration = "2019-12-25"
		testReq.EndTimeDuration = "2019-12-26"
		//		testReq.Offset = 10
		testReq.SizeLimit = 40

		getResumeRes(&testReq)
	})
}
