package client

import (
	"context"
	"fmt"
	"grabmail/grab/mail"
	"grabmail/models/account"
	"strings"
	"sync"
)

var (
	constReceiveFolder          []string = []string{"INBOX", "其他文件夹", "Deleted Messages", "Junk", "已删除", "垃圾邮件", "其它邮件", "FoxmailSpamBox"}
	constReceiveFolderOnlyInbox []string = []string{"INBOX"}
	constMustSkipFolder         []string = []string{
		"Sent Messages",
		"已发送",
		"Drafts",
		"草稿",
		"联系人",
		"任务",
		"日记",
		"日历",
	}

	switchConstReceiveFolder sync.Map
)

func isReceiveMailBox(username string, customInbox string, mailboxFolder []string) ([]string, bool) {
	result, onPurpose := isReceiveMailBoxCore(username, customInbox, mailboxFolder)
	var back []string
	for folder, _ := range result {
		back = append(back, folder)
	}
	return back, onPurpose
}

func isReceiveMailBoxCore(username string, customInbox string, mailboxFolder []string) (map[string]struct{}, bool) {
	var mailboxFolderAfterSkip = make(map[string]struct{})
	for _, folder := range mailboxFolder {
		mailboxFolderAfterSkip[folder] = struct{}{}
	}

	for _, folder := range mailboxFolder {
		for _, skipFolder := range constMustSkipFolder {
			if strings.Contains(folder, skipFolder) {
				delete(mailboxFolderAfterSkip, folder)
			}
		}
	}

	onPurpose := false
	allReceiveFolder := constReceiveFolder

	if _, ok := switchConstReceiveFolder.Load(username); ok {
		onPurpose = true
		// INBOX Only
		allReceiveFolder = constReceiveFolderOnlyInbox
		switchConstReceiveFolder.Delete(username)
	} else {
		switchConstReceiveFolder.Store(username, true)
		return mailboxFolderAfterSkip, false
		// for _, customI := range strings.Split(customInbox, ",") {
		// 	if len(customI) > 0 {
		// 		allReceiveFolder = append(allReceiveFolder, customI)
		// 	}
		// }
	}

	result := make(map[string]struct{})
	for folder, _ := range mailboxFolderAfterSkip {
		for _, receiveFolder := range allReceiveFolder {
			if receiveFolder == folder || strings.HasPrefix(folder, receiveFolder+"/") {
				result[folder] = struct{}{}
				continue
			}
		}
	}
	return result, onPurpose
}

type Client interface {
	Close()
	GetIndexMail(ctx context.Context) (map[string]*mail.IndexMail, error)
	FetchInboxMail(ctx context.Context, indexMail []*mail.IndexMail) ([]*mail.InboxMail, error)
}

func NewClient(ctx context.Context, ac *account.Account, isLogin bool) (cli Client, errMsg string, err error) {
	switch ac.TYPE() {
	case account.IMAP:
		cli, errMsg, err = newIMAP(ctx, ac)
	case account.POP3:
		cli, errMsg, err = newPOP3(ctx, ac)
	case account.EXCHANGE:
		cli, errMsg, err = newEXCHANGE(ctx, ac, isLogin)
	default:
		panic(fmt.Sprintf("Unknown account type %s", ac.TYPE()))
	}
	return cli, errMsg, err
}
