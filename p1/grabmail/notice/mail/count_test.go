package mail

import (
	"testing"
)

func TestCount(t *testing.T) {
	if err := sendMail(count()); err != nil {
		t.Fatal(err)
	}
}
