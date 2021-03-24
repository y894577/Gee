package geeorm

import (
	"Gee/geeorm/dialect"
	"Gee/geeorm/log"
	"Gee/geeorm/session"
	"database/sql"
)

type Engine struct {
	db      *sql.DB
	dialect dialect.Dialect
}

func NewEngine(driver, source string) (e *Engine, err error) {
	db, err := sql.Open(driver, source) // 连接数据库，返回*sql.DB
	if err != nil {
		log.Error(err)
		return nil, err
	}
	// 检查数据库是否能够正常连接
	if err = db.Ping(); err != nil {
		log.Error(err)
		return nil, err
	}

	d, ok := dialect.GetDialect(driver)

	if !ok {
		log.Errorf("dialect %s Not Found", driver)
	}

	e = &Engine{
		db:      db,
		dialect: d,
	}

	log.Info("Connect database success")
	return
}

func (engine *Engine) Close() {
	if err := engine.db.Close(); err != nil {
		log.Error("Failed to close database")
	}
	log.Info("Close database success")
}

// 创建会话
func (engine *Engine) NewSession() *session.Session {
	return session.New(engine.db, engine.dialect)
}
