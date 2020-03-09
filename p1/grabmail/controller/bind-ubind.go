package controller

import (
	"context"
	"database/sql"
	"fmt"
	"grabmail/grab/client"
	"grabmail/models/account"
	"grabmail/models/inbox"
	accountPush "grabmail/push/account"
	"runtime"
	"strings"
	"time"

	"github.com/beiping96/grace"
	"github.com/bluele/gcache"
	"github.com/pkg/errors"
	"gitlab.ifchange.com/data/cordwood/crypto/des3"
	"gitlab.ifchange.com/data/cordwood/log"
	"gitlab.ifchange.com/data/cordwood/mysql"
	handler "gitlab.ifchange.com/data/cordwood/rpc/rpc-handler"
)

func mailUnbind(req *handler.Request, rsp *handler.Response, p *params) error {
	userName := p.UserName
	row := mysql.GetConstConn().QueryRow(`SELECT id FROM accounts WHERE username=? LIMIT 1`, userName)
	var id int
	if err := row.Scan(&id); err != nil {
		return handler.WrapError(errors.Wrap(err, "SQL Scan"),
			85084009, "username不存在")
	}

	if id == 0 {
		return handler.WrapError(errors.Errorf("AccountID is zero"),
			85084009, "username不存在")
	}
	if _, err := mysql.GetConstConn().Exec(`UPDATE accounts
		SET is_deleted='Y', msg='', errcount=0, updated_at=?
		WHERE username=? LIMIT 1`, time.Now().Format("2006-01-02 15:04:05"), userName); err != nil {
		return handler.WrapError(errors.Wrap(err, "SQL Exec"),
			85084005, "解绑失败")
	}
	rsp.SetResults(true)
	return nil
}

func mailBind(req *handler.Request, rsp *handler.Response, p *params,
	splitUserName []string) error {
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
	newAC.Password = password
	newAC.MailServer = p.MailServer
	newAC.Port = p.Port
	newAC.Ssl = p.Ssl
	newAC.LastReceiveTime = time.Unix(p.LastReceiveTime, 0)

	switch p.ServerType {
	case "imap":
		newAC.User = p.UserName
		newAC.Type = 0
	case "pop3":
		newAC.User = p.UserName
		newAC.Type = 1
	case "exchange":
		if len(p.Name) > 0 {
			newAC.User = p.Name
		}
		newAC.Type = 2
	default:
		return handler.WrapError(errors.Errorf("Unknown server_type %s", p.ServerType), 85084006, "server_type非法")
	}

	// find old account
	oldAC := account.NewAC()
	err = mysql.GetConstConn().QueryRow(`SELECT
		id,username,user,password,type,
		mail_server,port,accounts.ssl,
		last_receive_time
		FROM accounts
		WHERE username=? ORDER BY id DESC LIMIT 1`, newAC.UserName).Scan(&oldAC.ID,
		&oldAC.UserName, &oldAC.User, &oldAC.Password, &oldAC.Type,
		&oldAC.MailServer, &oldAC.Port, &oldAC.Ssl, &oldAC.LastReceiveTime)
	if err == sql.ErrNoRows {
		// if old account is not exist
		// safe in chan and return
		return inQueue(req, rsp, &bindMessage{
			newAC:  newAC,
			source: p.Source,
		})
	}
	// handle sql error
	if err != nil {
		return handler.WrapError(errors.Wrap(err, "SYS-SQL-SELECT"), 85085000, "系统错误")
	}

	if oldAC.TYPE() != newAC.TYPE() {
		// if protocol changed
		// NOT Allowed
		return handler.WrapError(errors.Errorf("try change mail %s protocol %s => %s ",
			oldAC.UserName, oldAC.TYPE(), newAC.TYPE()), 85084016, "mail协议变化")
	}
	return inQueue(req, rsp, &bindMessage{
		newAC:  newAC,
		oldAC:  oldAC,
		source: p.Source,
	})
}

