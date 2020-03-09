package grab

import (
	"context"
	"grabmail/models/account"
	"testing"
	"time"

	"gitlab.ifchange.com/data/cordwood/cfg"
	"gitlab.ifchange.com/data/cordwood/cfg/loader"
	"gitlab.ifchange.com/data/cordwood/crypto/des3"
	"gitlab.ifchange.com/data/cordwood/mysql"
)

func TestMain(m *testing.M) {
	loader.LoadCfgInDev("grabmail")
	mysql.Construct(cfg.GetCfgMySQL())
	des3.Setup(cfg.GetCfgCustom().Get("DES3"),
		true /* open memory-cache */)

	m.Run()
}

func Test_GRAB_IMAP(t *testing.T) {
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

	allSize, notExistSize, successSize, err := run(ctx, "TEST-IMAP", imapAC, true)
	if err != nil {
		t.Fatalf("GRAB IMAP %v", err)
	}
	if notExistSize != successSize {
		t.Fatalf("GRAB IMAP notExistSize:%d != successSize:%d",
			notExistSize, successSize)
	}
	t.Logf("GRAB IMAP %d/%d/%d", allSize, notExistSize, successSize)
}

func Test_GRAB_POP3(t *testing.T) {
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

	allSize, notExistSize, successSize, err := run(ctx, "TEST-POP3", popAC, true)
	if err != nil {
		t.Fatalf("GRAB POP3 %v", err)
	}
	if notExistSize != successSize {
		t.Fatalf("GRAB POP3 notExistSize:%d != successSize:%d",
			notExistSize, successSize)
	}
	t.Logf("GRAB POP3 %d/%d/%d", allSize, notExistSize, successSize)
}
