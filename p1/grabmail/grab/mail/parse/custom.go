package parse

import (
	"bytes"
	"fmt"
	"grabmail/models/inbox"
	"io"
	"io/ioutil"
	"regexp"

	"github.com/jhillyerd/enmime"
)

func readEnvelope(i *inbox.Inbox, body io.Reader) (*enmime.Envelope, bool) {
	b, _ := ioutil.ReadAll(body)

	env, step1Err := enmime.ReadEnvelope(bytes.NewBuffer(b))
	if step1Err == nil {
		return env, true
	}

	s := regexp.MustCompile(`(filename|name)==\?UTF-8.*`).ReplaceAll(b, nil)
	env, step2Err := enmime.ReadEnvelope(bytes.NewBuffer(s))
	if step2Err == nil {
		return env, true
	}
	i.Msg = inbox.ConstReadBodyErrorFunc(fmt.Sprintf("step1:%v step2:%v",
		step1Err, step2Err))
	return env, false
}
