package parse

import (
	"fmt"
	"grabmail/contact"
	"grabmail/models/account"
	"grabmail/models/inbox"
	stdHTML "html"
	"io"
	"net/mail"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"
	"gitlab.ifchange.com/data/cordwood/dfs"
	"gitlab.ifchange.com/data/cordwood/encoding/json"
)

// Parse - parse mail body by mime
// 该函数当前含有错误返回
// 发生错误时，调用者会跳过该邮件并打印错误信息
// 当邮箱中存在非法mime邮件时，会导致该邮件被多次加载
// 功能趋于稳定后，可考虑屏蔽该函数的错误
func Parse(ac *account.Account, inboxName string, uuid string,
	body io.Reader) (*inbox.Inbox, func(accountID int, inboxID int64) error, error) {

	i := inbox.NewInbox()
	i.AccountID = ac.ID
	i.UID = uuid
	i.InboxName = inboxName

	env, ok := readEnvelope(i, body)
	if !ok {
		return i, nil, nil
	}
	if env == nil {
		i.Msg = inbox.ConstNilBodyFunc()
		return i, nil, nil
	}
	i.Subject = env.GetHeader("Subject")
	sendTime := parseSendTime(env.GetHeader("Date"))
	i.SendTime = sendTime.Format("2006-01-02 15:04:05")
	if sendTime.After(time.Time{}) && sendTime.Before(ac.LastReceiveTime) {
		i.Msg = inbox.ConstBeforeReceiveTimeFunc()
		return i, nil, nil
	}
	from, err := mail.ParseAddress(env.GetHeader("From"))
	if err != nil {
		i.Msg = inbox.ConstUnknownMailFromFunc(env.GetHeader("From"))
		return i, nil, nil
	}
	i.SiteID = getSiteID(i.Subject, from.Address)
	if i.SiteID == 0 {
		// skip unknown mail, just save
		return i, nil, nil
	}
	html := env.HTML
	if !strings.Contains(html, "</html>") {
		html = fmt.Sprintf(`<html><body>%s</body></html>`, html)
	}
	html = strings.Replace(html, "</body>",
		`<div id="grab_email_sync">x</div></body>`, -1)
	var callback func(accountID int, inboxID int64) error
	var contentDfs = make(map[string]interface{})
	switch i.SiteID {
	case 2:
		html = strings.Replace(html, "charset=gb2312", "charset=utf8", 1)
		// upload all attachments for user-avatar
		attachDfs := make(map[string]*dfs.Dfs)
		for _, attach := range env.Attachments {
			// ext := strings.ToLower(path.Ext(attach.FileName))
			// if ext != ".docx" && ext != ".pdf" && ext != ".doc" && ext != ".html" &&
			// 	ext != ".htm" && ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
			// 	continue
			// }
			dfsIns, err := dfs.NewDfsWriter(attach.Content)
			if err != nil {
				return nil, nil, errors.Wrap(err, "attach DFS create")
			}
			if _, err = dfsIns.Write(); err != nil {
				return nil, nil, errors.Wrap(err, "attach DFS write")
			}
			attachDfs[attach.FileName] = dfsIns
		}
		if len(attachDfs) == 0 {
			break
		}
		contentDfs, err := json.Marshal(attachDfs, json.UnEscapeHTML())
		if err != nil {
			return nil, nil, errors.Wrap(err, "attach DFS json marshal")
		}
		i.AttachDfs = string(contentDfs)
	case 3:
		regInside := regexp.MustCompile(`href="(.*?)".*?>联系目标人选</a>`)
		matchInsideUrl := regInside.FindStringSubmatch(html)
		regOutside := regexp.MustCompile(`简历编号：(.*?)&nbsp;&nbsp;最近登录`)
		matchOutsideUrl := regOutside.FindStringSubmatch(html)
		if len(matchInsideUrl) >= 2 && len(matchOutsideUrl) >= 2 {
			i.Status = 10
			contactInsideURL := matchInsideUrl[1]
			contactInsideURL = strings.Replace(contactInsideURL, "\u0026", `&`, -1)
			contactInsideURL = strings.Replace(contactInsideURL, `\u0026`, `&`, -1)
			contactInsideURL = stdHTML.UnescapeString(contactInsideURL)
			if strings.Contains(contactInsideURL, "fireeye.com") {
				splitURL := strings.SplitAfter(contactInsideURL, "&u=")
				if len(splitURL) == 2 {
					contactInsideURL = splitURL[1]
				}
			}
			contentDfs["inside_contact_url"] = contactInsideURL
			callback = func(accountID int, inboxID int64) error { return contact.Liepin(accountID, inboxID, contactInsideURL) }

			contactOutsideURL := "https://lpt.liepin.cn/cvview/showresumedetail?resIdEncode=" + matchOutsideUrl[1]
			contentDfs["contact_url"] = contactOutsideURL
		} else {
			i.Status = 1
			i.Msg = "grabmail liepin no match contact url"
		}
	case 4, 27, 33, 34, 38, 52:
		attachDfs := make(map[string]*dfs.Dfs)
		for _, attach := range env.Attachments {
			ext := strings.ToLower(path.Ext(attach.FileName))
			if ext != ".docx" && ext != ".pdf" && ext != ".doc" && ext != ".html" &&
				ext != ".htm" && ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
				continue
			}
			dfsIns, err := dfs.NewDfsWriter(attach.Content)
			if err != nil {
				return nil, nil, errors.Wrap(err, "attach DFS create")
			}
			if _, err = dfsIns.Write(); err != nil {
				return nil, nil, errors.Wrap(err, "attach DFS write")
			}
			attachDfs[attach.FileName] = dfsIns
		}
		if len(attachDfs) == 0 {
			break
		}
		attachDfsBytes, err := json.Marshal(attachDfs, json.UnEscapeHTML())
		if err != nil {
			return nil, nil, errors.Wrap(err, "attach DFS json marshal")
		}
		i.AttachDfs = string(attachDfsBytes)
	}
	dfsIns, err := dfs.NewDfsWriter([]byte(stdHTML.UnescapeString(html)))
	if err != nil {
		return nil, nil, errors.Wrap(err, "content DFS create")
	}
	if _, err = dfsIns.Write(); err != nil {
		return nil, nil, errors.Wrap(err, "content DFS write")
	}
	contentDfs["html"] = dfsIns
	contentDfsBytes, err := json.Marshal(contentDfs, json.UnEscapeHTML())
	if err != nil {
		return nil, nil, errors.Wrap(err, "content DFS json marshal")
	}
	i.ContentDfs = string(contentDfsBytes)
	return i, callback, nil
}

