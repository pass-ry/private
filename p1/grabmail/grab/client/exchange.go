package client

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"grabmail/grab/mail"
	"grabmail/models/account"
	"time"

	"github.com/pkg/errors"
	"gitlab.ifchange.com/data/cordwood/crypto/des3"
	"gitlab.ifchange.com/data/cordwood/encoding/json"
	"gitlab.ifchange.com/data/cordwood/redis"
)

const (
	constExchangePubQueue = "exchange_mail_pub_queue"
	constExchangeSubQueue = "exchange_mail_sub_queue"
)

type exchange struct {
	ctx      context.Context
	ac       *account.Account
	conn     redis.Client
	subQueue string
	isEND    bool
}

func (c *exchange) Close() { c.conn.Close() }

func (c *exchange) GetIndexMail(ctx context.Context) (result map[string]*mail.IndexMail, err error) {
	return
}

func newEXCHANGE(ctx context.Context, ac *account.Account, isLogin bool) (Client, string, error) {
	cli := &exchange{
		ctx: ctx,
		ac:  ac,
	}

	conn, err := redis.GetConstClient()
	if err != nil {
		return nil, "系统错误", errors.Wrap(err, "redis.GetConstClient")
	}
	cli.conn = conn
	cli.subQueue = constExchangeSubQueue + "_" + ac.UserName + "_" + fmt.Sprintf("%d", time.Now().Unix())

	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(time.Duration(10) * time.Minute)
	}
	deadline = deadline.Add(time.Duration(-30) * time.Second)

	password, err := des3.Decrypt(ac.Password)
	if err != nil {
		return nil, "系统错误", errors.Wrap(err, "EXCHANGE-Decrypt")
	}

	params := struct {
		UserName    string  `json:"username"`
		Name        *string `json:"name,omitempty"`
		Password    string  `json:"password"`
		Server      string  `json:"server"`
		Queue       string  `json:"queue"`
		IsLogin     bool    `json:"is_login"`
		Deadline    string  `json:"deadline"`
		ReceiveTime string  `json:"receive_time"`
	}{
		UserName:    ac.UserName,
		Password:    password,
		Server:      ac.MailServer,
		Queue:       cli.subQueue,
		IsLogin:     isLogin,
		Deadline:    deadline.Format("2006-01-02 15:04:05"),
		ReceiveTime: ac.LastReceiveTime.Format("2006-01-02 15:04:05"),
	}
	if len(ac.User) > 0 {
		name := ac.User
		params.Name = &name
	}

	task, err := json.Marshal(params, json.UnEscapeHTML())
	if err != nil {
		return nil, "系统错误", errors.Wrap(err, "json.Marshal")
	}

	_, err = cli.conn.Do("LPUSH", constExchangePubQueue, task)
	if err != nil {
		return nil, "系统错误", errors.Wrap(err, "redis LPUSH")
	}

	receive, err := cli.receive(ctx)
	if err != nil {
		return nil, "系统错误", errors.Wrap(err, "exchange receive")
	}
	if !receive.LoginStatus {
		ac.LoginFail(account.ConstLoginFailFlag + receive.LoginErrMsg)
		return nil, "登陆失败", errors.Errorf("Login Fail: %s", receive.LoginErrMsg)
	}
	ac.LoginSuccess()
	return cli, "", nil
}

func (c *exchange) FetchInboxMail(ctx context.Context, indexMail []*mail.IndexMail) (result []*mail.InboxMail, err error) {
	for {
		if c.isEND {
			return
		}
		receive, err := c.receive(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "c.receive")
		}
		if receive.Result == nil {
			continue
		}
		inboxmail := &mail.InboxMail{
			Index: mail.NewIndexMail(),
		}
		inboxmail.Index.Inbox = receive.Result.Inbox
		inboxmail.Index.UUID = receive.Result.UID
		inboxmail.Index.GraphID = receive.Result.UID
		mime, err := base64.StdEncoding.DecodeString(receive.Result.Base64MIME)
		if err != nil {
			return nil, errors.Wrap(err, "Decode Exchange MIME")
		}
		inboxmail.Body = bytes.NewBuffer(mime)
		result = append(result, inboxmail)
	}
}

// 1. login success package
// 2. resume package
// 3. end package
type receiveParams struct {
	LoginStatus bool   `json:"login_status"`
	LoginErrMsg string `json:"login_err"`
	IsEnd       bool   `json:"is_end"`
	Result      *struct {
		UID        string `json:"uid"`
		Inbox      string `json:"inbox"`
		Base64MIME string `json:"mime"`
	} `json:"resumes"`
}

func (c *exchange) receive(ctx context.Context) (*receiveParams, error) {
	var sleep time.Duration
	params := new(receiveParams)
	for {
		select {
		case <-ctx.Done():
			return nil, errors.Errorf("context cancel")
		case <-time.After(sleep):
			sleep = 0
		}
		result, err := c.conn.DoBytes("RPOP", c.subQueue)
		if err == c.conn.ErrNil() {
			sleep = time.Duration(3) * time.Second
			continue
		}
		if err != nil {
			return nil, errors.Wrap(err, "redis RPOP")
		}
		err = json.Unmarshal(result, params)
		if err != nil {
			return nil, errors.Wrap(err, "json Unmarshal")
		}
		c.isEND = params.IsEnd
		return params, nil
	}
}
