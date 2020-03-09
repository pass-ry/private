package account

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"gitlab.ifchange.com/data/cordwood/encoding/json"
	"gitlab.ifchange.com/data/cordwood/log"
	"gitlab.ifchange.com/data/cordwood/mysql"
)

const (
	constTableNum    = 64
	constMaxErrCount = 36
)

var (
	acPOOL = sync.Pool{
		New: func() interface{} { return new(Account) },
	}
	nilAC = Account{}
)

func NewAC() *Account {
	return acPOOL.Get().(*Account)
}

func (ac *Account) Close() {
	if ac == nil {
		return
	}
	*ac = nilAC
	acPOOL.Put(ac)
}

type Account struct {
	ID              int       `json:"id"`
	UserName        string    `json:"username"`
	User            string    `json:"user"`
	Password        string    `json:"password"`
	Type            int       `json:"type"`
	MailServer      string    `json:"mail_server"`
	Port            int       `json:"port"`
	Ssl             int       `json:"ssltiny"`
	Msg             string    `json:"msg"`
	Status          int       `json:"status"`
	LastReceiveTime time.Time `json:"last_receive_time"`
	LastCrawlerTime time.Time `json:"last_crawler_time"`
	ErrCount        int       `json:"errcount"`
	CustomInbox     string    `json:"custom_inbox"`
}

func (ac *Account) String() string {
	if ac == nil {
		return "NIL Account"
	}
	all, _ := json.Marshal(ac)
	allMap := make(map[string]interface{})
	json.Unmarshal(all, &allMap)
	allMap["password"] = len(allMap["password"].(string))
	all, _ = json.Marshal(allMap)
	return string(all)
}

func GetCanUsedAccountByUsername(username string) (*Account, error) {
	ac := NewAC()
	return ac, mysql.GetConstConn().QueryRow(`SELECT
	accounts.id,accounts.username,accounts.user,accounts.password,
	accounts.type,accounts.mail_server,accounts.port,accounts.ssl,
	accounts.msg,accounts.status,accounts.last_receive_time,accounts.errcount,
	accounts.custom_inbox
	FROM accounts WHERE is_deleted="N" AND username=?
	ORDER BY id DESC LIMIT 1`,
		username).Scan(&ac.ID, &ac.UserName, &ac.User, &ac.Password,
		&ac.Type, &ac.MailServer, &ac.Port, &ac.Ssl, &ac.Msg, &ac.Status,
		&ac.LastReceiveTime, &ac.ErrCount, &ac.CustomInbox)
}

func GetAccountByID(id int) (*Account, error) {
	ac := NewAC()
	return ac, mysql.GetConstConn().QueryRow(`SELECT
	accounts.id,accounts.username,accounts.user,accounts.password,
	accounts.type,accounts.mail_server,accounts.port,accounts.ssl,
	accounts.msg,accounts.status,accounts.last_receive_time,accounts.errcount,
	accounts.custom_inbox
	FROM accounts WHERE is_deleted="N" AND id=?
	ORDER BY id DESC LIMIT 1`,
		id).Scan(&ac.ID, &ac.UserName, &ac.User, &ac.Password,
		&ac.Type, &ac.MailServer, &ac.Port, &ac.Ssl, &ac.Msg, &ac.Status,
		&ac.LastReceiveTime, &ac.ErrCount, &ac.CustomInbox)
}

