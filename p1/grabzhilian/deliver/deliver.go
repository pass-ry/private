package deliver

import (
	"context"
	"encoding/json"
	"fmt"
	"grabzhilian/account"
	result "grabzhilian/grab-result"
	"runtime"
	"time"

	"github.com/pkg/errors"
	"gitlab.ifchange.com/data/cordwood/crypto/des3"
	fdfs "gitlab.ifchange.com/data/cordwood/fast-dfs"
	"gitlab.ifchange.com/data/cordwood/log"
	"gitlab.ifchange.com/data/cordwood/mysql"
	"gitlab.ifchange.com/data/cordwood/redis"
)

var (
	//	constDeliverQueue = "webkit_sync_resume_push"
	constDeliverQueue = "webkit_push_parser"
)

func Run(ctx context.Context) {
	ticker := time.Tick(time.Minute)
	for {
		run(ctx)
		select {
		case <-ticker:
		case <-ctx.Done():
			log.Infof("Deliver stopped")
			return
		}
	}
}

func run(ctx context.Context) {
	acs, err := account.GetAllUsefulAccounts()
	if err != nil {
		log.Errorf("Deliver GetAllUsefulAccounts %v", err)
		return
	}

	for _, ac := range acs {
		select {
		case <-ctx.Done():
			return
		default:
		}
		err := acRun(ctx, ac)
		if err != nil {
			log.Warnf("Deliver %+v Error %v",
				ac, err)
			continue
		}
		log.Infof("Deliver %+v Success", ac)
	}
}

func acRun(ctx context.Context, ac *account.Account) error {
	defer func() {
		err := recover()
		if err == nil {
			return
		}
		buf := make([]byte, 1<<10)
		num := runtime.Stack(buf, false)
		log.Errorf("Deliver %v PANIC %v %v %v",
			ac, err, num, string(buf))
	}()

	query := fmt.Sprintf(`
	SELECT
	id,account_id,uid,position_id,receive_time,
	cv_name,jd_name,status,msg,content_dfs
	FROM %s
	WHERE account_id=? AND status=0 AND is_deleted='N'
	`, result.ConstTable(ac.ID))
	rows, err := mysql.GetConstConn().Query(query, ac.ID)
	if err != nil {
		return errors.Wrap(err, "SQL Query")
	}
	defer rows.Close()

	all := []*result.Result{}
	for rows.Next() {
		r := new(result.Result)
		err := rows.Scan(&r.ID,
			&r.AccountID, &r.CV_ID, &r.JD_ID, &r.ReceiveTime,
			&r.CV_Name, &r.JD_Name, &r.Status, &r.Msg, &r.ContentDfs)
		if err != nil {
			return errors.Wrap(err, "SQL Scan")
		}
		all = append(all, r)
	}
	if err := rows.Err(); err != nil {
		return errors.Wrap(err, "SQL Rows")
	}

	conn, err := redis.GetConstClient()
	if err != nil {
		return errors.Wrap(err, "Get Redis Const Client")
	}
	defer conn.Close()

	for _, one := range all {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		err := delive(conn, ac, one)
		if err != nil {
			log.Errorf("delive err=%v  ac=%v", err, one)
			continue
		}
	}
	return nil
}

func delive(conn redis.Client, ac *account.Account, r *result.Result) error {
	decryptPassword, err := des3.Decrypt(ac.Password)
	if err != nil {
		return errors.Wrap(err, "password Decrypt")
	}

	dfsM := make(map[string]string)
	err = json.Unmarshal([]byte(r.ContentDfs), &dfsM)
	if err != nil {
		return errors.Wrap(err, "delive  json.Unmarshal")
	}

	dfsB, err := fdfs.GetConstClient().Get(dfsM["img"])
	if err != nil {
		return errors.Wrap(err, "delive fdfs.GetConstClient .Get")
	}

	dfsJ, err := fdfs.GetConstClient().Get(dfsM["json"])
	if err != nil {
		return errors.Wrap(err, "delive fdfs.GetConstClient .Get")
	}

	dfsM["img"] = string(dfsB)
	dfsM["json"] = string(dfsJ)

	jsonDFS, err := json.Marshal(dfsM)
	if err != nil {
		return errors.Wrap(err, "delive json.Marshal error")
	}

	var dfsRaw json.RawMessage
	err = json.Unmarshal(jsonDFS, &dfsRaw)
	if err != nil {
		return errors.Wrap(err, "delive json.Unmarshal json.RawMessage error")
	}

	uID := fmt.Sprintf("%d_%d", ac.ID, r.ID)

	params := struct {
		UniqueID    string          `json:"unique_id"`
		SiteID      int             `json:"site_id"`
		ContentDfs  json.RawMessage `json:"content"`
		JDName      string          `json:"jd_name"`
		JDID        string          `json:"jd_id"`
		ID          int             `json:"inbox_id"`
		AccountID   int             `json:"account_id"`
		Table       string          `json:"table"`
		UserName    string          `json:"username"`
		Password    string          `json:"password"`
		ReceiveTime time.Time       `json:"send_time"`
	}{
		UniqueID:    uID,
		SiteID:      1,
		ContentDfs:  dfsRaw,
		JDName:      r.JD_Name,
		JDID:        r.JD_ID,
		ID:          int(r.ID),
		AccountID:   int(r.AccountID),
		Table:       result.ConstTable(ac.ID),
		UserName:    ac.Username,
		Password:    decryptPassword,
		ReceiveTime: r.ReceiveTime,
	}

	jsonParams, err := json.Marshal(params)
	if err != nil {
		return errors.Wrap(err, "Json Marshal")
	}

	query := fmt.Sprintf(`
	UPDATE %s SET status=1
	WHERE id=? AND status=0
	LIMIT 1`, result.ConstTable(ac.ID))
	execResult, err := mysql.GetConstConn().Exec(query, r.ID)
	if err != nil {
		return errors.Wrap(err, "SQL Update")
	}
	affectedRows, err := execResult.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "SQL RowsAffected")
	}
	if affectedRows != 1 {
		return errors.New("status!=0 Means this row has deliver")
	}
	_, err = conn.Do("LPUSH", constDeliverQueue, string(jsonParams))
	if err != nil {
		return errors.Wrap(err, "Redis Exec LPUSH")
	}

	return nil
}