func inQueue(req *handler.Request, rsp *handler.Response, bm *bindMessage) error {
	if bm == nil {
		return handler.WrapError(errors.Errorf("NIL bindMessage"), 85085001, "系统错误")
	}
	if bm.newAC == nil {
		return handler.WrapError(errors.Errorf("NIL bindMessage.newAC"), 85085001, "系统错误")
	}
	if bm.newAC.UserName == "" {
		return handler.WrapError(errors.Errorf("NIL bindMessage.newAC.UserName"), 85085001, "系统错误")
	}

	if _, err := bindLock.Get(bm.newAC.UserName); err != gcache.KeyNotFoundError {
		return handler.WrapError(errors.Errorf("%s 重复提交", bm.newAC.UserName), 85085001, "重复提交")
	}
	select {
	case bindChan <- bm:
		bindLock.Set(bm.newAC.UserName, true)
		rsp.SetResults(true)
		return nil
	default:
		return handler.WrapError(errors.Errorf("FULL replace BIND-CHANNEL"), 85085001, "系统错误")
	}
}

var (
	bindLock                   = gcache.New(1000).Expiration(time.Minute).Build()
	bindChan chan *bindMessage = make(chan *bindMessage, 999)
)

type bindMessage struct {
	oldAC  *account.Account
	newAC  *account.Account
	source int
}

func BindHandler(ctx context.Context) {
	var sleep time.Duration
	for {
		select {
		case bm := <-bindChan:
			grace.Go(func(ctx context.Context) {
				ctx, cancel := context.WithTimeout(ctx, time.Duration(2)*time.Minute)
				defer cancel()
				defer bindLock.Remove(bm.newAC.UserName)
				bm.handle(ctx)
			})
			sleep = 0
		default:
			sleep = time.Duration(3) * time.Second
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(sleep):
		}
	}
}

func (bm *bindMessage) handle(ctx context.Context) {
	// protect check
	if bm == nil {
		log.Errorf("Bind-handle receive nil bind message")
		return
	}
	if bm.newAC == nil {
		log.Errorf("Bind-handle receive nil new account bind message")
		return
	}
	// recover in handle
	// because exec-handle goroutine doesn't have restart method
	// allow one message exec fail
	// but avoid full bind channel
	log.Infof("Bind-handle receive old:%+v new:%+v and start exec",
		bm.oldAC, bm.newAC)
	defer func() {
		log.Infof("Bind-handle receive old:%+v new:%+v and stop exec",
			bm.oldAC, bm.newAC)
		err := recover()
		if err == nil {
			return
		}
		buf := make([]byte, 1<<10)
		num := runtime.Stack(buf, false)
		log.Errorf("Bind-handle old:%+v new:%+v PANIC %v %v %v",
			bm.oldAC, bm.newAC, err, num, string(buf))
	}()

	if bm.oldAC == nil {
		// create new account
		bm.handleCreate(ctx)
		return
	}
	bm.handleReplace(ctx)
	return
}

func (bm *bindMessage) handleCreate(ctx context.Context) {
	ac := bm.newAC
	if err := tryLogin(ctx, ac); err != nil {
		accountPush.Push(nil, bm.source, ac, 2, "Login Fail - "+err.Error())
		return

	}
	_, err := mysql.GetConstConn().ExecContext(ctx,
		`INSERT INTO accounts
		(accounts.username,accounts.user,accounts.password,accounts.type,
		accounts.mail_server,accounts.port,accounts.ssl,accounts.last_receive_time,updated_at)
		VALUES(?,?,?,?,?,?,?,?,?)`,
		ac.UserName, ac.User, ac.Password, ac.Type,
		ac.MailServer, ac.Port, ac.Ssl,
		ac.LastReceiveTime.Format("2006-01-02 15:04:05"), time.Now().Format("2006-01-02 15:04:05"))
	if err != nil {
		log.Errorf("Bind-handle SQL-INSERT error %v", err)
		accountPush.Push(nil, bm.source, ac, 2, "SYS error")
		return
	}
	accountPush.Push(nil, bm.source, ac, 0, "Bind Success")
	return
}

