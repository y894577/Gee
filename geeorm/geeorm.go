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

// NewSession 创建会话
func (engine *Engine) NewSession() *session.Session {
	return session.New(engine.db, engine.dialect)
}

// TxFunc 自定义一个函数类型
type TxFunc func(*session.Session) (interface{}, error)

// Transaction 传入上面的函数类型
func (engine *Engine) Transaction(f TxFunc) (result interface{}, err error) {
	s := engine.NewSession()
	if err := s.Begin(); err != nil {
		return nil, err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = s.Rollback()
			panic(p) // re-throw panic after Rollback
		} else if err != nil {
			_ = s.Rollback() // err is non-nil; don't change it
		} else {
			err = s.Commit() // err is nil; if Commit returns error update err
		}
	}()
	return f(s)
}
