package controller

import (
	"fmt"
	"strings"
	"sync"

	"gitlab.ifchange.com/data/cordwood/counter"

	"github.com/pkg/errors"
	handler "gitlab.ifchange.com/data/cordwood/rpc/rpc-handler"
)

type params struct {
	UserName        string `json:"username,omitempty"`
	Name            string `json:"name,omitempty"`
	Password        string `json:"password,omitempty"`
	MailServer      string `json:"mail_server,omitempty"`
	Port            int    `json:"port,omitempty"`
	Ssl             int    `json:"ssl,omitempty"`
	ServerType      string `json:"server_type,omitempty"`
	LastReceiveTime int64  `json:"last_receive_time,omitempty"`
	Source          int    `json:"source,omitempty"`
}

var (
	paramsPOOL = sync.Pool{New: func() interface{} { return new(params) }}

	nilParams = params{}
)

func newParams() *params {
	return paramsPOOL.Get().(*params)
}

func (p *params) close() {
	if p == nil {
		return
	}
	*p = nilParams
	paramsPOOL.Put(p)
}

func (p *params) String() string {
	return fmt.Sprintf("username:%s password:****** mail_server:%s port:%d ssl:%d server_type:%s last_receive_time:%d source:%d",
		p.UserName, p.MailServer, p.Port, p.Ssl, p.ServerType, p.LastReceiveTime, p.Source)
}

func MailToB(req *handler.Request, rsp *handler.Response) error {
	p := newParams()
	defer p.close()

	if err := req.Unmarshal(p); err != nil {
		return err
	}
	if len(p.UserName) == 0 {
		return handler.WrapError(errors.Errorf("NIL Username"),
			85084002, "username无效")
	}
	p.UserName = strings.ToLower(p.UserName)
	splitUserName := strings.Split(p.UserName, "@")
	if len(splitUserName) != 2 {
		return handler.WrapError(errors.Errorf("Bad Username %s", p.UserName),
			85084002, "username无效")
	}
	switch req.GetM() {
	case "get_server":
		return getServer(req, rsp, p, splitUserName)
	case "unbind":
		return mailUnbind(req, rsp, p)
	case "test":
		return mailTest(req, rsp, p, splitUserName)
	case "status":
		return mailStatus(req, rsp, p)
	default: // bind
		counter.NewGrabAdminRPCCounter(counter.Kind_Bind, 0).Inc(true, "mail-bind", "unknown-bind-result")
		return mailBind(req, rsp, p, splitUserName)
	}
}
