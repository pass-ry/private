package mail

import (
	"context"
	"fmt"
	"grabmail/grab/mail/parse"
	"grabmail/models/account"
	"io"

	"gitlab.ifchange.com/data/cordwood/log"
	"gitlab.ifchange.com/data/cordwood/mysql"
)

type InboxMail struct {
	Index  *IndexMail
	Body   io.Reader
	Status int
}

func InboxSaveMail(ctx context.Context, ac *account.Account, mails []*InboxMail) {
	sql := fmt.Sprintf(`INSERT INTO %s
	(account_id,uid,
	inbox_name,subject,send_time,
	site_id,status,msg,content_dfs,attach_dfs)
	VALUES (?,?,?,?,?,?,?,?,?,?)`, ac.MailTable())
	filterSQL := fmt.Sprintf(`SELECT id FROM %s WHERE
	account_id=? AND
	subject=? AND
	send_time=? AND
	site_id=? AND
	is_deleted='N' LIMIT 1`, ac.MailTable())
	for _, m := range mails {
		select {
		case <-ctx.Done():
			return
		default:
		}
		inbox, callback, err := parse.Parse(ac, m.Index.Inbox, m.Index.UUID, m.Body)
		if err != nil {
			log.Warnf("parse ac:%+v mail index:%+v error %v",
				ac, m.Index, err)
			inbox.Close()
			continue
		}
		if inbox == nil {
			log.Warnf("parse ac:%+v mail index:%+v error NIL inbox.Inbox",
				ac, m.Index)
			inbox.Close()
			continue
		}
		if inbox.SiteID > 0 {
			var id int
			err := mysql.GetConstConn().QueryRowContext(ctx, filterSQL,
				ac.ID, inbox.Subject, inbox.SendTime, inbox.SiteID).Scan(&id)
			switch {
			case err == nil: // find exist
				inbox.Status = 1
				inbox.Msg = fmt.Sprintf("has exist table id=%d", id)
				log.Infof("ac:%+v subject:%s has exist table id=%d", ac, inbox.Subject, id)
			case err == mysql.ErrNoRows:
			default:
				log.Errorf("inbox filter mysql exec error %v", err)
			}
		}
		execResult, err := mysql.GetConstConn().ExecContext(ctx, sql,
			inbox.AccountID, inbox.UID, inbox.InboxName,
			inbox.Subject, inbox.SendTime, inbox.SiteID,
			inbox.Status,
			inbox.Msg, inbox.ContentDfs, inbox.AttachDfs)
		if err != nil {
			log.Errorf("parse ac:%+v mail index:%+v error SQL INSERT %v",
				ac, m.Index, err)
			inbox.Close()
			continue
		}
		inboxID, err := execResult.LastInsertId()
		if err != nil {
			log.Errorf("parse ac:%+v mail index:%+v error SQL LastInsertId %v",
				ac, m.Index, err)
			inbox.Close()
			continue
		}
		if callback != nil {
			if err := callback(ac.ID, inboxID); err != nil {
				log.Errorf("parse ac:%+v mail index:%+v Callback %v",
					ac, m.Index, err)
			}
		}
		inbox.Close()
	}
	return
}
