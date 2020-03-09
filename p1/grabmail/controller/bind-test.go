package controller

import (
	"context"
	"database/sql"
	"grabmail/models/account"
	"time"

	"github.com/pkg/errors"
	"gitlab.ifchange.com/data/cordwood/crypto/des3"
	"gitlab.ifchange.com/data/cordwood/mysql"
	handler "gitlab.ifchange.com/data/cordwood/rpc/rpc-handler"
)

func mailTest(req *handler.Request, rsp *handler.Response, p *params, splitUserName []string) error {
	// args check
	switch {
	case p.Password == "":
		return handler.WrapError(errors.Errorf("Unknown password"), 85084013, "password无效")
	case p.MailServer == "":
		return handler.WrapError(errors.Errorf("Unknown mail_server"), 85084012, "mail_server无效")
	case p.Port == 0 && p.ServerType != "exchange":
		return handler.WrapError(errors.Errorf("Unknown port"), 85084011, "port无效")
	case p.Ssl != 0 && p.Ssl != 1 && p.ServerType != "exchange":
		return handler.WrapError(errors.Errorf("Unknown ssl"), 85084014, "ssl无效")
	case p.LastReceiveTime < 0:
		return handler.WrapError(errors.Errorf("Unknown last_receive_time"), 85084010, "last_receive_time无效")
	case p.ServerType != "pop3" && p.ServerType != "imap" && p.ServerType != "exchange":
		return handler.WrapError(errors.Errorf("Unknown server_type"), 85084015, "server_type无效")
	}
	// password Encrypt
	password, err := des3.Encrypt(p.Password)
	if err != nil {
		return handler.WrapError(err, 85085000, "系统错误")
	}
	newAC := account.NewAC()
	newAC.UserName = p.UserName
	newAC.User = p.UserName
	newAC.Password = password
	newAC.MailServer = p.MailServer
	newAC.Port = p.Port
	newAC.Ssl = p.Ssl
	newAC.LastReceiveTime = time.Unix(p.LastReceiveTime, 0)

	switch p.ServerType {
	case "imap":
		newAC.Type = 0
	case "pop3":
		newAC.Type = 1
	case "exchange":
		newAC.Type = 2
	default:
		return handler.WrapError(errors.Errorf("Unknown server_type %s", p.ServerType), 85084006, "server_type非法")
	}

	oldAC := account.NewAC()
	err = mysql.GetConstConn().QueryRow(`SELECT
		id,username,user,password,type,
		mail_server,port,accounts.ssl,
		last_receive_time
		FROM accounts
		WHERE username=? ORDER BY id DESC LIMIT 1`, newAC.UserName).Scan(&oldAC.ID,
		&oldAC.UserName, &oldAC.User, &oldAC.Password, &oldAC.Type,
		&oldAC.MailServer, &oldAC.Port, &oldAC.Ssl, &oldAC.LastReceiveTime)
	if err != sql.ErrNoRows && err != nil {
		return handler.WrapError(errors.Wrap(err, "SYS-SQL-SELECT"), 85085000, "系统错误")
	}
	if err == nil && oldAC.TYPE() != newAC.TYPE() {
		return handler.WrapError(errors.Errorf("try change mail %s protocol %s => %s ",
			oldAC.UserName, oldAC.TYPE(), newAC.TYPE()), 85084016, "mail协议变化")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = tryLogin(ctx, newAC)
	if err == nil {
		rsp.SetResults("Check Bind PASS")
		return nil
	}

	rsp.SetErrNo(-1)
	rsp.SetErrMsg("Check Bind Failed")
	rsp.SetResults(err)
	return nil
}
