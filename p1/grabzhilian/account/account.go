package account

import (
	"database/sql"
	"fmt"
	"math/rand"
	"time"

	"github.com/pkg/errors"
	"gitlab.ifchange.com/data/cordwood/mysql"
)

type Account struct {
	ID          int64
	Username    string
	Password    string
	ReceiveTime time.Time
	Errcount    int
	Msg         string
	Cookie      string
	IsDeleted   bool
	UpdatedAt   time.Time
}

func (ac *Account) String() string {
	return fmt.Sprintf("ID:%d Username:%s Password:%d ReceiveTime:%v ErrCount:%d Msg:%s IsDeleted:%v",
		ac.ID, ac.Username, len(ac.Password), ac.ReceiveTime, ac.Errcount, ac.Msg, ac.IsDeleted)
}

func (ac *Account) Insert() error {
	isDeleted := "N"
	if ac.IsDeleted {
		isDeleted = "Y"
	}
	execResult, err := mysql.GetConstConn().Exec(`
	INSERT INTO zhilian
	(username,password,receive_time,
	errcount,msg,cookie,is_deleted,updated_at) VALUES
	(?,?,?,?,?,?,?,?)`,
		ac.Username, ac.Password, ac.ReceiveTime,
		ac.Errcount, ac.Msg, ac.Cookie, isDeleted, time.Now())
	if err != nil {
		return errors.Wrap(err, "SQL Exec Insert")
	}
	ac.ID, err = execResult.LastInsertId()
	if err != nil {
		return errors.Wrap(err, "SQL Exec Insert LastInsertId")
	}
	return nil
}

func (ac *Account) SyncC() error {
	if ac.ID == 0 {
		return errors.New("Sync Must Has ID")
	}
	isDeleted := "N"
	if ac.IsDeleted {
		isDeleted = "Y"
	}

	execResult, err := mysql.GetConstConn().Exec(`
	UPDATE zhilian SET
	username=?,password=?,receive_time=?,
	errcount=?,msg=?,cookie=?,is_deleted=?,updated_at=?
	WHERE id=? LIMIT 1
	`, ac.Username, ac.Password, ac.ReceiveTime,
		ac.Errcount, ac.Msg, ac.Cookie, isDeleted, time.Now(),
		ac.ID)
	if err != nil {
		return errors.Wrap(err, "SQL Exec UpdateC")
	}
	affectedRows, err := execResult.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "SQL Exec UpdateC RowsAffected")
	}
	if affectedRows != 1 {
		return ErrNotExist
	}
	return nil
}
func (ac *Account) Sync() error {
	if ac.ID == 0 {
		return errors.New("Sync Must Has ID")
	}
	isDeleted := "N"
	if ac.IsDeleted {
		isDeleted = "Y"
	}

	execResult, err := mysql.GetConstConn().Exec(`
	UPDATE zhilian SET
	username=?,password=?,receive_time=?,
	errcount=?,msg=?,is_deleted=?,updated_at=?
	WHERE id=? LIMIT 1
	`, ac.Username, ac.Password, ac.ReceiveTime,
		ac.Errcount, ac.Msg, isDeleted, time.Now(),
		ac.ID)
	if err != nil {
		return errors.Wrap(err, "SQL Exec Update")
	}
	affectedRows, err := execResult.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "SQL Exec Update RowsAffected")
	}
	if affectedRows != 1 {
		return ErrNotExist
	}
	return nil
}

var (
	ErrNotExist = sql.ErrNoRows
)

func GetByUsername(username string) (*Account, error) {
	var (
		ac        = new(Account)
		isDeleted string
	)
	err := mysql.GetConstConn().QueryRow(`SELECT
	id,username,password,receive_time,
	errcount,msg,is_deleted 
	FROM zhilian WHERE username=? LIMIT 1`,
		username).
		Scan(&ac.ID, &ac.Username, &ac.Password, &ac.ReceiveTime,
			&ac.Errcount, &ac.Msg, &isDeleted)
	if err == sql.ErrNoRows {
		return nil, err
	}
	if err != nil {
		return nil, errors.Wrap(err, "SQL Query Select")
	}
	if isDeleted == "Y" {
		ac.IsDeleted = true
	}
	return ac, nil
}

func GetAllUsefulAccounts() ([]*Account, error) {
	acs := []*Account{}
	rows, err := mysql.GetConstConn().Query(`SELECT
	id,username,password,receive_time,
	errcount,msg 
	FROM zhilian WHERE is_deleted='N'`)
	if err != nil {
		return nil, errors.Wrap(err, "SQL Query")
	}
	defer rows.Close()
	for rows.Next() {
		ac := new(Account)
		err := rows.Scan(&ac.ID, &ac.Username, &ac.Password, &ac.ReceiveTime,
			&ac.Errcount, &ac.Msg)
		if err != nil {
			return nil, errors.Wrap(err, "SQL Scan")
		}
		acs = append(acs, ac)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "SQL Rows")
	}
	return acs, nil
}

func GetAll() ([]*Account, error) {
	rows, err := mysql.GetConstConn().Query(`SELECT
	id,username,password,receive_time,errcount,msg,is_deleted,updated_at
	FROM zhilian`)
	if err != nil {
		return nil, fmt.Errorf("GetAll Query %v",
			err)
	}
	defer rows.Close()
	acs := []*Account{}
	for rows.Next() {
		var isDeleted string
		dbAC := new(Account)
		err := rows.Scan(&dbAC.ID,
			&dbAC.Username, &dbAC.Password,
			&dbAC.ReceiveTime,
			&dbAC.Errcount, &dbAC.Msg,
			&isDeleted, &dbAC.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("GetAll Scan %v",
				err)
		}
		if isDeleted == "Y" {
			dbAC.IsDeleted = true
		}
		acs = append(acs, dbAC)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GetAll rows.Err %v",
			err)
	}
	rand.Shuffle(len(acs), func(i, j int) {
		acs[i], acs[j] = acs[j], acs[i]
	})
	return acs, nil

}
