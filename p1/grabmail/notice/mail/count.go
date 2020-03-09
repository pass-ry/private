package mail

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"gitlab.ifchange.com/data/cordwood/log"
	"gitlab.ifchange.com/data/cordwood/mysql"
)

func count() string {
	now := time.Now()
	today0 := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	from, to := today0, now
	if to.Sub(from) < time.Hour {
		return ""
	}

	tables := make(map[string]*table)
	accountTable, err := countAccount(mysql.GetConstConn(), from, to)
	if err != nil {
		log.Errorf("grabmail/admin countAccount error %v", err)
	} else {
		tables["解绑邮箱"] = accountTable
	}

	successTable, _, internalFailTable, err := countInbox(mysql.GetConstConn(), from, to)
	if err != nil {
		log.Errorf("grabmail/admin countInbox error %v", err)
	} else {
		tables["成功邮件统计"] = successTable
		// tables["失败邮件统计"] = failTable
		tables["内部邮件统计"] = internalFailTable
	}

	if len(tables["内部邮件统计"].Rows) == 0 && len(tables["解绑邮箱"].Rows) == 0 {
		return ""
	}

	return style(from, to, tables)
}

type table struct {
	Columns []string
	Rows    [][]string
}

func countAccount(conn *sql.DB, from, to time.Time) (*table, error) {
	rows, err := conn.Query(`SELECT id,username,msg,errcount,updated_at
	FROM accounts WHERE is_deleted='Y' AND updated_at>?`, from)
	if err != nil {
		return nil, errors.Wrap(err, "SQL-QUERY")
	}
	defer rows.Close()
	t := new(table)
	t.Columns = []string{
		"ID",
		"邮箱",
		"失败原因",
		"失败次数",
		"标记删除时间",
	}
	for rows.Next() {
		var (
			id        int
			username  string
			msg       string
			errCount  int
			updatedAt time.Time
		)
		err := rows.Scan(&id, &username, &msg, &errCount, &updatedAt)
		if err != nil {
			return nil, errors.Wrap(err, "SQL-SCAN")
		}
		t.Rows = append(t.Rows, []string{
			fmt.Sprintf("%d", id),
			username,
			msg,
			fmt.Sprintf("%d", errCount),
			updatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	if rows.Err() != nil {
		return nil, errors.Wrap(rows.Err(), "SQL-ROWS")
	}
	return t, nil
}

func countInbox(conn *sql.DB, from, to time.Time) (successTable, failTable, internalFailTable *table, err error) {
	successCount := make(map[int]int)
	failCount := make(map[int]map[int]map[string]int)
	internalFailCount := make(map[int]map[int]map[string]int)
	for i := 0; i < 64; i++ {
		if err := func() error {
			inbox := fmt.Sprintf("inboxes_%d", i)
			sql := fmt.Sprintf(`SELECT site_id,%s.status,internal_err,msg,icdc_id
				FROM %s WHERE site_id>0 AND created_at>?`, inbox, inbox)
			rows, err := conn.Query(sql, from)
			if err != nil {
				return errors.Wrap(err, "SQL-QUERY")
			}
			defer rows.Close()
			for rows.Next() {
				var (
					siteID      int
					status      int
					internalErr string
					msg         string
					icdcID      int
				)
				err := rows.Scan(&siteID, &status, &internalErr, &msg, &icdcID)
				if err != nil {
					return errors.Wrap(err, "SQL-SCAN")
				}
				if status == 20 {
					successCount[siteID] += 1
					continue
				}
				switch internalErr {
				case "Y":
					site, ok := internalFailCount[siteID]
					if !ok {
						internalFailCount[siteID] = make(map[int]map[string]int)
						site = internalFailCount[siteID]
					}
					st, ok := site[status]
					if !ok {
						site[status] = make(map[string]int)
						st = site[status]
					}
					st[msg] += 1
				case "N":
					site, ok := failCount[siteID]
					if !ok {
						failCount[siteID] = make(map[int]map[string]int)
						site = failCount[siteID]
					}
					st, ok := site[status]
					if !ok {
						site[status] = make(map[string]int)
						st = site[status]
					}
					st[msg] += 1
				}
			}
			if rows.Err() != nil {
				return errors.Wrap(rows.Err(), "SQL-ROWS")
			}
			return nil
		}(); err != nil {
			return nil, nil, nil, err
		}
	}

	successTable = new(table)
	successTable.Columns = []string{
		"渠道",
		"总计",
	}
	for siteID, count := range successCount {
		successTable.Rows = append(successTable.Rows,
			[]string{
				siteName(siteID),
				fmt.Sprintf("%d", count),
			})
	}

	failTable = new(table)
	failTable.Columns = []string{
		"渠道",
		"状态",
		"错误原因",
		"总计",
	}
	for siteID, site := range failCount {
		for status, st := range site {
			for msg, num := range st {
				failTable.Rows = append(failTable.Rows,
					[]string{
						siteName(siteID),
						fmt.Sprintf("%d", status),
						msg,
						fmt.Sprintf("%d", num),
					})
			}
		}
	}

	internalFailTable = new(table)
	internalFailTable.Columns = []string{
		"渠道",
		"状态",
		"错误原因",
		"总计",
	}
	for siteID, site := range internalFailCount {
		for status, st := range site {
			for msg, num := range st {
				internalFailTable.Rows = append(internalFailTable.Rows,
					[]string{
						siteName(siteID),
						fmt.Sprintf("%d", status),
						msg,
						fmt.Sprintf("%d", num),
					})
			}
		}
	}
	return
}
