package result

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"gitlab.ifchange.com/data/cordwood/mysql"
)

func ConstTable(accountID int64) string {
	return fmt.Sprintf("zhilian_%d", accountID%16)
}

type Result struct {
	ID          int64
	AccountID   int64
	CV_ID       string
	ReceiveTime time.Time
	JD_ID       string
	CV_Name     string
	JD_Name     string
	Status      int
	Msg         string
	ContentDfs  string
	IsDeleted   bool
}

func (r *Result) Insert() error {
	if r.AccountID == 0 {
		return errors.New("Need AccountID")
	}
	query := fmt.Sprintf(`
	INSERT INTO %s
	(account_id,uid,position_id,receive_time,
	cv_name,jd_name,status,msg,content_dfs,
	is_deleted,updated_at) VALUES
	(?,?,?,?,?,?,?,?,?,?,?)`,
		ConstTable(r.AccountID))
	isDeleted := "N"
	if r.IsDeleted {
		isDeleted = "Y"
	}
	execResult, err := mysql.GetConstConn().Exec(query,
		r.AccountID, r.CV_ID, r.JD_ID, r.ReceiveTime,
		r.CV_Name, r.JD_Name, r.Status, r.Msg, r.ContentDfs,
		isDeleted, time.Now())
	if err != nil {
		return errors.Wrap(err, "SQL Exec Insert")
	}
	r.ID, err = execResult.LastInsertId()
	if err != nil {
		return errors.Wrap(err, "SQL Exec Insert LastInsertId")
	}
	return nil
}

func (r *Result) CheckDuplicated() (exist bool, err error) {
	if r.AccountID == 0 {
		return false, errors.New("Need AccountID")
	}
	query := fmt.Sprintf(`
	SELECT id FROM %s
	WHERE
	account_id=? AND
	uid=? AND position_id=? AND
	receive_time=? AND is_deleted='N'
	LIMIT 1`, ConstTable(r.AccountID))
	var (
		id int64
	)
	err = mysql.GetConstConn().QueryRow(query,
		r.AccountID, r.CV_ID, r.JD_ID,
		r.ReceiveTime).Scan(&id)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, errors.Wrap(err, "SQL QueryRow Scan")
	}
	return true, nil
}
