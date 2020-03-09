package rpc

import (
	"context"
	"database/sql"
	"fmt"
	"grabzhilian/account"
	result "grabzhilian/grab-result"
	"grabzhilian/puber"
	"grabzhilian/resume"
	"strings"
	"time"

	"gitlab.ifchange.com/data/cordwood/counter"
	"gitlab.ifchange.com/data/cordwood/crypto/des3"
	"gitlab.ifchange.com/data/cordwood/log"
	"gitlab.ifchange.com/data/cordwood/mysql"
	"gitlab.ifchange.com/data/cordwood/redis"
	handler "gitlab.ifchange.com/data/cordwood/rpc/rpc-handler"
	router "gitlab.ifchange.com/data/cordwood/rpc/rpc-router"
	server "gitlab.ifchange.com/data/cordwood/rpc/rpc-server"
)

func Run(ctx context.Context, port string) {
	r := router.NewRouter().WithPPROF().WithMetrics().
		Handler("/zhilian_tob", zhilianTob).
		Handler("/zhilian_exist", zhilianExist).
		Handler("/zhilian_phone_msg", zhilianPhoneMsg).
		Handler("/zhilian_judge_account", zhilianJudgeAccount).
		Handler("/tob", resume.GetResume).
		Handler("/zhilian_check_password", zhilianCheckPassword)

	server.NewServer(ctx, port, r).GraceRun()
}

func zhilianTob(req *handler.Request, rsp *handler.Response) error {
	// 判断是否是验证是否登录
	if req.GetM() == "judge_account" {
		return zhilianJudgeAccount(req, rsp)
	}

	params := struct {
		Username    string `json:"username"`
		Password    string `json:"password,omitempty"`
		ReceiveTime string `json:"receive_time,omitempty"`
		Phone       string `json:"phone,omitempty"`
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
		return zhilianUbind(params.Username, rsp)
	default:
		counter.NewGrabAdminRPCCounter(counter.Kind_Bind, 1).Inc(true, "zhilian-bind", "unknown-bind-result")
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
		err = puber.BindPub(ac, params.Phone, params.Source, false /* account is not exist */)
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
		err = puber.BindPub(oldAc, params.Phone, params.Source, true /* account is exist */)
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

func zhilianUbind(username string, rsp *handler.Response) error {
	ac, err := account.GetByUsername(username)
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

func zhilianExist(req *handler.Request, rsp *handler.Response) error {
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

	query := fmt.Sprintf(`SELECT uid,position_id,receive_time FROM %s WHERE
	account_id=%d AND is_deleted='N' AND uid IN (%s)`,
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

func zhilianCheckPassword(req *handler.Request, rsp *handler.Response) error {
	params := struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}{}
	err := req.Unmarshal(&params)
	if err != nil {
		return err
	}
	if len(params.Username) == 0 {
		return handler.WrapError(fmt.Errorf("username is nil"),
			-1, "username不存在")
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

	ac.ReceiveTime = time.Now().Add(time.Duration(-10*24) * time.Hour)
	success, err := puber.BindPubCheckPassword(ac)
	if err != nil {
		return handler.WrapError(err,
			1, err.Error())
	}
	rsp.SetResults(success)
	return nil
}

func zhilianPhoneMsg(req *handler.Request, rsp *handler.Response) error {
	params := struct {
		Username string `json:"username"`
		Phone    string `json:"phone"`
		Msg      string `json:"msg"`
	}{}

	err := req.Unmarshal(&params)
	if err != nil {
		return err
	}

	if len(params.Username) == 0 {
		return handler.WrapError(fmt.Errorf("username is nil"), -1, "username not exist")
	}

	if len(params.Phone) == 0 {
		return handler.WrapError(fmt.Errorf("phone is nil"), -1, "phone not exist")
	}

	if len(params.Msg) == 0 {
		return handler.WrapError(fmt.Errorf("msg is nil"), -1, "msg not exist")
	}

	conn, err := redis.GetConstClient()
	if err != nil {
		log.Errorf("zhilianPhoneMsg redis.GetConstConn err:%v", err)
		return err
	}
	keyPhoneMsg := fmt.Sprintf("pyspider_%s_%s", params.Username, params.Phone)
	valueMsg := fmt.Sprintf("%s_%s", params.Phone, params.Msg)
	_, err = conn.Do("SETEX", keyPhoneMsg, 120, valueMsg)
	if err != nil {
		log.Errorf("zhilianPhoneMsg conn.Do SETEX err:%v", err)
		return err
	}

	rsp.SetResults(true)
	return nil
}

func zhilianJudgeAccount(req *handler.Request, rsp *handler.Response) error {
	params := struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Source   int    `json:"source"`
	}{}
	err := req.Unmarshal(&params)
	if err != nil {
		rsp.SetResults(false)
		log.Errorf("zhilianJudgeAccount req.Unmarshal ERROR=%v", err)
		return handler.WrapError(err, -1, "结构体不正确")
	}
	if len(params.Username) == 0 {
		rsp.SetResults(false)
		return handler.WrapError(fmt.Errorf("username is nil"),
			-1, "username不存在")
	}

	if len(params.Password) == 0 {
		rsp.SetResults(false)
		return handler.WrapError(fmt.Errorf("password is nil"),
			-1, "password不存在")
	}

	ac, err := account.GetByUsername(params.Username)
	if err == sql.ErrNoRows {
		rsp.SetResults(false)
		return handler.WrapError(fmt.Errorf("username not exist"), -1, "username not exist")
	}
	if err != nil {
		rsp.SetResults(false)
		log.Errorf("zhilianJudgeAccount account.GetByUsername ERROR=%v", err)
		return handler.WrapError(err, -1, "Get username err")
	}

	password, err := des3.Encrypt(params.Password)
	if err != nil {
		rsp.SetResults(false)
		return handler.WrapError(err, 85085000, "系统错误")
	}
	if ac.Password != password {
		rsp.SetResults(false)
		return handler.WrapError(err, -1, "账号或密码不正确")
		return nil
	}

	rsp.SetResults(!ac.IsDeleted)
	return nil
}
