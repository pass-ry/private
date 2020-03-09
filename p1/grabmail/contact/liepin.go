package contact

import (
	"context"
	"grabmail/models/account"
	"time"

	"github.com/pkg/errors"
	"gitlab.ifchange.com/data/cordwood/encoding/json"
	"gitlab.ifchange.com/data/cordwood/log"
	"gitlab.ifchange.com/data/cordwood/mysql"
	"gitlab.ifchange.com/data/cordwood/redis"
)

const (
	pubQueue = "liepin_contact_url"
	subQueue = "liepin_contact_callback"
)

func Liepin(accountID int, inboxID int64, insideContantURL string) error {
	conn, err := redis.GetConstClient()
	if err != nil {
		return errors.Wrap(err, "Liepin redis.GetConstClient")
	}
	defer conn.Close()

	params := struct {
		URL   string     `json:"url"`
		Data  liepinDATA `json:"data"`
		Reply string     `json:"reply_queue"`
	}{
		URL: insideContantURL,
		Data: liepinDATA{
			AccountID: accountID,
			InboxID:   inboxID,
		},
		Reply: subQueue,
	}

	paramsBytes, err := json.Marshal(params, json.UnEscapeHTML())
	if err != nil {
		return errors.Wrap(err, "Liepin json.Marshal")
	}
	_, err = conn.Do("LPUSH", pubQueue, paramsBytes)
	if err != nil {
		return errors.Wrap(err, "Liepin LPUSH")
	}
	return nil
}

type liepinDATA struct {
	AccountID int   `json:"account_id"`
	InboxID   int64 `json:"inbox_id"`
}

func Run(ctx context.Context) {
	conn, err := redis.GetConstClient()
	if err != nil {
		panic(errors.Wrap(err, "Liepin Suber redis.GetConstClient"))
	}
	defer conn.Close()

	var sleep time.Duration
	log.Infof("liepin contact is started")
	for {
		select {
		case <-ctx.Done():
			log.Infof("liepin contact is stopped")
			return
		case <-time.After(sleep):
			sleep = 0
		}
		info, err := conn.DoBytes("RPOP", subQueue)
		if err == conn.ErrNil() {
			sleep = time.Second
			continue
		}
		err = handleLiepin(info)
		if err != nil {
			log.Errorf("liepin Handle %s %v",
				string(info), err)
			continue
		}
		log.Debugf("liepin Handle %s success", string(info))
	}
}

func handleLiepin(info []byte) error {
	params := struct {
		Data   liepinDATA `json:"data"`
		Phone  string     `json:"phone"`
		Email  string     `json:"email"`
		ErrMsg string     `json:"err_msg"`
	}{}
	err := json.Unmarshal(info, &params)
	if err != nil {
		return errors.Wrap(err, "json.Unmarshal")
	}
	if params.Data.AccountID <= 0 || params.Data.InboxID <= 0 {
		return errors.Errorf("Unknown args")
	}
	ac, err := account.GetAccountByID(params.Data.AccountID)
	if err != nil {
		return errors.Wrapf(err, "account.GetAccountByID(%d)", params.Data.AccountID)
	}
	contentDfsStr := ""
	err = mysql.GetConstConn().QueryRow("SELECT content_dfs FROM "+
		ac.MailTable()+
		" WHERE id=? LIMIT 1",
		params.Data.InboxID).
		Scan(&contentDfsStr)
	if err != nil {
		return errors.Wrapf(err, "QueryRow(%s.%d)",
			ac.MailTable(), params.Data.InboxID)
	}

	var (
		contentDfs = make(map[string]interface{})
		status     = 0
	)

	err = json.Unmarshal([]byte(contentDfsStr), &contentDfs)
	if err != nil {
		return errors.Wrapf(err, "json.Unmarshal %s", contentDfsStr)
	}

	status = 0
	contentDfs["phone"] = params.Phone
	contentDfs["email"] = params.Email

	contentDfsBytes, err := json.Marshal(contentDfs, json.UnEscapeHTML())
	if err != nil {
		return errors.Wrapf(err, "json.Marshal %v", contentDfs)
	}

	_, err = mysql.GetConstConn().Exec("UPDATE "+
		ac.MailTable()+
		" SET status=?, msg=?, content_dfs=? WHERE id=? LIMIT 1",
		status, params.ErrMsg, string(contentDfsBytes), params.Data.InboxID)
	if err != nil {
		return errors.Wrapf(err, "MySQL Exec Update %s.%d", ac.MailTable(), params.Data.InboxID)
	}
	return nil
}
