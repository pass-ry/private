package mail

import (
	"context"
	"fmt"
	"grabmail/models/account"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"gitlab.ifchange.com/data/cordwood/mysql"
)

type IndexMail struct {
	Inbox string
	UUID  string

	// IMAP
	UID uint32
	// POP3
	NumberID int
	// EXCHANGE
	GraphID string
}

var (
	indexMailPOOL = sync.Pool{New: func() interface{} { return new(IndexMail) }}
	nilIndexMail  = IndexMail{}
)

func NewIndexMail() *IndexMail {
	return indexMailPOOL.Get().(*IndexMail)
}

func (i *IndexMail) Close() {
	if i == nil {
		return
	}
	*i = nilIndexMail
	indexMailPOOL.Put(i)
}

func IndexFilterExistMail(ctx context.Context, ac *account.Account, allIndexMail map[string]*IndexMail) (notExistMail []*IndexMail, err error) {
	groupByInbox := make(map[string][]*IndexMail)
	for _, one := range allIndexMail {
		groupByInbox[one.Inbox] = append(groupByInbox[one.Inbox], one)
	}

	for inbox, all := range groupByInbox {
		for len(all) > 0 {
			count := len(all)
			if count > 1000 {
				count = 1000
			}
			groupAll := all[:count]
			all = all[count:]

			allIndexMail := make(map[string][]*IndexMail, count)
			uid := make([]interface{}, len(groupAll))
			for i, one := range groupAll {
				allIndexMail[one.UUID] = append(allIndexMail[one.UUID], one)
				uid[i] = one.UUID
			}

			sql := fmt.Sprintf(`SELECT uid FROM %s WHERE
			is_deleted='N' AND account_id=%d AND
			inbox_name='%s' AND uid IN (?%s)`,
				ac.MailTable(), ac.ID, inbox, strings.Repeat(",?", len(uid)-1))

			rows, err := mysql.GetConstConn().QueryContext(ctx, sql,
				uid...)
			if err != nil {
				return nil, errors.Wrap(err, "SQL SELECT")
			}
			for rows.Next() {
				var uuid string
				if err := rows.Scan(&uuid); err != nil {
					return nil, errors.Wrap(err, "SQL SCAN")
				}
				for _, i := range allIndexMail[uuid] {
					i.Close()
				}
				delete(allIndexMail, uuid)
			}
			rows.Close()
			for _, indexMail := range allIndexMail {
				notExistMail = append(notExistMail, indexMail...)
			}
		}
	}
	return
}
