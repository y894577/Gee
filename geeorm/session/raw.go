package session

import (
	"Gee/geeorm/clause"
	"Gee/geeorm/dialect"
	"Gee/geeorm/log"
	"Gee/geeorm/schema"
	"database/sql"
	"strings"
)

// 创建会话，与数据库进行交互
type Session struct {
	db      *sql.DB         //使用sql.Open()连接数据库成功后返回的指针
	sql     strings.Builder // 拼接sql语句
	sqlVars []interface{}   // sql语句中占位符对应值

	dialect  dialect.Dialect // 数据库类型
	refTable *schema.Schema  // 数据库表

	clause clause.Clause // 用于构造SQL语句
}

func New(db *sql.DB, dialect dialect.Dialect) *Session {
	return &Session{
		db:      db,
		dialect: dialect,
	}
}

func (s *Session) Clear() {
	s.sql.Reset()
	s.sqlVars = nil
	s.clause = clause.Clause{}
}

func (s *Session) DB() *sql.DB {
	return s.db
}

//
func (s *Session) Raw(sql string, values ...interface{}) *Session {
	s.sql.WriteString(sql)
	s.sql.WriteString(" ")
	s.sqlVars = append(s.sqlVars, values...)
	return s
}

func (s *Session) Exec() (result sql.Result, err error) {
	defer s.Clear()
	log.Info(s.sql.String(), s.sqlVars)
	if result, err = s.DB().Exec(s.sql.String(), s.sqlVars...); err != nil {
		log.Error(err)
	}
	return
}

// QueryRow gets a record from db
func (s *Session) QueryRow() *sql.Row {
	defer s.Clear()
	log.Info(s.sql.String(), s.sqlVars)
	return s.DB().QueryRow(s.sql.String(), s.sqlVars...)
}

// QueryRows gets a list of records from db
func (s *Session) QueryRows() (rows *sql.Rows, err error) {
	defer s.Clear()
	log.Info(s.sql.String(), s.sqlVars)
	if rows, err = s.DB().Query(s.sql.String(), s.sqlVars...); err != nil {
		log.Error(err)
	}
	return
}
