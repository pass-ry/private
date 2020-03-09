package mail

import (
	"fmt"
	"grabmail/models/account"
	"testing"
)

func TestIndexSort(t *testing.T) {
	index := []*IndexMail{
		&IndexMail{
			Inbox: "a",
			UUID:  "1",
			UID:   1,
		},
		&IndexMail{
			Inbox: "a",
			UUID:  "2",
			UID:   2,
		},
		&IndexMail{
			Inbox: "a",
			UUID:  "3",
			UID:   3,
		},
		&IndexMail{
			Inbox: "b",
			UUID:  "1",
			UID:   1,
		},
		&IndexMail{
			Inbox: "b",
			UUID:  "2",
			UID:   2,
		},
		&IndexMail{
			Inbox: "b",
			UUID:  "3",
			UID:   3,
		},
		&IndexMail{
			Inbox: "c",
			UUID:  "1",
			UID:   1,
		},
		&IndexMail{
			Inbox: "c",
			UUID:  "2",
			UID:   2,
		},
		&IndexMail{
			Inbox: "c",
			UUID:  "3",
			UID:   3,
		},
	}
	got := IndexSort(2, new(account.Account), index)

	print := []string{}
	for _, i := range got {
		print = append(print, fmt.Sprintf("%s:%s", i.Inbox, i.UUID))
	}
	t.Log(print)
}
