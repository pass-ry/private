package puber

import (
	"context"
	"fmt"
	"grabliepin/account"
	"grabliepin/suber"
	"time"

	"github.com/beiping96/grace"
	"github.com/pkg/errors"
)

func BindPub(ac *account.Account, source int, accountIsExist bool) (needMsg bool, errMsg string, err error) {
	retryTimes := 10
	pubQueue := ConstLoginPubQueue
	subQueue := fmt.Sprintf("%s_%s_%s_%d",
		suber.ConstSubQueue, ac.Username, ac.Phone, time.Now().Unix())
	// pub
	err = pubWithPhone(ac, ac.Phone, retryTimes, pubQueue, subQueue)
	if err != nil {
		err = errors.Wrap(err, "Pub Bind")
		return
	}
	// sync handle auth code package
	needMsg, errMsg = suber.BindAuthCodeSub(subQueue)
	if len(errMsg) != 0 {
		return
	}
	// async handle login package
	grace.Go(func(parentCtx context.Context) {
		suber.BindSub(parentCtx, ac, subQueue, source, accountIsExist)
	})
	return
}
