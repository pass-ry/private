package mail

import (
	"testing"
)

func TestSendMail(t *testing.T) {
	if err := sendMail(`<html><body><h3>Hello GrabMail</h3></body></html>`); err != nil {
		t.Fatal(err)
	}
}
