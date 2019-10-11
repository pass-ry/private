package redis

import (
	"fmt"
	"time"

	"github.com/beiping96/rd"
)

type Config struct {
	IsCluster    bool
	Address      string
	Password     string
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	Prefix       string
}

type Client = rd.RD

func NewClient(cfg Config) (cli Client, err error) {
	rdCfg := rd.Config{
		IsCluster:    cfg.IsCluster,
		Address:      cfg.Address,
		Password:     cfg.Password,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		OpenConns:    100,
		Lifetime:     -1,
		Prefix: func(key string) (keyWithPrefix string) {
			return fmt.Sprintf("%s%s", cfg.Prefix, key)
		},
	}

	for i := 0; i < 5; i++ {
		cli, err = rd.New(rdCfg)
		if err == nil {
			return cli, nil
		}
	}
	return nil, fmt.Errorf("Redis New Client Error %v %+v",
		err, cfg)
}
