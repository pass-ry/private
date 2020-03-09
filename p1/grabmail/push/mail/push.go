package mail

import (
	"context"
	"encoding/base64"
	"fmt"
	"grabmail/models/account"
	"grabmail/register"
	"html"
	"runtime"
	"sync"
	"time"

	"github.com/pkg/errors"
	"gitlab.ifchange.com/data/cordwood/crypto/des3"
	"gitlab.ifchange.com/data/cordwood/dfs"
	"gitlab.ifchange.com/data/cordwood/encoding/json"
	"gitlab.ifchange.com/data/cordwood/log"
	"gitlab.ifchange.com/data/cordwood/mysql"
	"gitlab.ifchange.com/data/cordwood/redis"
)

func (p *Push) constRedisQueue() string {
	return "grabmail_push_parser"
}

const (
	constMaxQueueSize = 2000
)

type Push struct {
	Config
}

type Config struct {
	CTX            context.Context
	NAME           string
	Register       register.Register
	AccountFilter  func(ac *account.Account) bool
	InboxFilter    func(siteID int) bool
	Status         int
	WorkerDuration time.Duration
}

func (cfg *Config) check() {
	if cfg == nil {
		panic("grabmail/push/mail config check error nil cfg")
	}
	if cfg.CTX == nil {
		panic("grabmail/push/mail config check error nil cfg.CTX")
	}
	if cfg.Register == nil {
		panic("grabmail/push/mail config check error nil cfg.Register")
	}
	if cfg.AccountFilter == nil {
		panic("grabmail/push/mail config check error nil cfg.AccountFilter")
	}
	if cfg.WorkerDuration <= 0 {
		panic(fmt.Sprintf("grabmail/push/mail config check error cfg.WorkerDuration not allowed %d",
			cfg.WorkerDuration))
	}
}

func New(cfg *Config) *Push {
	cfg.check()
	return &Push{
		Config: *cfg,
	}
}

func (p *Push) SyncRun() {
	for {
		log.Infof("grabmail/push/mail %s Activating", p.NAME)
		p.acRun()
		log.Infof("grabmail/push/mail %s Hiddening", p.NAME)
		select {
		case <-p.CTX.Done():
			return
		case <-time.After(p.WorkerDuration):
		}
	}
}

func (p *Push) acRun() {
	defer func() {
		err := recover()
		if err == nil {
			return
		}
		buf := make([]byte, 1<<10)
		num := runtime.Stack(buf, false)
		log.Errorf("grabmail/push/mail PANIC %v %v %v",
			err, num, string(buf))
	}()

	conn, err := redis.GetConstClient()
	if err != nil {
		log.Errorf("grabmail/push/mail GetConstClient error %v", err)
		return
	}
	defer conn.Close()

	allAccount, err := account.GetCanUsedAll()
	if err != nil {
		log.Errorf("grabmail/push/mail account.GetAll error %v",
			err)
		return
	}

	for _, ac := range allAccount {
		if p.AccountFilter(ac) == false {
			continue
		}
		select {
		case <-p.CTX.Done():
			return
		default:
		}

		size, err := conn.DoInt("LLEN", p.constRedisQueue())
		if err != nil {
			log.Errorf("grabmail/push/mail Redis LLEN error %v",
				err)
			return
		}
		if size >= constMaxQueueSize {
			log.Warnf("grabmail/push/mail redis queue is out of size %d",
				size)
			return
		}

		log.Infof("grabmail/push/mail %s %s start push",
			ac.UserName, p.constRedisQueue())

		p.inboxRun(ac, conn, constMaxQueueSize-size)

		log.Infof("grabmail/push/mail %s %s stop push",
			ac.UserName, p.constRedisQueue())
	}
}

type params struct {
	AccountID int    `json:"account_id"`
	UserName  string `json:"username"`
	Password  string `json:"password"`
	MailTable string `json:"mail_table"`
	MailUUID  string `json:"mail_uuid"`
	InboxID   int    `json:"inbox_id"`

	Subject     string `json:"subject"`
	SendTime    string `json:"send_time"`
	SiteID      int    `json:"site_id"`
	contentJson string
	Content     map[string]interface{} `json:"content"`
	attachJson  string
	Attach      map[string]interface{} `json:"attach"`

	UniqueID string `json:"unique_id"`
}

var (
	paramsPOOL = sync.Pool{New: func() interface{} { return new(params) }}

	nilParams = params{}
)

func newParams() *params {
	return paramsPOOL.Get().(*params)
}

func (params *params) close() {
	if params == nil {
		return
	}
	*params = nilParams
	paramsPOOL.Put(params)
}

func (params *params) String() string {
	j, _ := json.Marshal(params)
	p := make(map[string]interface{})
	json.Unmarshal(j, &p)
	delete(p, "password")
	delete(p, "content")
	delete(p, "attach")
	r, _ := json.Marshal(p, json.UnEscapeHTML())
	return string(r)
}

