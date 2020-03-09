package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"grabmail/grab/mail"
	"grabmail/models/account"
	"net"
	"strconv"
	"strings"
	"time"

	imapDriver "github.com/beiping96/go-imap"
	imapDriverClient "github.com/beiping96/go-imap/client"
	"github.com/pkg/errors"
	"gitlab.ifchange.com/data/cordwood/crypto/des3"
	"gitlab.ifchange.com/data/cordwood/log"
)

type imap struct {
	ctx   context.Context
	ac    *account.Account
	cli   *imapDriverClient.Client
	inbox string
}

func newIMAP(ctx context.Context, ac *account.Account) (Client, string, error) {
	cli := &imap{
		ctx: ctx,
		ac:  ac,
	}
	err := cli.initClient()
	if err != nil {
		return nil, "连接邮件服务器失败", errors.Wrap(err, "IMAP-INIT")
	}

	password, err := des3.Decrypt(ac.Password)
	if err != nil {
		return nil, "系统错误", errors.Wrap(err, "IMAP-Decrypt")
	}
	if err := cli.login(password); err != nil {
		ac.LoginFail(account.ConstLoginFailFlag + err.Error())
		return nil, "登陆邮件服务器失败", errors.Wrap(err, "IMAP-LOGIN")
	}
	ac.LoginSuccess()

	return cli, "", nil
}

func (c *imap) Close() {
	go func() {
		c.logout()
	}()
}

func (c *imap) GetIndexMail(ctx context.Context) (result map[string]*mail.IndexMail, err error) {
	result = make(map[string]*mail.IndexMail)
	allMailboxFolders := []string{}
	mailboxInfoChan := make(chan *imapDriver.MailboxInfo, 10)
	done := make(chan error)
	go func() {
		done <- c.cli.List("", "*", mailboxInfoChan)
		close(done)
	}()
	for mailboxInfo := range mailboxInfoChan {
		select {
		case <-ctx.Done():
			return nil, errors.Errorf("IMAP receive stop signal")
		default:
		}
		noSelect := false
		for _, attr := range mailboxInfo.Attributes {
			if strings.Contains(attr, "NoSelect") {
				noSelect = true
				break
			}
		}
		if noSelect {
			log.Infof("IMAP Account: %s skip inbox %s [NoSelect]",
				c.ac.UserName, mailboxInfo.Name)
			continue
		}
		allMailboxFolders = append(allMailboxFolders,
			mailboxInfo.Name)
	}
	if err := <-done; err != nil {
		return nil, errors.Wrap(err, "IMAP list mail box")
	}

	mailboxFolders, onPurpose := isReceiveMailBox(c.ac.UserName, c.ac.CustomInbox, allMailboxFolders)

	log.Infof("IMAP Account: %s all inbox %v used inbox %v onPurpose:%v",
		c.ac.UserName, allMailboxFolders, mailboxFolders, onPurpose)

	for _, mailboxFolder := range mailboxFolders {
		select {
		case <-ctx.Done():
			return nil, errors.Errorf("IMAP receive stop signal")
		default:
		}
		mBox, err := c.cli.Select(mailboxFolder, true)
		if err != nil {
			log.Warnf("IMAP mail box fetch uuids %s SELECT %s %v", c.ac.UserName, mailboxFolder, err)
			continue
		}
		if mBox.Messages == 0 {
			log.Debugf("IMAP mail box fetch uuids %s skip %s NULL INBOX", c.ac.UserName, mailboxFolder)
			continue
		}
		seqSet := new(imapDriver.SeqSet)
		seqSet.AddRange(1, mBox.Messages)
		messages := make(chan *imapDriver.Message, 10)
		done := make(chan error)
		go func() {
			done <- c.cli.Fetch(seqSet, []imapDriver.FetchItem{
				imapDriver.FetchUid,
				imapDriver.FetchItem(imapDriver.FetchUid),
			}, messages)
			close(done)
		}()
		for msg := range messages {
			indexMail := mail.NewIndexMail()
			indexMail.UUID = strconv.Itoa(int(msg.Uid))
			indexMail.Inbox = mailboxFolder
			indexMail.UID = msg.Uid

			result[indexMail.UUID] = indexMail
		}
		if err := <-done; err != nil {
			return nil, errors.Wrapf(err, "IMAP mail box fetch uuids fetch %s",
				mailboxFolder)
		}
	}
	return result, nil
}

