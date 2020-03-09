package suber

import (
	"context"
	"grabzhilian/account"
	"time"

	"github.com/pkg/errors"
	"gitlab.ifchange.com/data/cordwood/log"
)

var (
	constGuessBindReplyDuration = time.Duration(10) * time.Minute
)

func BindSub(parentCtx context.Context, ac *account.Account, subQueue string,
	source int, accountIsExist bool) {
	ctx, cancel := context.WithTimeout(parentCtx, constGuessBindReplyDuration)
	defer cancel()

	ticker := time.Tick(time.Second)
	for {
		select {
		case <-ticker:
			result, err := sub(subQueue)
			if err == errEmptySubQueue {
				continue
			}
			if err != nil {
				log.Errorf("Suber %s BindSub %v",
					subQueue, err)
				continue
			}
			err = result.handleBind(ac, source, accountIsExist)
			if err != nil {
				log.Errorf("Suber %s BindSub-HandleBind %v",
					subQueue, err)
			}
			return
		case <-ctx.Done():
			ac.IsDeleted = true
			ac.Msg = "timeout"
			ac.Push(-1, source)
			return
		}
	}
}

func BindSubCheckPassword(ac *account.Account, subQueue string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(150)*time.Second)
	defer cancel()

	ticker := time.Tick(time.Second)
	for {
		select {
		case <-ticker:
			result, err := sub(subQueue)
			if err == errEmptySubQueue {
				continue
			}
			if err != nil {
				return false, errors.Wrapf(err, "Suber %s BindSubCheckPassword", subQueue)
			}
			// handle result
			if result.Code == 1 {
				return true, nil
			}
			return false, nil
		case <-ctx.Done():
			return false, errors.Errorf("Suber %s BindSubCheckPassword TIME-OUT", subQueue)
		}
	}
}
