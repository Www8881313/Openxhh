package sqlite

import (
	"database/sql"
	"openxhh/loger"

	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"
)

var Db *sql.DB

func Init() {
	var err error
	Db, err = sql.Open("sqlite3", "./sql.db")
	if err != nil {
		loger.Loger.Fatal("[SQLite]无法读取文件", zap.Error(err))
	}
	Db.SetMaxOpenConns(1)
	Db.SetMaxIdleConns(1)
	_, err = Db.Exec("PRAGMA busy_timeout=8000")
	if err != nil {
		loger.Loger.Fatal("[SQLite]无法设置等待锁超时", zap.Error(err))
	}
	_, err = Db.Exec(`
	CREATE TABLE IF NOT EXISTS at (
	msg_id BIGINT PRIMARY KEY,
	comment_a_id BIGINT,
	comment_root_id BIGINT,
	link_id BIGINT,
	user_a_id BIGINT,
	comment_text TEXT,
	reply boolean
	)
	`)
	if err != nil {
		loger.Loger.Fatal("[Sqlite]无法创建新的数据库", zap.Error(err))
	}
	err = Db.Ping()
	if err != nil {
		loger.Loger.Fatal("[Sqlite]无法连接至新的数据库", zap.Error(err))
	}
	loger.Loger.Info("[SQLite]READY!")
}