func (c *imap) FetchInboxMail(ctx context.Context, indexMail []*mail.IndexMail) (result []*mail.InboxMail, err error) {
	for _, m := range indexMail {
		select {
		case <-ctx.Done():
			return
		default:
		}
		if c.inbox != m.Inbox {
			_, err = c.cli.Select(m.Inbox, true)
			if err != nil {
				log.Warnf("IMAP %s GetBody %+v select inbox error %v",
					c.ac.UserName, m, err)
				continue
			}
			c.inbox = m.Inbox
		}
		seqSet := new(imapDriver.SeqSet)
		seqSet.AddNum(m.UID)
		messageChan := make(chan *imapDriver.Message, 99)
		section := &imapDriver.BodySectionName{Peek: true}
		fetchItems := []imapDriver.FetchItem{section.FetchItem()}
		err := c.cli.UidFetch(seqSet, fetchItems, messageChan)
		if err != nil {
			log.Warnf("IMAP %s UidFetch %+v error %v",
				c.ac.UserName, m, err)
			continue
		}
		select {
		case <-ctx.Done():
			return nil, errors.Errorf("IMAP %+v receive stop signal", m)
		case msg := <-messageChan:
			if msg == nil {
				log.Warnf("IMAP %s FetchInboxMail %+v NIL *imapDriver.Message",
					c.ac.UserName, m)
				break
			}
			uuid := fmt.Sprintf("%d", msg.Uid)
			if uuid != m.UUID {
				log.Warnf("IMAP %s FetchInboxMail %+v UUIDs are not same",
					c.ac.UserName, m)
				break
			}
			body := msg.GetBody(section)
			if body == nil {
				log.Warnf("IMAP %s GetBody %+v mail body is nil",
					c.ac.UserName, m)
				break
			}
			result = append(result, &mail.InboxMail{
				Index: m,
				Body:  body,
			})
		case <-time.After(time.Duration(3) * time.Second):
			log.Warnf("IMAP %s FetchInboxMail %+v timeout",
				c.ac.UserName, m)
		}
	}
	return result, nil
}

type imapLogger struct{}

func (l *imapLogger) Printf(format string, v ...interface{}) {
	log.Warnf("IMAP "+format, v)
}

func (l *imapLogger) Println(v ...interface{}) {
	log.Warn(v...)
}

func (c *imap) initClient() (err error) {
	dialer := new(net.Dialer)
	dialer.Timeout = time.Duration(15) * time.Second
	dialer.Cancel = c.ctx.Done()

	if deadline, ok := c.ctx.Deadline(); ok {
		dialer.Timeout = deadline.Sub(time.Now())
	}

	defer func() {
		if c.cli == nil {
			return
		}
		c.cli.ErrorLog = new(imapLogger)
		if deadline, ok := c.ctx.Deadline(); ok {
			c.cli.Timeout = deadline.Sub(time.Now())
		}
	}()

	host := c.ac.MailServer
	port := c.ac.Port
	addr := fmt.Sprintf("%s:%d", host, port)

	if c.ac.Ssl == 0 {
		c.cli, err = imapDriverClient.DialWithDialer(dialer, addr)
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
	c.cli, err = imapDriverClient.DialWithDialerTLS(dialer, addr, tlsconfig)
	if err != nil {
		c.ac.LoginFail(account.ConstConnectFailFlag + err.Error())
		return errors.Wrap(err, "Dial-With-TLS")
	}
	return
}

func (c *imap) login(password string) (err error) {
	return c.cli.Login(c.ac.User, password)
}

func (c *imap) logout() {
	c.cli.Logout()
	c.cli.Terminate()
}
