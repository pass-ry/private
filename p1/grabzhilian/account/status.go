package account

import (
	"encoding/json"
	"github.com/pkg/errors"
	"gitlab.ifchange.com/data/cordwood/crypto/des3"
	"gitlab.ifchange.com/data/cordwood/redis"
)

type pushTobParam struct {
	AppID    int    `json:"appid"`
	IP       string `json:"ip"`
	WebID    int    `json:"web_id"`
	UserName string `json:"username"`
	VipName  string `json:"vipname"`
	Rebound  int    `json:"rebound"`
	Password string `json:"passwd"`
	//Reason   interface{} `json:"failure_reason,omitempty"`
	Reason   string `json:"reason"`
	PlusCode int    `json:"error_code"`
	Source   int    `json:"source"`
}

type reason struct {
	ID     int    `json:"id"`
	Reason string `json:"reason"`
}

func (ac *Account) Push(code, source int) error {
	rebound := 0
	if ac.IsDeleted || code != 1 {
		rebound = 1
	}
	password, err := des3.Decrypt(ac.Password)
	if err != nil {
		return errors.Wrap(err, "des3.Decrypt")
	}

	p := pushTobParam{
		UserName: ac.Username,
		Rebound:  rebound,
		Password: password,
		PlusCode: code,
		Source:   source,
	}

	if rebound > 0 {
		p.Reason = ac.Msg
	}

	pushMsg, err := json.Marshal(p)
	if err != nil {
		return errors.Wrap(err, "zhilian Account-Push Marshal")
	}

	conn, err := redis.GetConstClient()
	if err != nil {
		return errors.Wrap(err, "zhilianAccount-Push redis.GetConstClient")

	}
	defer conn.Close()

	_, err = conn.Do("LPUSH", "zhilian_account_forward_queue", pushMsg)
	if err != nil {
		return errors.Wrap(err, "zhilianAccount-Push LPUSH")
	}

	return nil

}

/*
func (ac *Account) Push(code int) error {
	// rebound 0成功 1失败
	rebound := 0
	if ac.IsDeleted {
		rebound = 1
	}

	password, err := des3.Decrypt(ac.Password)
	if err != nil {
		return errors.Wrap(err, "des3.Decrypt")
	}

	url := cfg.GetCfgCustom().Get("status_push")
	req := curl.NewRequest()
	req.SetC("setRebound")
	p := pushTobParam{
		AppID:    10,
		IP:       "192.168.8.23",
		WebID:    2,
		UserName: ac.Username,
		Rebound:  rebound,
		Password: password,
		PlusCode: code,
	}

	// 失败才传reason
	// 成功不需要这个字段
	if rebound >= 1 {
		p.Reason = reason{
			ID:     0,
			Reason: ac.Msg,
		}
	}
	req.SetP(p)

	rsp := curl.NewResponse()
	err = curl.Curl(url, req, rsp)
	if err != nil {
		return errors.Wrap(err, "Account Push")
	}
	if rsp.GetErrNo() != 0 {
		return errors.Errorf("Call %s Got %d %s",
			url, rsp.GetErrNo(), rsp.GetErrMsg())
	}
	return nil
}
*/
