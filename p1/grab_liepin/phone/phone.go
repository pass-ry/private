package phone

import (
	"time"

	"github.com/pkg/errors"
	"gitlab.ifchange.com/data/cordwood/redis"
)

var (
	constAuthCodeWaitDuration = int((time.Duration(10) * time.Minute).Seconds())
)

func Write(phone string, msg string) error {
	conn, err := redis.GetConstClient()
	if err != nil {
		return errors.Wrap(err, "redis.GetConstClient()")
	}
	defer conn.Close()
	authCodeKey, err := conn.DoString("GET", key(phone))
	if err != nil {
		return errors.Wrap(err, "GetAuthCodeKey")
	}
	_, err = conn.Do("SETEX", authCodeKey, constAuthCodeWaitDuration, msg)
	if err != nil {
		return errors.Wrap(err, "SaveAuthCode")
	}
	return nil
}

func ReadyWrite(phone string, authCodeKey string) error {
	conn, err := redis.GetConstClient()
	if err != nil {
		return errors.Wrap(err, "redis.GetConstClient()")
	}
	defer conn.Close()
	_, err = conn.Do("SETEX", key(phone), constAuthCodeWaitDuration, authCodeKey)
	if err != nil {
		return errors.Wrap(err, "SaveAuthCodeKey")
	}
	return nil
}

func key(phone string) string {
	return "grabliepin_save_phone_auth_code_key_" + phone
}
