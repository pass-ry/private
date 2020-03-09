package rpc

import (
	"context"
	"fmt"
	"grablagou/account"
	result "grablagou/grab-result"
	"grablagou/puber"
	"grablagou/resume"
	"strings"
	"time"

	"gitlab.ifchange.com/data/cordwood/counter"
	"gitlab.ifchange.com/data/cordwood/crypto/des3"
	"gitlab.ifchange.com/data/cordwood/log"
	"gitlab.ifchange.com/data/cordwood/mysql"
	handler "gitlab.ifchange.com/data/cordwood/rpc/rpc-handler"
	router "gitlab.ifchange.com/data/cordwood/rpc/rpc-router"
	server "gitlab.ifchange.com/data/cordwood/rpc/rpc-server"
)

func Run(ctx context.Context, port string) {
	r := router.NewRouter().WithPPROF().WithMetrics().
		Handler("/lagou_bind", lagouBind).
		Handler("/lagou_ubind", lagouUbind).
		Handler("/tob", resume.GetResume).
		Handler("/lagou_exist", lagouExist)

	server.NewServer(ctx, port, r).GraceRun()
}

func lagouBind(req *handler.Request, rsp *handler.Response) error {
	params := struct {
		Username    string `json:"username"`
		Password    string `json:"password"`
		ReceiveTime string `json:"receive_time,omitempty"`
		Source      int    `json:"source"`
	}{}
	err := req.Unmarshal(&params)
	if err != nil {
		return err
	}
	if len(params.Username) == 0 {
		return handler.WrapError(fmt.Errorf("username is nil"),
			-1, "username不存在")
	}
	switch req.GetM() {
	case "ubind", "unbind":
		return lagouUbind(req, rsp)
	default:
		counter.NewGrabAdminRPCCounter(counter.Kind_Bind, 11).Inc(true, "lagou-bind", "unknown-bind-result")
	}

	if len(params.Password) == 0 {
		return handler.WrapError(fmt.Errorf("password is nil"),
			-1, "password不存在")
	}
	password, err := des3.Encrypt(params.Password)
	if err != nil {
		return handler.WrapError(err, 85085000, "系统错误")
	}

	ac := new(account.Account)
	ac.Username = params.Username
	ac.Password = password
	now := time.Now()
	ac.ReceiveTime, err = time.ParseInLocation("2006-01-02 15:04:05",
		params.ReceiveTime,
		now.Location())
	if err != nil {
		ac.ReceiveTime = now.Add(time.Duration(-10*24) * time.Hour)
	}
	oldAc, err := account.GetByUsername(ac.Username)
	if err == account.ErrNotExist {
		err = puber.BindPub(ac, params.Source, false)
		if err != nil {
			return handler.WrapError(err, -1, "系统错误")
		}
		rsp.SetResults(true)
		return nil
	}
	if err != nil {
		return handler.WrapError(err, -1, "系统错误")
	}
	if oldAc.IsDeleted {
		oldAc.Password = ac.Password
		err = puber.BindPub(oldAc, params.Source, true)
		if err != nil {
			return handler.WrapError(err, -1, "系统错误")
		}
		rsp.SetResults(true)
		return nil
	}
	rsp.SetResults(true)
	err = oldAc.Push(1, params.Source)
	if err != nil {
		return handler.WrapError(err, -1, "系统错误")
	}
	return nil
}

func lagouUbind(req *handler.Request, rsp *handler.Response) error {
	params := struct {
		Username string `json:"username"`
	}{}
	err := req.Unmarshal(&params)
	if err != nil {
		return err
	}
	if len(params.Username) == 0 {
		return handler.WrapError(fmt.Errorf("username is nil"),
			-1, "username不存在")
	}
	ac, err := account.GetByUsername(params.Username)
	if err == account.ErrNotExist {
		return handler.WrapError(err, 2005001, "Account is not exist")
	}
	ac.IsDeleted = true
	err = ac.Sync()
	if err != nil {
		return handler.WrapError(err, -1, "系统错误")
	}
	rsp.SetResults(true)
	return nil
}

func lagouExist(req *handler.Request, rsp *handler.Response) error {
	params := struct {
		UserName string `json:"username"`
		Exists   []struct {
			UID         string `json:"resume_id"`
			PositionID  string `json:"position_id"`
			ReceiveTime string `json:"receive_time"`
		} `json:"check_exists"`
	}{}
	if err := req.Unmarshal(&params); err != nil {
		return err
	}
	reply := struct {
		UserName string   `json:"username"`
		Exists   []string `json:"exists"`
	}{
		UserName: params.UserName,
	}
	rsp.SetResults(&reply)

	if len(params.Exists) == 0 {
		return nil
	}

	ac, err := account.GetByUsername(params.UserName)
	if err != nil {
		log.Errorf("GetByUserName error %v", err)
		return nil
	}

	uids := make([]string, len(params.Exists))
	index := make(map[string]string, len(params.Exists))
	receiveTimeIndex := make(map[string]string, len(params.Exists))
	for i, check := range params.Exists {
		uids[i] = fmt.Sprintf(`'%s'`, check.UID)
		index[check.UID] = check.PositionID
		receiveTimeIndex[check.UID] = check.ReceiveTime
	}

	query := fmt.Sprintf(`SELECT cv_id,jd_id,receive_time FROM %s WHERE
	account_id=%d AND is_deleted='N' AND cv_id IN (%s)`,
		result.ConstTable(ac.ID), ac.ID, strings.Join(uids, ","))

	rows, err := mysql.GetConstConn().Query(query)
	if err != nil {
		return handler.WrapError(err, -1, "SQL Error Query")
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cvID        string
			jdID        string
			receiveTime time.Time
		)
		err := rows.Scan(&cvID, &jdID, &receiveTime)
		if err != nil {
			return handler.WrapError(err, -1, "SQL Error Scan")
		}
		if jdID == index[cvID] &&
			receiveTime.Format("2006-01-02 15:04:05") == receiveTimeIndex[cvID] {
			reply.Exists = append(reply.Exists, cvID)
		}
	}
	if err := rows.Err(); err != nil {
		return handler.WrapError(err, -1, "SQL Error Rows")
	}
	return nil
}
