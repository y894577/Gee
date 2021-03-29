package session

import (
	"Gee/geeorm/clause"
	"reflect"
)

func (s *Session) Insert(values ...interface{}) (int64, error) {
	recordValues := make([]interface{}, 0)
	for _, value := range values {
		table := s.Model(value).RefTable()
		// 多次调用 clause.Set() 构造好每一个子句
		s.clause.Set(clause.INSERT, table.Name, table.FieldNames)
		recordValues = append(recordValues, table.RecordValues(value))
	}

	s.clause.Set(clause.VALUES, recordValues...)
	sql, vars := s.clause.Build(clause.INSERT, clause.VALUES)
	result, err := s.Raw(sql, vars...).Exec()
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// values 传入的对象，根据values对象查找是否存在
func (s *Session) Find(values interface{}) error {
	// 通过反射构造对象切片
	destSlice := reflect.Indirect(reflect.ValueOf(values))
	// 获取切片的单个元素的类型 destType
	destType := destSlice.Type().Elem()
	// 映射出表结构 RefTable()
	table := s.Model(reflect.New(destType).Elem().Interface()).RefTable()

	// 根据表结构，使用 clause 构造出 SELECT 语句
	s.clause.Set(clause.SELECT, table.Name, table.FieldNames)
	sql, vars := s.clause.Build(clause.SELECT, clause.WHERE, clause.ORDERBY, clause.LIMIT)
	// 查询到所有符合条件的记录 rows
	rows, err := s.Raw(sql, vars...).QueryRows()
	if err != nil {
		return err
	}

	// 遍历每一行记录
	for rows.Next() {
		// 利用反射创建 destType 的实例 dest
		dest := reflect.New(destType).Elem()
		var values []interface{}
		// 将 dest 的所有字段平铺开，构造切片 values
		for _, name := range table.FieldNames {
			values = append(values, dest.FieldByName(name).Addr().Interface())
		}
		// 调用 rows.Scan()
		// 将该行记录每一列的值依次赋值给 values 中的每一个字段
		if err := rows.Scan(values...); err != nil {
			return err
		}
		destSlice.Set(reflect.Append(destSlice, dest))
	}
	return rows.Close()
}

func (s *Session) Limit(num int) *Session {
	s.clause.Set(clause.LIMIT, num)
	return s
}

func (s *Session) Where(desc string, args ...interface{}) *Session {
	var vars []interface{}
	s.clause.Set(clause.WHERE, append(append(vars, desc), args...)...)
	return s
}

func (s *Session) OrderBy(desc string) *Session {
	s.clause.Set(clause.ORDERBY, desc)
	return s
}
