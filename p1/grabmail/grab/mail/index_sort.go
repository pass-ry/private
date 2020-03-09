package mail

import (
	"grabmail/models/account"
	"sort"
)

func IndexSort(groupSize int, ac *account.Account, source []*IndexMail) []*IndexMail {
	beforeSort := make(map[string][]*IndexMail)

	for _, i := range source {
		beforeSort[i.Inbox] = append(beforeSort[i.Inbox], i)
	}

	afterSort := make(map[string][]*IndexMail)

	for key, ii := range beforeSort {
		is := indexSort{
			s: make([]*IndexMail, len(ii)),
			t: ac.TYPE(),
		}
		copy(is.s, ii)
		sort.Sort(is)
		afterSort[key] = is.s
	}

	result := []*IndexMail{}
	for {
		allDone := true
		for key := range afterSort {
			count := len(afterSort[key])
			if count == 0 {
				continue
			}
			allDone = false
			if count > groupSize {
				count = groupSize
			}
			result = append(result, afterSort[key][:count]...)
			afterSort[key] = afterSort[key][count:]
		}
		if allDone {
			return result
		}
	}
}

type indexSort struct {
	s []*IndexMail
	t account.ClientType
}

func (is indexSort) Len() int      { return len(is.s) }
func (is indexSort) Swap(i, j int) { is.s[i], is.s[j] = is.s[j], is.s[i] }
func (is indexSort) Less(i, j int) bool {
	switch is.t {
	case account.IMAP:
		if is.s[i].Inbox == is.s[j].Inbox {
			return is.s[i].UID > is.s[j].UID
		}
		return is.s[i].Inbox < is.s[j].Inbox
	case account.POP3:
		return is.s[i].NumberID > is.s[j].NumberID
	}
	return is.s[i].UUID > is.s[j].UUID
}
