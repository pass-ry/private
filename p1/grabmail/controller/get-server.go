package controller

import (
	"net"
	"strings"

	"github.com/pkg/errors"
	handler "gitlab.ifchange.com/data/cordwood/rpc/rpc-handler"
)

func getServer(req *handler.Request, rsp *handler.Response, p *params, splitUserName []string) error {
	domain := splitUserName[1]
	mxs, err := net.LookupMX(domain)
	if err != nil {
		return handler.WrapError(errors.Wrapf(err, "net.LookupMX(%s)", domain),
			85084003, "domain无效")
	}
	if len(mxs) == 0 {
		return handler.WrapError(errors.Errorf("len(mxs) == 0 domain %s", domain),
			85084004, "domain解析失败")
	}
	mx := ""
	for _, mxS := range strings.Split(mxs[0].Host, ".") {
		if len(mxS) == 0 {
			continue
		}
		if len(mx) == 0 {
			mx = mxS
			continue
		}
		mx += "." + mxS
	}
	if len(mx) == 0 {
		return handler.WrapError(errors.Errorf("len(mxs) == 0 domain %s", domain),
			85084004, "domain解析失败")
	}

	mailServer := domain
	switch {
	case strings.Contains(mx, domain) && strings.Contains(domain, "hotmail.com"):
		mailServer = "outlook.com"
	case strings.Contains(mx, "foxmail.com") || strings.Contains(domain, "qq.com"):
		mailServer = "qq.com"
	case strings.Contains(domain, "163.com"):
		mailServer = "163.com"
	case strings.Contains(mx, "zfsc.com"):
		mailServer = "qiye.163.com"
	case strings.Contains(mx, "qq.com"):
		mailServer = "exmail.qq.com"
	case strings.Contains(mx, "qiye163"):
		mailServer = "qiye.163.com"
	case strings.Contains(mx, "263xmail.com"):
		mailServer = "263.net"
	case strings.Contains(mx, "mxhichina.com"):
		mailServer = "mxhichina.com"
	}

	result := map[string]map[string]map[string]string{
		"imap": {
			"0": {
				"mail_server": "",
				"port":        "143",
				"ssl":         "0",
			},
			"1": {
				"mail_server": "",
				"port":        "993",
				"ssl":         "1",
			},
		},
		"pop3": {
			"0": {
				"mail_server": "",
				"port":        "110",
				"ssl":         "0",
			},
			"1": {
				"mail_server": "",
				"port":        "995",
				"ssl":         "1",
			},
		},
	}
	rsp.SetResults(result)

	for mailType, ports := range result {
		prefix := "imap"
		if mailType == "pop3" {
			prefix = "pop"
		}
		if mailServer == "outlook.com" {
			prefix += "-mail"
		}
		for _, port := range ports {
			port["mail_server"] = prefix + "." + mailServer
		}
	}
	return nil
}
