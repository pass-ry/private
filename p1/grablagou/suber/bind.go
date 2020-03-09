package suber

import (
	"context"
	"grablagou/account"
	"time"

	"gitlab.ifchange.com/data/cordwood/log"
)

var (
	constGuessBindReplyDuration = time.Duration(10) * time.Minute
)

func BindSub(parentCtx context.Context, ac *account.Account, subQueue string, source int, accountIsExist bool) {
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
