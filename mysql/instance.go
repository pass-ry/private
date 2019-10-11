package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"runtime"

	"github.com/beiping96/grace"
)

var (
	constConn *sql.DB
	constCfg  Config
)

func GetConstConn() *sql.DB {
	for i := 0; i < 5; i++ {
		err := constConn.Ping()
		if err == nil {
			break
		}
		constConn, _ = Connect(constCfg)
	}
	return constConn
}

func Construct(cfgMySQL Config) {
	if constConn != nil {
		return
	}
	conn, err := Connect(cfgMySQL)
	if err != nil {
		panic(fmt.Errorf("Connect MySQL %+v %v",
			cfgMySQL, err))
	}
	constConn = conn
	constCfg = cfgMySQL

	grace.Go(connCloseMonitor)
	return
}

func connCloseMonitor(ctx context.Context) {
	<-ctx.Done()
	runtime.Gosched()
	constConn.Close()
}
