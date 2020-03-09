package suber

import (
	"context"
	"fmt"
	"grabliepin/account"
	"grabliepin/phone"
	"time"

	"gitlab.ifchange.com/data/cordwood/log"
)

var (
	constGuessAuthCodeReplyDuration = time.Minute
	constGuessBindReplyDuration     = time.Duration(10) * time.Minute
)

func BindAuthCodeSub(subQueue string) (needMsg bool, errMsg string) {
	ctx, cancel := context.WithTimeout(context.Background(), constGuessAuthCodeReplyDuration)
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
				log.Errorf("Suber %s BindAuthCodeSub %v",
					subQueue, err)
				continue
			}
			switch result.AuthCodeStatus {
			case 0:
				errMsg = "非预期状态0"
				return
			case 1: // 需要验证码
				if len(result.Msg) > 0 {
					errMsg = result.Msg
					return
				}
				if len(result.Phone) == 0 || len(result.AuthCodeKey) == 0 {
					errMsg = "非预期状态"
					return
				}
				needMsg = true
				err = phone.ReadyWrite(result.Phone, result.AuthCodeKey)
				if err != nil {
					log.Errorf("Suber %s BindAuthCodeSub phone.ReadyWrite %v",
						subQueue, err)
					needMsg = false
					errMsg = "系统异常"
					return
				}
				return
			case 2: // 不需要验证码
				return
			default:
				errMsg = fmt.Sprintf("非预期状态%d", result.AuthCodeStatus)
				return
			}
		case <-ctx.Done():
			errMsg = "系统超时"
			return
		}
	}
}

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
