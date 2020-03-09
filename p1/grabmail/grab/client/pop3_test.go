package client

import (
	"context"
	"grabmail/grab/mail"
	"grabmail/models/account"
	"testing"
	"time"
)

func Test_POP3(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(3)*time.Minute)
	defer cancel()

	popAC := &account.Account{
		ID:              1,
		UserName:        "haifeng.wu@cheng96.com",
		User:            "haifeng.wu@cheng96.com",
		Password:        "EEDAFB904FE54E11036969957F968F86",
		Type:            1,
		MailServer:      "mail.cheng96.com",
		Port:            110,
		Ssl:             0,
		Status:          0,
		LastReceiveTime: time.Now(),
		LastCrawlerTime: time.Now(),
		ErrCount:        0,
		CustomInbox:     "",
	}

	cli, _, err := NewClient(ctx, popAC, false)
	if err != nil {
		t.Fatalf("newPOP3(%+v) = %v", popAC, err)
	}
	defer cli.Close()

	t.Logf("Create Client Success")

	index, err := cli.GetIndexMail(ctx)
	if err != nil {
		t.Fatalf("GetIndexMail() = %v", err)
	}

	t.Logf("Get Index Mail %d", len(index))

	testSize := 5
	testFetch := []*mail.IndexMail{}
	for _, i := range index {
		testFetch = append(testFetch, i)
		if len(testFetch) == testSize {
			break
		}
	}

	inbox, err := cli.FetchInboxMail(ctx, testFetch)
	if err != nil {
		t.Fatalf("FetchInboxMail() = %v", err)
	}
	if len(inbox) != testSize {
		t.Fatalf("UnFetch Enough mail want:%d got:%d", testSize, len(inbox))
	}
}
