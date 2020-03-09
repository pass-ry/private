package puber

import (
	"context"
	"fmt"
	"grablagou/account"
	"grablagou/suber"
	"time"

	"github.com/beiping96/grace"
	"github.com/pkg/errors"
)

func BindPub(ac *account.Account, source int, accountIsExist bool) error {
	retryTimes := 10
	pubQueue := ConstLoginPubQueue
	subQueue := fmt.Sprintf("%s_%s_%d",
		suber.ConstSubQueue, ac.Username, time.Now().Unix())
	err := pub(ac, retryTimes, pubQueue, subQueue)
	if err != nil {
		return errors.Wrap(err, "Pub Bind")
	}

	grace.Go(func(parentCtx context.Context) {
		suber.BindSub(parentCtx, ac, subQueue, source, accountIsExist)
	})
	return nil
}