func GetCanUsedAll() ([]*Account, error) {
	rows, err := mysql.GetConstConn().Query(`SELECT
	accounts.id,accounts.username,accounts.user,accounts.password,
	accounts.type,accounts.mail_server,accounts.port,accounts.ssl,
	accounts.msg,accounts.status,accounts.last_receive_time,accounts.errcount,
	accounts.custom_inbox
	FROM accounts WHERE is_deleted="N"`)
	if err != nil {
		return nil, errors.Wrap(err, "SQL accounts Table Select")
	}
	defer rows.Close()

	metrics := make(map[string]int)

	acs := []*Account{}
	for rows.Next() {
		ac := NewAC()
		if err := rows.Scan(&ac.ID, &ac.UserName, &ac.User, &ac.Password,
			&ac.Type, &ac.MailServer, &ac.Port, &ac.Ssl, &ac.Msg, &ac.Status,
			&ac.LastReceiveTime, &ac.ErrCount, &ac.CustomInbox); err != nil {
			return nil, errors.Wrap(err, "SQL accounts Table Scan")
		}
		ac.UserName = strings.TrimSpace(ac.UserName)
		ac.User = strings.TrimSpace(ac.User)
		ac.Password = strings.TrimSpace(ac.Password)
		ac.MailServer = strings.TrimSpace(ac.MailServer)
		acs = append(acs, ac)
		metrics[ac.TYPE().String()] += 1
	}

	for protocol, num := range metrics {
		metricsCanUsedAccounts(protocol, num)
	}

	return acs, nil
}

type ClientType int

func (t ClientType) String() string {
	switch t {
	case POP3:
		return "pop3"
	case IMAP:
		return "IMAP"
	case EXCHANGE:
		return "EXCHANGE"
	}
	return fmt.Sprintf("Unknown Client Type %d", int(t))
}

const (
	IMAP     ClientType = 0
	POP3     ClientType = 1
	EXCHANGE ClientType = 2
)

func (ac *Account) TYPE() ClientType {
	switch ac.Type {
	case 0:
		return IMAP
	case 1:
		return POP3
	case 2:
		return EXCHANGE
	}
	return ClientType(ac.Type)
}

func (ac *Account) MailTable() string {
	return fmt.Sprintf("inboxes_%d", ac.ID%constTableNum)
}

func (ac *Account) UpdateLastCrawlerTime(ctx context.Context) error {
	if ac.ID == 0 {
		return nil
	}
	ac.LastCrawlerTime = time.Now()
	_, err := mysql.GetConstConn().ExecContext(ctx,
		`UPDATE accounts SET last_crawler_time=? WHERE id=? LIMIT 1`,
		ac.LastCrawlerTime.Unix(), ac.ID)
	if err != nil {
		return errors.Wrapf(err, "UpdateLastCrawlerTime %d",
			ac.ID)
	}
	return nil
}

func (ac *Account) MarkAsDeleted() {
	if ac.ID == 0 {
		return
	}
	_, err := mysql.GetConstConn().Exec(`UPDATE accounts SET is_deleted='Y',updated_at=? WHERE id=? LIMIT 1`,
		time.Now(), ac.ID)
	if err != nil {
		log.Errorf("account-MarkAsDeleted %d error %v",
			ac.ID, err)
	}
}

const (
	ConstLoginFailFlag   = "Login Fail"
	ConstConnectFailFlag = "Connect Fail"
)

func (ac *Account) LoginFail(flag string) {
	// protect login check
	if ac.ID == 0 {
		return
	}
	ac.ErrCount++
	_, err := mysql.GetConstConn().Exec(`UPDATE accounts SET msg=?, errcount=? WHERE id=? LIMIT 1`,
		flag, ac.ErrCount, ac.ID)
	if err != nil {
		log.Errorf("account-LoginFail %d error %v",
			ac.ID, err)
	}
}

func (ac *Account) LoginSuccess() {
	// protect login check
	if ac.ID == 0 {
		return
	}
	// protect useless update
	if ac.ErrCount == 0 {
		return
	}
	ac.ErrCount = 0
	_, err := mysql.GetConstConn().Exec(`UPDATE accounts SET msg=?, errcount=?, is_deleted='N' WHERE id=? LIMIT 1`,
		"", ac.ErrCount, ac.ID)
	if err != nil {
		log.Errorf("account-LoginSuccess %d error %v",
			ac.ID, err)
	}
}

func GetLoginFailAccount() ([]*Account, error) {
	all, err := GetCanUsedAll()
	if err != nil {
		return nil, err
	}
	loginFailAccount := []*Account{}
	for _, one := range all {
		if one.ErrCount <= constMaxErrCount {
			continue
		}
		loginFailAccount = append(loginFailAccount, one)
	}
	defer metricsFailAccounts(len(loginFailAccount))

	return loginFailAccount, nil
}
