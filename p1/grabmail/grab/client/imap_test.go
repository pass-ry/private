package client

import (
	"context"
	"grabmail/grab/mail"
	"grabmail/models/account"
	"testing"
	"time"
)

func Test_IMAP(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(3)*time.Minute)
	defer cancel()

	imapAC := &account.Account{
		ID:              1,
		UserName:        "ifchangetest@ifchange.com",
		User:            "ifchangetest@ifchange.com",
		Password:        "19C7058B993512EBB2D3DF7EEADD24DF",
		Type:            0,
		MailServer:      "imap.exmail.qq.com",
		Port:            993,
		Ssl:             1,
		Status:          0,
		LastReceiveTime: time.Now(),
		LastCrawlerTime: time.Now(),
		ErrCount:        0,
		CustomInbox:     "",
	}

	cli, _, err := NewClient(ctx, imapAC, false)
	if err != nil {
		t.Fatalf("newIMAP(%+v) = %v", imapAC, err)
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