func parseSendTime(mayBeDate string) time.Time {
	var formats = []string{
		"2 Jan 2006 15:04:05 +0000",
		"2 Jan 2006 15:04:05 +0000 (UTC)",
		"2 Jan 2006 15:04:05 -0700",
		"2 Jan 2006 15:04:05 -0700 (CST)",
		"2 Jan 2006 15:04:05 +0800 (GMT+08:00)",

		"02 Jan 2006 15:04:05 +0000",
		"02 Jan 2006 15:04:05 +0000 (UTC)",
		"02 Jan 2006 15:04:05 -0700",
		"02 Jan 2006 15:04:05 -0700 (CST)",
		"02 Jan 2006 15:04:05 +0800 (GMT+08:00)",

		"Mon, 2 Jan 2006 15:04:05 +0000",
		"Mon, 2 Jan 2006 15:04:05 +0000 (UTC)",
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"Mon, 2 Jan 2006 15:04:05 -0700 (CST)",
		"Mon, 2 Jan 2006 15:04:05 +0800 (GMT+08:00)",

		"Mon, 02 Jan 2006 15:04:05 +0000",
		"Mon, 02 Jan 2006 15:04:05 +0000 (UTC)",
		"Mon, 02 Jan 2006 15:04:05 -0700",
		"Mon, 02 Jan 2006 15:04:05 -0700 (CST)",
		"Mon, 02 Jan 2006 15:04:05 +0800 (GMT+08:00)",
	}
	for _, format := range formats {
		t2, err := time.Parse(format, mayBeDate)
		if err != nil {
			continue
		}
		return t2
	}
	return time.Time{}
}

func getSiteID(subject string, from string) (siteID int) {
	subject = strings.ToLower(subject)
	switch {
	case strings.Contains(subject, "转发") || strings.Contains(subject, "回复") || strings.Contains(subject, "答复"):
		siteID = 0
		return
	// case strings.Contains(subject, "zhaopin.com"):
	// 	siteID = 1
	// 	return
	case strings.Contains(subject, "51job.com"):
		siteID = 2
		return
	case (strings.Contains(subject, "来自猎聘网的候选人") || strings.Contains(subject, "来自猎聘的候选人")) && strings.Contains(from, "lietou"):
		siteID = 3
		return
	case strings.Contains(from, "dajie.com"):
		siteID = 4
		return
	case strings.Contains(from, "cjol.com"):
		siteID = 7
		return
	// case strings.Contains(from, "lagou.com") || strings.Contains(from, "lagoujobs.com"):
	// 	siteID = 11
	// 	return
	case strings.Contains(subject, "58.com") && strings.Contains(from, "58.com"):
		siteID = 12
		return
	case strings.Contains(from, "linkedin.com"):
		siteID = 20
		return
	case strings.Contains(from, "goodjobs.cn"):
		siteID = 22
		return
	case strings.Contains(from, "kanzhun.com"):
		if strings.Contains(subject, "【Boss直聘】") {
			siteID = 33
			return
		}
		siteID = 27
		return
	case strings.Contains(from, "kshr"):
		siteID = 32
		return
	case strings.Contains(from, "bosszhipin") || strings.Contains(from, "zhipin.com"):
		siteID = 33
		return
	case strings.Contains(from, "buildhr.com"):
		siteID = 34
		return
	case strings.Contains(subject, "【脉脉招聘】"):
		siteID = 38
		return
	case strings.Contains(from, "jobcn.com"):
		siteID = 44
		return
	case strings.Contains(from, "shixiseng.com"):
		siteID = 45
		return
	case strings.Contains(from, "cqjob.com") || strings.Contains(from, "huibo.com"):
		siteID = 46
		return
	case strings.Contains(from, "xumurc.com"):
		siteID = 49
		return
	case strings.Contains(from, "doctorjob.com.cn"):
		siteID = 52
		return
	case strings.Contains(from, "bjxjob.com"):
		siteID = 53
		return
	case strings.Contains(from, "xmrc.com.cn"):
		siteID = 60
		return
	case strings.Contains(from, "mail.dxy.cn"):
		siteID = 10003
		return
	}
	return
}
