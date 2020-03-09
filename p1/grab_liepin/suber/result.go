package suber

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"
	dfs "gitlab.ifchange.com/data/cordwood/fast-dfs"
)

type SubResult struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Phone    string `json:"phone"`
	Cookie   string `json:"cookie"`

	AuthCodeStatus int    `json:"auth_code_status"`
	AuthCodeKey    string `json:"auth_code_key"`

	Code   int             `json:"code"`
	Msg    string          `json:"msg"`
	Resume SubResultResume `json:"resume,omitempty"`
}

func (s *SubResult) String() string {
	return fmt.Sprintf("Username: %s Password: %d Code: %d Msg: %s Resume: %v",
		s.Username, len(s.Password), s.Code, s.Msg, s.Resume)
}

type SubResultResume map[string]interface{}

func (stringer SubResultResume) String() string {
	if len(stringer) > 0 {
		cvID, _ := stringer.ResumeID()
		jdID, _ := stringer.PositionID()
		receiveTime, _ := stringer.ReceiveTime()
		return fmt.Sprintf("Has Resume cvID:%s jdID:%s receiveTime:%v",
			cvID, jdID, receiveTime)
	}
	return "No Resume"
}

func (r SubResultResume) ResumeID() (string, error) {
	Interface, ok := r["resume_id"]
	if !ok {
		return "", errors.New("resume_id Not Exist")
	}
	res, ok := Interface.(string)
	if !ok {
		return "", errors.New("resume_id Not String")
	}
	return res, nil
}

func (r SubResultResume) PositionID() (string, error) {
	Interface, ok := r["position_id"]
	if !ok {
		return "", errors.New("position_id Not Exist")
	}
	res, ok := Interface.(string)
	if !ok {
		return "", errors.New("position_id Not String")
	}
	return res, nil
}

func (r SubResultResume) ReceiveTime() (time.Time, error) {
	receiveTimeInterface, ok := r["receive_time"]
	if !ok {
		return time.Time{}, errors.New("receive_time Not Exist")
	}
	receiveTimeString, ok := receiveTimeInterface.(string)
	if !ok {
		return time.Time{}, errors.New("receive_time Not String")
	}
	receiveTime, err := time.ParseInLocation("2006-01-02 15:04:05", receiveTimeString,
		time.Now().Location())
	if err != nil {
		return time.Time{}, errors.Errorf("Parse %s error %v", receiveTimeString, err)
	}
	return receiveTime, nil
}

func (r SubResultResume) CVName() (string, error) {
	Interface, ok := r["name"]
	if !ok {
		return "", errors.New("cv name Not Exist")
	}
	res, ok := Interface.(string)
	if !ok {
		return "", errors.New("cv name Not String")
	}
	return res, nil
}

func (r SubResultResume) JDName() (string, error) {
	Interface, ok := r["position_name"]
	if !ok {
		return "", errors.New("jd name Not Exist")
	}
	res, ok := Interface.(string)
	if !ok {
		return "", errors.New("jd name Not String")
	}
	return res, nil
}

func (r SubResultResume) Content() (string, error) {
	content := make(map[string]interface{})
	for key, value := range r {
		switch key {
		case "html":
			htmlInterface, ok := r["html"]
			if !ok {
				return "", errors.New("html Not Exist")
			}
			htmlContent, ok := htmlInterface.(string)
			if !ok {
				htmlContentJson, ok := htmlInterface.(map[string]interface{})
				if !ok {
					return "", errors.New("html Not Object")
				}
				htmlContentBytes, _ := json.Marshal(htmlContentJson)
				htmlContent = string(htmlContentBytes)
			}
			dfsFileID, err := dfs.GetConstClient().Set("html", []byte(htmlContent))
			if err != nil {
				return "", errors.Errorf("Upload html To Dfs %v", err)
			}
			content["html"] = dfsFileID
		case "json":
			jsonInterface, ok := r["json"]
			if !ok {
				return "", errors.New("json Not Exist")
			}
			jsonContent, ok := jsonInterface.(string)
			if !ok {
				jsonContentJson, ok := jsonInterface.(map[string]interface{})
				if !ok {
					return "", errors.New("json Not Object")
				}
				jsonContentBytes, _ := json.Marshal(jsonContentJson)
				jsonContent = string(jsonContentBytes)
			}
			dfsFileID, err := dfs.GetConstClient().Set("json", []byte(jsonContent))
			if err != nil {
				return "", errors.Errorf("Upload json To Dfs %v", err)
			}
			content["json"] = dfsFileID
		case "img":
			imgBase64Interface, ok := r["img"]
			if !ok {
				return "", errors.New("img Not Exist")
			}
			imgBase64String, ok := imgBase64Interface.(string)
			if !ok {
				return "", errors.New("img Not String")
			}

			dfsFileID, err := dfs.GetConstClient().Set("img", []byte(imgBase64String))
			if err != nil {
				return "", errors.Errorf("Upload img To Dfs %v", err)
			}
			content["img"] = dfsFileID
		case "pdf":
			pdfBase64Interface, ok := r["pdf"]
			if !ok {
				return "", errors.New("pdf Not Exist")
			}
			pdfBase64String, ok := pdfBase64Interface.(string)
			if !ok {
				return "", errors.New("pdf Not String")
			}

			content["pdf"] = ""
			if len(pdfBase64String) > 1000 {
				dfsFileID, err := dfs.GetConstClient().Set("pdf", []byte(pdfBase64String))
				if err != nil {
					return "", errors.Errorf("Upload pdf To Dfs %v", err)
				}
				content["pdf"] = dfsFileID
			}

		default:
			content[key] = value
		}
	}
	d, err := json.Marshal(content)
	if err != nil {
		return "", errors.Wrap(err, "Json Marshal")
	}
	return string(d), nil
}
