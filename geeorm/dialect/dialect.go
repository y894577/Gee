package dialect

import "reflect"

var dialectsMap = map[string]Dialect{}

// 抽象数据库类型
type Dialect interface {
	DataTypeOf(typ reflect.Value) string                    // 将go的类型转换成数据库数据类型
	TableExistSQL(tableName string) (string, []interface{}) // 返回某个表是否存在
}

// 注册dialect实例
func RegisterDialect(name string, dialect Dialect) {
	dialectsMap[name] = dialect
}

// 获取dialect实例
func GetDialect(name string) (dialect Dialect, ok bool) {
	dialect, ok = dialectsMap[name]
	return
}
