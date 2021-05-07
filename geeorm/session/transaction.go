package session

import "Gee/geeorm/log"

// 开启事务
func (s *Session) Begin() (err error) {
	log.Info("transaction begin")
	if s.tx, err = s.db.Begin(); err != nil {
		log.Error(err)
		return err
	}
	return
}

func (s *Session) Commit() (err error) {
	log.Info("transaction commit")
	if err := s.tx.Commit(); err != nil {
		log.Error(err)
		return err
	}
	return
}

func (s *Session) Rollback() (err error) {
	log.Info("transaction rollback")
	if err := s.tx.Rollback(); err != nil {
		log.Error(err)
	}
	return
}
