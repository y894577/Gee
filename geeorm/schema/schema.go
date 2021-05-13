package schema

import (
	"Gee/geeorm/dialect"
	"go/ast"
	"reflect"
)

// Field 字段
type Field struct {
	Name string // 字段名
	Type string // 类型
	Tag  string // 约束条件
}

// Schema 数据库表
type Schema struct {
	Model      interface{}       // 被映射的对象
	Name       string            // 表名
	Fields     []*Field          // 字段
	FieldNames []string          // 所有的字段名（列名）
	fieldMap   map[string]*Field // 记录字段名和 Field 的映射关系
}

// Parse 将任意的对象解析为 Schema 实例
func Parse(dest interface{}, d dialect.Dialect) *Schema {
	// reflect.Indirect() 获取指针指向的实例
	modelType := reflect.Indirect(reflect.ValueOf(dest)).Type()
	schema := &Schema{
		Model:    dest,
		Name:     modelType.Name(), // 获取结构体名称作为表名
		fieldMap: make(map[string]*Field),
	}
	// 获取实例的字段个数
	for i := 0; i < modelType.NumField(); i++ {
		p := modelType.Field(i)
		if !p.Anonymous && ast.IsExported(p.Name) {
			// 根据对象的字段名转换成数据表的字段
			field := &Field{
				Name: p.Name, // 字段名
				// p.Type 即字段类型
				// 通过 (Dialect).DataTypeOf() 转换为数据库的字段类型
				Type: d.DataTypeOf(reflect.Indirect(reflect.New(p.Type))),
			}
			if v, ok := p.Tag.Lookup("geeorm"); ok {
				field.Tag = v
			}
			schema.Fields = append(schema.Fields, field)
			schema.FieldNames = append(schema.FieldNames, p.Name)
			schema.fieldMap[p.Name] = field
		}

	}
	return schema
}

// RecordValues 根据数据库中列的顺序，从对象中找到对应的值，按顺序平铺
// 即 u1 := &User{Name: "Tom", Age: 18} 转换为 ("Tom", 18) 这样的格式
func (schema *Schema) RecordValues(dest interface{}) []interface{} {
	destValue := reflect.Indirect(reflect.ValueOf(dest))
	var fieldValues []interface{}
	for _, field := range schema.Fields {
		fieldValues = append(fieldValues, destValue.FieldByName(field.Name).Interface())
	}
	return fieldValues
}
