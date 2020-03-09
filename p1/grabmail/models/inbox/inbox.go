package inbox

import (
	"fmt"
	"sync"
)

var (
	ConstReadBodyErrorFunc     = func(reason string) string { return fmt.Sprintf("ReadBodyError %s", reason) }
	ConstNilBodyFunc           = func() string { return "NilBody" }
	ConstBeforeReceiveTimeFunc = func() string { return "BeforeReceiveTime" }
	ConstUnknownMailFromFunc   = func(detail string) string { return fmt.Sprintf("UnknownMailFrom %s", detail) }

	inboxPOOL = sync.Pool{New: func() interface{} { return new(Inbox) }}
	nilInbox  = Inbox{}
)

func NewInbox() *Inbox {
	return inboxPOOL.Get().(*Inbox)
}

func (i *Inbox) Close() {
	if i == nil {
		return
	}
	*i = nilInbox
	inboxPOOL.Put(i)
}

type Inbox struct {
	ID        int    `json:"id"`
	AccountID int    `json:"account_id"`
	UID       string `json:"uid"`
	InboxName string `json:"inbox_name"`
	Subject   string `json:"subject"`
	SendTime  string `json:"send_time"`
	SiteID    int    `json:"site_id"`
	Status    int    `json:"status"`
	Msg       string `json:"msg"`

	ContentDfs string `json:"content_dfs"`
	AttachDfs  string `json:"attach_dfs"`
}
