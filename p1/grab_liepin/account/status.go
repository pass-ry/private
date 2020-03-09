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
	Reason   string `json:"reason"`
	PlusCode int    `json:"error_code"`
	Source   int    `json:"source"`
}

func (ac *Account) Push(code, source int) error {
	// rebound 0成功 1失败
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
		return errors.Wrap(err, "liepin Account-Push Marshal")
	}

	conn, err := redis.GetConstClient()
	if err != nil {
		return errors.Wrap(err, "liepinAccount-Push redis.GetConstClient")

	}
	defer conn.Close()
	_, err = conn.Do("LPUSH", "liepin_account_forward_queue", pushMsg)
	if err != nil {
		return errors.Wrap(err, "liepinAccount-Push LPUSH")
	}

	return nil
}