func (p *Push) inboxRun(ac *account.Account, conn redis.Client, num int) {
	defer ac.Close()

	if !p.Register.LogIn(p.NAME, ac.UserName) {
		log.Infof("grabmail/push/mail %s Register Failed",
			ac.UserName)
		return
	}
	// defer p.Register.LogOut(p.NAME, ac.UserName)

	sql := fmt.Sprintf(`SELECT id,uid,site_id,
	subject,send_time,content_dfs,attach_dfs FROM %s WHERE
	account_id=? AND site_id != 0 AND status = ? AND is_deleted='N'
	ORDER BY send_time DESC`,
		ac.MailTable())
	rows, err := mysql.GetConstConn().QueryContext(p.CTX, sql, ac.ID, p.Status)
	if err != nil {
		log.Errorf("grabmail/push/mail %s-SQL-SELECT error %v",
			ac.MailTable(), err)
		return
	}
	all := []*params{}
	for rows.Next() {
		var (
			id     int
			uid    string
			siteID int

			subject     string
			sendTime    string
			contentJson string
			attachJson  string
		)
		err := rows.Scan(&id, &uid, &siteID, &subject, &sendTime, &contentJson, &attachJson)
		if err != nil {
			log.Warnf("grabmail/push/mail %s-SQL-SELECT-SCAN error %s %v",
				ac.MailTable(), ac.UserName, err)
			continue
		}
		if p.InboxFilter(siteID) == false {
			continue
		}

		showPassword, err := des3.Decrypt(ac.Password)
		if err != nil {
			log.Errorf("password Decrypt %v", err)
			continue
		}

		paramsPayload := newParams()
		paramsPayload.AccountID = ac.ID
		paramsPayload.UserName = ac.UserName
		paramsPayload.Password = showPassword
		paramsPayload.MailTable = ac.MailTable()
		paramsPayload.MailUUID = uid
		paramsPayload.InboxID = id
		paramsPayload.Subject = subject
		paramsPayload.SendTime = sendTime
		paramsPayload.SiteID = siteID
		paramsPayload.contentJson = contentJson
		paramsPayload.attachJson = attachJson
		paramsPayload.UniqueID = fmt.Sprintf("%d_%s", ac.ID, uid)

		all = append(all, paramsPayload)
	}
	rows.Close()

	log.Infof("grabmail/push/mail %s len:%d",
		ac.UserName, len(all))

	for _, one := range all {
		select {
		case <-p.CTX.Done():
			return
		default:
		}
		if num <= 0 {
			break
		}

		one.Content = dfsDownload(one.contentJson)
		if hInterface, ok := one.Content["html"]; ok {
			if h, ok := hInterface.(string); ok {
				one.Content["html"] = html.UnescapeString(h)
			}
		}
		one.Attach = dfsDownload(one.attachJson)
		err := p.syncPushGo(p.CTX, conn, one)
		if err != nil {
			log.Errorf("PUSH %+v error %v",
				one, err)
			continue
		}
		num--
	}
}

func (p *Push) syncPushGo(ctx context.Context, conn redis.Client, one *params) error {
	defer one.close()

	message, err := json.Marshal(one, json.UnEscapeHTML())
	if err != nil {
		return errors.Wrapf(err, "Marshal push message %+v", one)
	}

	query := fmt.Sprintf("UPDATE %s SET status=11 WHERE id=? AND account_id=? AND status=? LIMIT 1",
		one.MailTable)
	result, err := mysql.GetConstConn().ExecContext(ctx, query, one.InboxID, one.AccountID, p.Status)
	if err != nil {
		return errors.Wrap(err, "UPDATE inboxes status")
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "UPDATE inboxes status RowsAffected")
	}
	if affected != 1 {
		return errors.New("UPDATE inboxes status error: UPDATE FAILED")
	}
	if _, err := conn.Do("LPUSH", p.constRedisQueue(), message); err != nil {
		return errors.Wrapf(err, "Lagou Screenshot LPUSH %+v", one)
	}
	log.Infof("grabmail/push/mail Push %+v success",
		one)

	metricsPushTotal()

	return nil
}

func dfsDownload(dfsJson string) map[string]interface{} {
	if len(dfsJson) == 0 {
		return nil
	}
	temp := make(map[string]interface{})
	err := json.Unmarshal([]byte(dfsJson), &temp)
	if err != nil {
		log.Errorf("grabmail/push/mail PushGO dfs download %v %s",
			err, dfsJson)
		return nil
	}
	result := make(map[string]interface{})
	for k, v := range temp {
		maybeDfs, ok := v.(map[string]interface{})
		if !ok {
			result[k] = v
			continue
		}
		maybeDfsJson, _ := json.Marshal(maybeDfs, json.UnEscapeHTML())
		isDfs, err := dfs.NewDfsReader(maybeDfsJson)
		if err != nil {
			result[k] = v
			continue
		}
		value, err := isDfs.Read()
		if err != nil {
			log.Errorf("grabmail/push/mail PushGO dfs-dfs download %v %s %+v",
				err, string(maybeDfsJson), v)
			continue
		}
		result[k] = base64.StdEncoding.EncodeToString([]byte(value))
	}
	return result
}
