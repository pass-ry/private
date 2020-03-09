package controller

import (
	"database/sql"

	"gitlab.ifchange.com/data/cordwood/mysql"
	handler "gitlab.ifchange.com/data/cordwood/rpc/rpc-handler"
)

func mailStatus(req *handler.Request, rsp *handler.Response, p *params) error {
	var (
		isDeleted string
		errMsg    string
	)
	err := mysql.GetConstConn().QueryRow(`SELECT is_deleted,msg FROM accounts WHERE username=? LIMIT 1`,
		p.UserName).Scan(&isDeleted, &errMsg)
	reply := &struct {
		Msg    string `json:"msg"`
		Status bool   `json:"status"`
	}{}
	if err == sql.ErrNoRows {
		reply.Status = false
		reply.Msg = "not bind"
		rsp.SetResults(reply)
		return nil
	}
	if err != nil {
		return handler.WrapError(err, -1, "系统错误")
	}
	if isDeleted == "N" {
		reply.Status = true
		rsp.SetResults(reply)
		return nil
	}
	reply.Status = false
	reply.Msg = errMsg
	rsp.SetResults(reply)
	return nil
}
