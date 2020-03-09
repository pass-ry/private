package rpc

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/beiping96/grace"
	"gitlab.ifchange.com/data/cordwood/cfg"
	"gitlab.ifchange.com/data/cordwood/cfg/loader"
	"gitlab.ifchange.com/data/cordwood/crypto/des3"
	"gitlab.ifchange.com/data/cordwood/mysql"
	"gitlab.ifchange.com/data/cordwood/redis"
	handler "gitlab.ifchange.com/data/cordwood/rpc/rpc-handler"
)

func TestMain(m *testing.M) {
	loader.LoadCfgInDev("grabzhilian")
	mysql.Construct(cfg.GetCfgMySQL())
	redis.Construct(cfg.GetCfgRedis())
	des3.Setup(cfg.GetCfgCustom().Get("DES3"), true)
	m.Run()
}

func TestZhilianPhoneMsg(t *testing.T) {
	p := `{"username":"ifchange01", "phone":"17739872745","msg":"this is a phone msg"}`
	a := struct {
		C string          `json:"c"`
		M string          `json:"m"`
		P json.RawMessage `json:"p"`
	}{
		C: "",
		M: "",
	}

	json.Unmarshal([]byte(p), &a.P)

	req := &handler.Request{
		Request: a,
	}

	t.Log(req)

	rsp := &handler.Response{}
	t.Run("zhilian: ", func(t *testing.T) {
		if err := zhilianPhoneMsg(req, rsp); err != nil {
			t.Errorf("zhilianBind err = %v", err)
		} else {
			t.Log("success...")
		}
	})
	grace.Run(time.Duration(10) * time.Second)
}

/*
func TestZhilianBind(t *testing.T) {
	p := `{"username":"ifchange01", "password":"123456abc","receive_time":"2019-06-26 16:08:42"}`
	a := struct {
		C string          `json:"c"`
		M string          `json:"m"`
		P json.RawMessage `json:"p"`
	}{
		C: "",
		M: "",
	}

	json.Unmarshal([]byte(p), &a.P)

	req := &handler.Request{
		Request: a,
	}

	t.Log(req)

	rsp := &handler.Response{}
	t.Run("zhilian: ", func(t *testing.T) {
		if err := zhilianTob(req, rsp); err != nil {
			t.Errorf("zhilianBind err = %v", err)
		} else {
			t.Log("success...")
		}
	})
	grace.Run(time.Duration(10) * time.Second)
}


func TestZhilianCheckPassword(t *testing.T) {
	p := `{"username":"zhang.yuncheng@weidaitouch.com", "password":"123456abc"}`
	a := struct {
		C string          `json:"c"`
		M string          `json:"m"`
		P json.RawMessage `json:"p"`
	}{
		C: "",
		M: "",
	}

	json.Unmarshal([]byte(p), &a.P)

	req := &handler.Request{
		Request: a,
	}

	t.Log(req)

	rsp := &handler.Response{}
	t.Run("zhilian: ", func(t *testing.T) {
		if err := zhilianCheckPassword(req, rsp); err != nil {
			t.Errorf("zhilianBind err = %v", err)
		} else {
			t.Log("check success...")
		}
	})
}
*/