func (bm *bindMessage) handleReplace(ctx context.Context) {
	if err := tryLogin(ctx, bm.newAC); err != nil {
		oldCanUsed := false
		if err := tryLogin(ctx, bm.oldAC); err == nil {
			oldCanUsed = true
		}
		if oldCanUsed {
			accountPush.Push(nil, bm.source, bm.oldAC, 0,
				"exist account can works, but new one does not")
			return
		}
		accountPush.Push(nil, bm.source, bm.newAC, 2, "Login Fail - "+err.Error())
		return
	}

	// new account login success
	accountID := bm.oldAC.ID
	bm.newAC.ID = accountID
	// handle last receive time
	if bm.oldAC.LastReceiveTime.Before(bm.newAC.LastReceiveTime) {
		bm.newAC.LastReceiveTime = bm.oldAC.LastReceiveTime
	}

	now := time.Now()
	earlyReceiveTime := time.Date(now.Year()-1, now.Month(), now.Day(),
		now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), now.Location())
	if bm.newAC.LastReceiveTime.Before(earlyReceiveTime) {
		bm.newAC.LastReceiveTime = earlyReceiveTime
	}
	if bm.newAC.LastReceiveTime.Before(bm.oldAC.LastReceiveTime) {
		// update inboxes
		_, err := mysql.GetConstConn().ExecContext(ctx,
			fmt.Sprintf(`UPDATE %s SET is_deleted='Y'
			WHERE icdc_id=0 AND msg=? AND
			account_id=? AND send_time>? AND send_time<?`, bm.oldAC.MailTable()),
			inbox.ConstBeforeReceiveTimeFunc(), accountID,
			bm.newAC.LastReceiveTime, bm.oldAC.LastReceiveTime)
		if err != nil {
			log.Errorf("Bind-handle SQL-UPDATE-inboxes error %v", err)
			accountPush.Push(nil, bm.source, bm.newAC, 2, "SYS error")
			return
		}
	}
	// update accounts
	_, err := mysql.GetConstConn().ExecContext(ctx,
		`UPDATE accounts SET
		accounts.is_deleted='N',accounts.errcount=0,accounts.msg='',accounts.last_receive_time=?,
		accounts.ssl=?,accounts.port=?,accounts.mail_server=?,accounts.type=?,
		accounts.password=?,accounts.user=?,accounts.updated_at=?
		WHERE id=? LIMIT 1`,
		bm.newAC.LastReceiveTime, bm.newAC.Ssl, bm.newAC.Port,
		bm.newAC.MailServer, bm.newAC.Type, bm.newAC.Password,
		bm.newAC.User, time.Now().Format("2006-01-02 15:04:05"), accountID)
	if err != nil {
		log.Errorf("Bind-handle SQL-UPDATE-accounts error %v", err)
		accountPush.Push(nil, bm.source, bm.newAC, 2, "SYS error")
		return
	}
	accountPush.Push(nil, bm.source, bm.newAC, 0, "Replace Success")
	return
}

func tryLogin(ctx context.Context, ac *account.Account) error {
	cli, errMsg, err := client.NewClient(ctx, ac, true)
	if err == nil {
		cli.Close()
		return nil
	}
	err = errors.Errorf("%s %v", errMsg, err)
	log.Errorf("tryLogin error %+v NewClient %v", ac, err)
	if ac.TYPE() == account.EXCHANGE {
		return err
	}
	// if default login name is failed, try sub-username
	ac.User = strings.Split(ac.UserName, "@")[0]
	cli, errMsg, subUserNameErr := client.NewClient(ctx, ac, true)
	if subUserNameErr == nil {
		cli.Close()
		return nil
	}
	return err
}
