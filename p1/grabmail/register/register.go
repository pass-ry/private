package register

import (
	"fmt"
	"strings"
	"time"

	"gitlab.ifchange.com/data/cordwood/log"
	"gitlab.ifchange.com/data/cordwood/redis"
)

var (
	NilRegister = new(nilRegister)
)

type nilRegister struct{}

func (n *nilRegister) LogIn(keys ...string) (success bool) { return true }
func (n *nilRegister) LogOut(keys ...string)               { return }
func (n *nilRegister) raceKey(keys ...string) string       { return "NIL_REGISTER" }

type ROLE = string

const (
	ROLE_GRAB   ROLE = "ROLE_GRAB"
	ROLE_LG_MAP ROLE = "ROLE_LG_MAP"
	// ROLE_LG_REDUCE    ROLE = "ROLE_LG_REDUCE" // achieve by queue
	ROLE_ZL_MAP ROLE = "ROLE_ZL_MAP"
	// ROLE_ZL_REDUCE    ROLE = "ROLE_ZL_REDUCE" // achieve by queue
	ROLE_ZL_PUSH      ROLE = "ROLE_ZL_PUSH"
	ROLE_ZL_MAIL      ROLE = "ROLE_ZL_MAIL"
	ROLE_PUSH_MAIL    ROLE = "ROLE_PUSH_MAIL"
	ROLE_PUSH_ACCOUNT ROLE = "ROLE_PUSH_ACCOUNT"
	ROLE_COUNT_MAIL   ROLE = "ROLE_COUNT_MAIL"
	ROLE_ACCOUNT_SYNC ROLE = "ROLE_ACCOUNT_SYNC"
)

type Register interface {
	LogIn(keys ...string) (success bool)
	LogOut(keys ...string)

	// private
	raceKey(keys ...string) string
}

func New(role ROLE, expire time.Duration) Register {
	if len(role) == 0 {
		panic("Register NIL ROLE")
	}
	if expire < time.Second {
		panic("Register expire must out of one second")
	}
	return &register{
		role:   role,
		expire: expire,
	}
}

var _ Register = (*register)(nil)

type register struct {
	role   ROLE
	expire time.Duration
}

func (r *register) LogIn(keys ...string) (success bool) {
	conn, err := redis.GetConstClient()
	if err != nil {
		log.Errorf("REGISTER LogIn error redis.GetConstClient() %v",
			err)
		return
	}
	defer conn.Close()

	isWorking, err := conn.DoString("GET", r.raceKey(keys...))
	if err != nil && err != conn.ErrNil() {
		log.Errorf("REGISTER LogIn error GET %v",
			err)
		return
	}
	if len(isWorking) > 0 {
		return
	}
	_, err = conn.Do("SETEX",
		r.raceKey(keys...),
		int(r.expire.Seconds()),
		"working")
	if err != nil {
		log.Errorf("REGISTER LogIn error SETEX %v",
			err)
		return
	}
	return true
}

func (r *register) LogOut(keys ...string) {
	conn, err := redis.GetConstClient()
	if err != nil {
		log.Errorf("REGISTER LogOut error redis.GetConstClient() %v",
			err)
		return
	}
	defer conn.Close()

	if _, err := conn.Do("DEL",
		r.raceKey(keys...)); err != nil {
		log.Errorf("REGISTER LogOut error redis.Do %v",
			err)
	}
}

func (r *register) raceKey(keys ...string) string {
	return fmt.Sprintf("REGISTER_%s_%s",
		r.role,
		strings.Join(keys, "_"))
}
