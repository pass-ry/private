package mysql

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type Config struct {
	Address  string
	Port     string
	DB       string
	Username string
	Password string

	KeepAlive    int
	MaxOpenConns int
	MaxIdleConns int
}

const constKeepAlive = true

var ErrNoRows = sql.ErrNoRows

func Connect(cfgMySQL Config) (*sql.DB, error) {
	conn, err := sql.Open("mysql",
		getConnStr(cfgMySQL))
	if err != nil {
		return nil, fmt.Errorf("try connect MySQL %+v error %v",
			cfgMySQL, err)
	}
	conn.SetConnMaxLifetime(time.Duration(3) * time.Second)
	if constKeepAlive && cfgMySQL.KeepAlive > 0 {
		conn.SetConnMaxLifetime(time.Duration(cfgMySQL.KeepAlive) * time.Second)
	}
	conn.SetMaxOpenConns(10)
	if constKeepAlive && cfgMySQL.MaxOpenConns > 0 {
		conn.SetMaxOpenConns(cfgMySQL.MaxOpenConns)
	}
	conn.SetMaxIdleConns(10)
	if constKeepAlive && cfgMySQL.MaxIdleConns > 0 {
		conn.SetMaxIdleConns(cfgMySQL.MaxIdleConns)
	}
	err = conn.Ping()
	return conn, err
}

func GetConnStr(cfgMySQL Config) string {
	return getConnStr(cfgMySQL)
}

func getConnStr(cfgMySQL Config) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8&sql_notes=false&sql_notes=false&timeout=90s&collation=utf8_general_ci&parseTime=True&loc=Local&interpolateParams=true",
		cfgMySQL.Username, cfgMySQL.Password,
		cfgMySQL.Address, cfgMySQL.Port,
		cfgMySQL.DB)
}
