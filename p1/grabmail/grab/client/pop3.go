package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"grabmail/grab/mail"
	"grabmail/models/account"
	"net"
	"strings"
	"time"

	pop3Driver "github.com/linuxtea/go-pop3"
	"github.com/pkg/errors"
	"gitlab.ifchange.com/data/cordwood/crypto/des3"
	"gitlab.ifchange.com/data/cordwood/log"
)

type pop3 struct {
	ctx context.Context
	ac  *account.Account
	cli *pop3Driver.Client
}

func newPOP3(ctx context.Context, ac *account.Account) (Client, string, error) {
	cli := &pop3{
		ctx: ctx,
		ac:  ac,
	}
	err := cli.initClient()
	if err != nil {
		return nil, "连接邮件服务器失败", errors.Wrap(err, "POP3-INIT")
	}

	password, err := des3.Decrypt(ac.Password)
	if err != nil {
		return nil, "系统错误", errors.Wrap(err, "POP3-Decrypt")
	}
	if err := cli.login(password); err != nil {
		ac.LoginFail(account.ConstLoginFailFlag + err.Error())
		return nil, "登陆邮件服务器失败", errors.Wrap(err, "POP3-LOGIN")
	}
	ac.LoginSuccess()
	return cli, "", nil
}

func (c *pop3) Close() {
	go func() {
		c.logout()
	}()
}

func (c *pop3) GetIndexMail(ctx context.Context) (map[string]*mail.IndexMail, error) {
	pop3mails, err := c.cli.UidlAll()
	if err != nil {
		return nil, errors.Wrap(err, "pop3 UidlAll")
	}
	mails := make(map[string]*mail.IndexMail)
	for _, m := range pop3mails {
		indexMail := mail.NewIndexMail()
		indexMail.UUID = m.Uid
		indexMail.NumberID = m.Number
		indexMail.Inbox = "INBOX"

		mails[m.Uid] = indexMail
	}
	return mails, nil
}

func (c *pop3) FetchInboxMail(ctx context.Context, indexMail []*mail.IndexMail) ([]*mail.InboxMail, error) {
	inboxMail := []*mail.InboxMail{}
	for _, m := range indexMail {
		select {
		case <-ctx.Done():
			return nil, errors.Errorf("POP3 receive stop signal")
		default:
		}
		mailBody, err := c.cli.Retr(m.NumberID)
		if err != nil {
			log.Warnf("POP3 %s %+v Retr error %v",
				c.ac.UserName, m, err)
			continue
		}
		checkNumberID, checkUUID, err := c.cli.Uidl(m.NumberID)
		if err != nil || checkNumberID != m.NumberID || checkUUID != m.UUID {
			log.Warnf("POP3 %s %+v UUIDs are not same UUID:%s NumberID:%d err:%v",
				c.ac.UserName, m, checkUUID, checkNumberID, err)
			continue
		}
		inboxMail = append(inboxMail, &mail.InboxMail{
			Index: m,
			Body:  bytes.NewBufferString(mailBody),
		})
	}
	return inboxMail, nil
}

func (c *pop3) initClient() (err error) {
	timedeadline := int64(15)

	dialer := new(net.Dialer)
	dialer.Timeout = time.Duration(timedeadline) * time.Second

	if deadline, ok := c.ctx.Deadline(); ok {
		timedeadline = int64(deadline.Sub(time.Now()).Seconds())
	}

	host := c.ac.MailServer
	port := c.ac.Port
	addr := fmt.Sprintf("%s:%d", host, port)

	if c.ac.Ssl == 0 {
		c.cli, err = pop3Driver.DialWithDialer(timedeadline, dialer, addr)
		if err != nil {
			c.ac.LoginFail(account.ConstConnectFailFlag + err.Error())
			return errors.Wrap(err, "Dial")
		}
		return
	}
	tlsconfig := new(tls.Config)
	tlsconfig.InsecureSkipVerify = true
	if strings.Contains(host, "rondaful.com") ||
		strings.Contains(host, "jinke.com") {
		tlsconfig.ServerName = host
	}
	c.cli, err = pop3Driver.DialWithDialerTLS(timedeadline, dialer, addr, tlsconfig)
	if err != nil {
		c.ac.LoginFail(account.ConstConnectFailFlag)
		return errors.Wrap(err, "Dial-With-TLS")
	}
	return
}

func (c *pop3) login(password string) error {
	if err := c.cli.User(c.ac.User); err != nil {
		return errors.Wrap(err, "USER")
	}
	if err := c.cli.Pass(password); err != nil {
		return errors.Wrap(err, "PASS")
	}
	return nil
}

func (c *pop3) logout() {
	if err := c.cli.Quit(); err != nil {
		return
	}
	if err := c.cli.Close(); err != nil {
		return
	}
}
