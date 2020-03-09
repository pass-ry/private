package controller

import (
	"context"
	"database/sql"
	"grabmail/grab/mail"
	"grabmail/models/account"
	"time"

	"github.com/beiping96/grace"
	"gitlab.ifchange.com/data/cordwood/log"
	"gitlab.ifchange.com/data/cordwood/mysql"
	handler "gitlab.ifchange.com/data/cordwood/rpc/rpc-handler"
)

func MailExist(req *handler.Request, rsp *handler.Response) error {
	params := struct {
		UserName    string `json:"username"`
		CheckExists []struct {
			UID       string `json:"uid"`
			InboxName string `json:"inbox_name"`
		} `json:"check_exists"`
	}{}
	if err := req.Unmarshal(&params); err != nil {
		return err
	}
	reply := struct {
		UserName  string   `json:"username"`
		NotExists []string `json:"not_exists"`
	}{
		UserName:  params.UserName,
		NotExists: []string{},
	}
	rsp.SetResults(&reply)

	if len(params.CheckExists) == 0 {
		return nil
	}

	ac := account.NewAC()
	err := mysql.GetConstConn().QueryRow(`SELECT
		id,username,user,password,type,
		mail_server,port,accounts.ssl,
		last_receive_time
		FROM accounts
		WHERE username=? ORDER BY id DESC LIMIT 1`,
		params.UserName).Scan(&ac.ID,
		&ac.UserName, &ac.User, &ac.Password, &ac.Type,
		&ac.MailServer, &ac.Port, &ac.Ssl, &ac.LastReceiveTime)
	if err == sql.ErrNoRows {
		log.Warnf("/mail_exist username:%s not exist, maybe is bind message.",
			params.UserName)
		return nil
	}

	ctx, cancel := context.WithTimeout(grace.CTX, time.Minute)
	defer cancel()

	allIndex := make(map[string]*mail.IndexMail)

	for _, p := range params.CheckExists {
		allIndex[p.UID] = mail.NewIndexMail()
		allIndex[p.UID].UUID = p.UID
		allIndex[p.UID].GraphID = p.UID
		allIndex[p.UID].Inbox = p.InboxName
	}

	notExistIndex, err := mail.IndexFilterExistMail(ctx, ac, allIndex)
	if err != nil {
		rsp.SetErrNo(-1)
		rsp.SetErrMsg(err.Error())
		return nil
	}

	for _, index := range notExistIndex {
		reply.NotExists = append(reply.NotExists, index.UUID)
		index.Close()
	}

	return nil
}
