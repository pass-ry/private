package deliver

import (
	"context"
	"encoding/json"
	"fmt"
	"grablagou/account"
	result "grablagou/grab-result"
	"math/rand"
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
	constDeliverQueue = "lagou_push_parser"
	dfsM              dfsStruct
	params            PureLagou
	dfsByte           []byte
)

type PureLagou struct {
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
}

type dfsStruct struct {
	City          string `json:"city"`
	Img           string `json:"img"`
	Json          string `json:"json"`
	Name          string `json:"name"`
	Pdf           string `json:"pdf"`
	Position_id   string `json:"position_id"`
	ReceiveTime   string `json:"receive_time"`
	Resume_id     string `json:"resume_id"`
	Position_name string `json:"position_name"`
}

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
	id,account_id,cv_id,jd_id,receive_time,
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
			return errors.Wrapf(err,
				"delive %v", one)
		}
	}
	return nil
}

func delive(conn redis.Client, ac *account.Account, r *result.Result) error {
	var err error

	decryptPassword, err := des3.Decrypt(ac.Password)
	if err != nil {
		return errors.Wrap(err, "password Decrypt")
	}

	err = json.Unmarshal([]byte(r.ContentDfs), &dfsM)
	if err != nil {
		return errors.Wrap(err, "delive  json.Unmarshal")
	}

	dfsByte, err = fdfs.GetConstClient().Get(dfsM.Img)
	if err != nil {
		return errors.Wrap(err, "delive fdfs.GetConstClient .Get1")
	}
	dfsM.Img = string(dfsByte)

	dfsByte, err = fdfs.GetConstClient().Get(dfsM.Json)
	if err != nil {
		return errors.Wrap(err, "delive fdfs.GetConstClient .Get2")
	}
	dfsM.Json = string(dfsByte)

	dfsByte, err = fdfs.GetConstClient().Get(dfsM.Pdf)
	if err != nil {
		return errors.Wrap(err, "delive fdfs.GetConstClient .Get3")
	}
	dfsM.Pdf = string(dfsByte)

	dfsRaw, err := json.Marshal(dfsM)
	if err != nil {
		return errors.Wrap(err, "delive json.Marshal error")
	}

	uID := fmt.Sprintf("%d_%d", ac.ID, r.ID)

	params.UniqueID = uID
	params.SiteID = 11
	params.ContentDfs = dfsRaw
	params.JDName = r.JD_Name
	params.JDID = r.JD_ID
	params.ID = int(r.ID)
	params.AccountID = int(r.AccountID)
	params.Table = result.ConstTable(ac.ID)
	params.UserName = ac.Username
	params.Password = decryptPassword
	params.ReceiveTime = r.ReceiveTime

	jsonParams, err := json.Marshal(params)
	if err != nil {
		return errors.Wrap(err, "Json Marshal")
	}

	log.Infof("lagou delive content final size=%d", len(jsonParams))

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
	log.Infof("LPUSH...."+constDeliverQueue+" table:%s, usename:%s, receive_time:%v", params.Table, params.UserName, params.ReceiveTime)
	_, err = conn.Do("LPUSH", constDeliverQueue, string(jsonParams))
	if err != nil {
		return errors.Wrap(err, "Redis Exec LPUSH")
	}

	if rand.Intn(3) == 1 {
		runtime.GC()
	}

	return nil
}
