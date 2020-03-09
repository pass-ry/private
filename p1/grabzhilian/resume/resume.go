package resume

import (
	"fmt"
	"gitlab.ifchange.com/data/cordwood/log"
	"gitlab.ifchange.com/data/cordwood/mysql"
	handler "gitlab.ifchange.com/data/cordwood/rpc/rpc-handler"
	"time"
)

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

func GetResume(req *handler.Request, rsp *handler.Response) error {
	var params tobGetResumeListReq
	err := req.Unmarshal(&params)
	if err != nil {
		return fmt.Errorf("zhilian GetResume req.Unmarshal err=%v", err)
	}

	if len(params.Username) == 0 {
		log.Error("zhilian GetResume username is null")
		return handler.WrapError(fmt.Errorf("zhilian GetResume username is null"), -1, "username is null")
	}

	tobRes, err := getResumeRes(&params)
	if err != nil {
		return handler.WrapError(err, -1, "zhilian get resume is failed")
	}

	rsp.SetResults(tobRes)
	return nil
}

// 查询返回结果
func getResumeRes(params *tobGetResumeListReq) (*tobGetResumeListRsp, error) {
	// 获取tabName,表名 和 用户id即accounts_id
	tabName, id, err := getTableName(params.Username)
	if err != nil {
		return nil, fmt.Errorf("zhilian getResumeRes getTableName err=%v", err)
	}

	// 构造数据的默认值
	if params.SizeLimit == 0 {
		params.SizeLimit = 50
	}

	// 当查询时间为空时，默认为今日
	if len(params.StartTimeDuration) == 0 {
		params.StartTimeDuration = time.Now().Format("2006-01-02")
	}

	// 当查询结束时间为空时，默认为明天
	if len(params.EndTimeDuration) == 0 {
		params.EndTimeDuration = time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	}

	// 构造查询SQL
	sql := fmt.Sprintf("SELECT id, account_id, uid, position_id, receive_time, cv_name, jd_name, status, msg, content_dfs, icdc_id, is_deleted, updated_at, created_at FROM %s where account_id=%d and receive_time>='%s' and receive_time<='%s' limit %d, %d",
		tabName, id, params.StartTimeDuration, params.EndTimeDuration, params.Offset, params.SizeLimit)

	// 查询结果
	rows, err := mysql.GetConstConn().Query(sql)
	if err != nil {
		return nil, fmt.Errorf("zhilian getResumeRes SELECT error=%v", err)
	}

	// 构造数据格式
	var resume tobGetResumeListRsp
	var errCount int

	// scan
	for rows.Next() {
		rspResume := tobGetResumeListRspResume{}
		rows.Scan(&rspResume.ID, &rspResume.AccountID, &rspResume.CVID, &rspResume.JDID, &rspResume.ReceiveTime, &rspResume.CVName, &rspResume.JDName, &rspResume.Status, &rspResume.Msg, &rspResume.ContentDFS, &rspResume.ICDCID, &rspResume.IsDeleted, &rspResume.UpdatedAt, &rspResume.CreatedAt)

		// 开始补充数据库以外的内容
		rspResume.Username = params.Username
		rspResume.Password = params.Password
		rspResume.SiteID = params.SiteID
		resume.Resumes = append(resume.Resumes, &rspResume)

		// 判断简历状态是否是20
		if rspResume.Status != "20" && rspResume.Status != "1" {
			errCount = errCount + 1
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("zhilian getResumeRes rows.Err error=%v", err)
	}

	resume.Count = len(resume.Resumes)
	resume.FailCount = errCount

	return &resume, nil
}

// 返回查询表名 和 username的ID
func getTableName(name string) (string, int64, error) {
	var id int64
	err := mysql.GetConstConn().QueryRow(`SELECT id FROM zhilian where username=? limit 1`, name).Scan(
		&id)
	if err != nil {
		return "", 0, fmt.Errorf("zhilian resume getUserNameID err=%v", err)
	}

	return fmt.Sprintf("zhilian_%d", id%16), id, nil
}
