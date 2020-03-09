package puber

import (
	"context"
	"fmt"
	"grabzhilian/account"
	"grabzhilian/suber"
	"time"

	"github.com/beiping96/grace"
	"github.com/pkg/errors"
)

func BindPub(ac *account.Account, phone string, source int, accountIsExist bool) error {
	retryTimes := 10
	pubQueue := ConstLoginPubQueue
	subQueue := fmt.Sprintf("%s_%s_%d_bind",
		suber.ConstSubQueue, ac.Username, time.Now().Unix())
	err := pub(ac, retryTimes, pubQueue, subQueue, false, phone)
	if err != nil {
		return errors.Wrap(err, "Pub Bind")
	}

	grace.Go(func(parentCtx context.Context) {
		suber.BindSub(parentCtx, ac, subQueue, source, accountIsExist)
	})
	return nil
}

func BindPubCheckPassword(ac *account.Account) (success bool, err error) {
	retryTimes := 10
	pubQueue := ConstLoginPubQueue
	subQueue := fmt.Sprintf("%s_%s_%d_check_password",
		suber.ConstSubQueue, ac.Username, time.Now().Unix())
	if err := pub(ac, retryTimes, pubQueue, subQueue, true, ""); err != nil {
		return false, errors.Wrap(err, "Pub Bind CheckPassword")
	}
	return suber.BindSubCheckPassword(ac, subQueue)
}
