package controller

import (
	"database/sql"
	"grabmail/models/account"
	"strconv"
	"strings"
	"time"

	"gitlab.ifchange.com/data/cordwood/mysql"
	handler "gitlab.ifchange.com/data/cordwood/rpc/rpc-handler"
)

type tobParams struct {
}

func ToB(req *handler.Request, rsp *handler.Response) error {
	switch req.GetM() {
	case "get_resume_list":
		return tobGetResumeList(req, rsp)
	default:
		return handler.WrapError(nil, -1, "M Not Allowed")
	}
}

type (
	tobGetResumeListReq struct {
		Username          string `json:"username"`
		Password          string `json:"password"`
		SiteID            int    `json:"site_id"`
		Type              string `json:"type"`
		StartTimeDuration string `json:"receive_time"`
		EndTimeDuration   string `json:"receive_end_time"`
		Offset            int    `json:"start"`
		SizeLimit         int    `json:"limit"`
	}

	tobGetResumeListRsp struct {
		Count     int                          `json:"count"`
		FailCount int                          `json:"fail_count"`
		Resumes   []*tobGetResumeListRspResume `json:"resumes"`
	}

	tobGetResumeListRspResume struct {
		ID          string `json:"id"`
		Username    string `json:"username"`
		Password    string `json:"password"`
		AccountID   string `json:"account_id"`
		SiteID      int    `json:"site_id"`
		CVID        string `json:"cv_id"`
		InboxName   string `json:"inbox_name"`
		Subject     string `json:"subject"`
		JDID        string `json:"jd_id"`
		ReceiveTime string `json:"receive_time"`
		CVName      string `json:"cv_name"`
		JDName      string `json:"jd_name"`
		Status      string `json:"status"`
		Msg         string `json:"msg"`
		ContentDFS  string `json:"content_dfs"`
		ICDCID      string `json:"icdc_id"`
		IsDeleted   string `json:"is_deleted"`
		UpdatedAt   string `json:"updated_at"`
		CreatedAt   string `json:"created_at"`
	}
)

func tobGetResumeList(req *handler.Request, rsp *handler.Response) error {
	p := new(tobGetResumeListReq)

	if err := req.Unmarshal(p); err != nil {
		return err
	}
	if p.SiteID == 0 {
		return handler.WrapError(nil, 1, "unknown siteID == 0")
	}

	if p.SizeLimit == 0 {
		p.SizeLimit = 50
	}

	if len(p.StartTimeDuration) == 0 {
		p.StartTimeDuration = time.Now().Format("2006-01-02")
	}

	if len(p.EndTimeDuration) == 0 {
		p.EndTimeDuration = time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	}

	ac, err := account.GetCanUsedAccountByUsername(strings.ToLower(p.Username))
	if err == sql.ErrNoRows {
		return handler.WrapError(nil, 2, "email account not exist")
	}
	if err != nil {
		return handler.WrapError(err, 3, "find email account crash")
	}
	defer ac.Close()

	rows, err := mysql.GetConstConn().Query(`SELECT id,uid,
	inbox_name,subject,send_time,
	status,msg,icdc_id,
	content_dfs,is_deleted,
	updated_at,created_at FROM `+
		ac.MailTable()+
		` WHERE account_id=? AND site_id=? 
		AND send_time>? AND send_time<? 
		ORDER BY send_time DESC LIMIT ?,?`, ac.ID, p.SiteID,
		p.StartTimeDuration, p.EndTimeDuration,
		p.Offset, p.SizeLimit)
	if err != nil {
		return handler.WrapError(err, 4, "find resume select crash")
	}
	defer rows.Close()

	result := new(tobGetResumeListRsp)

	for rows.Next() {
		var (
			id         int
			uid        string
			inboxName  string
			subject    string
			sendTime   string
			status     int
			msg        string
			icdcID     string
			contentDFS string
			isDeleted  string
			updatedAt  string
			createdAt  string
		)
		err := rows.Scan(&id, &uid,
			&inboxName, &subject, &sendTime,
			&status, &msg, &icdcID,
			&contentDFS, &isDeleted,
			&updatedAt, &createdAt)
		if err != nil {
			return handler.WrapError(err, 5, "find resume scan crash")
		}

		resume := &tobGetResumeListRspResume{
			ID:          strconv.Itoa(id),
			Username:    p.Username,
			Password:    p.Password,
			AccountID:   strconv.Itoa(ac.ID),
			SiteID:      p.SiteID,
			CVID:        uid,
			InboxName:   inboxName,
			Subject:     subject,
			JDID:        "",
			ReceiveTime: sendTime,
			CVName:      "",
			JDName:      "",
			Status:      strconv.Itoa(status),
			Msg:         msg,
			ContentDFS:  contentDFS,
			ICDCID:      icdcID,
			IsDeleted:   isDeleted,
			UpdatedAt:   updatedAt,
			CreatedAt:   createdAt,
		}
		result.Resumes = append(result.Resumes, resume)

		result.Count++
		if len(resume.Msg) > 0 {
			result.FailCount++
		}
	}

	rsp.SetResults(result)
	return nil
}
