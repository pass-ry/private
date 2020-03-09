package controller

import (
	"strings"

	"github.com/pkg/errors"
	"gitlab.ifchange.com/data/cordwood/mysql"
	"gitlab.ifchange.com/data/cordwood/redis"
	handler "gitlab.ifchange.com/data/cordwood/rpc/rpc-handler"
)

func PinAccount(req *handler.Request, rsp *handler.Response) error {
	params := &struct {
		UserName   string `json:"username,omitempty"`
		PinHours   int    `json:"pin_hours,omitempty"`
		PinMinutes int    `json:"pin_minutes,omitempty"`
	}{}

	if err := req.Unmarshal(params); err != nil {
		return err
	}

	if len(params.UserName) == 0 {
		return handler.WrapError(errors.Errorf("NIL Username"),
			-1, "NIL Username")
	}
	params.UserName = strings.ToLower(params.UserName)

	row := mysql.GetConstConn().QueryRow(`SELECT id,is_deleted FROM accounts WHERE username=? LIMIT 1`, params.UserName)
	var (
		id        int
		isDeleted string
	)
	if err := row.Scan(&id, &isDeleted); err != nil {
		return handler.WrapError(errors.Wrap(err, "SQL Scan"),
			-1, "username不存在")
	}
	if id <= 0 || isDeleted != "N" {
		return handler.WrapError(errors.Errorf("account is deleted"),
			-1, "username被标记删除")
	}
	pinSeconds := params.PinHours * 3600
	if params.PinMinutes*60 > pinSeconds {
		pinSeconds = params.PinMinutes * 60
	}
	if pinSeconds <= 0 {
		pinSeconds = 60
	}

	conn, err := redis.GetConstClient()
	if err != nil {
		return handler.WrapError(err, -1, "系统错误")
	}
	defer conn.Close()

	_, err = conn.Do("SETEX", "pin_email_account", pinSeconds, params.UserName)
	if err != nil {
		return handler.WrapError(err, -1, "系统错误")
	}

	rsp.SetResults(true)
	return nil
}
