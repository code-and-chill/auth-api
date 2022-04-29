package mysql

import (
	"fmt"
	"html/template"
	"strconv"
	"strings"
	"time"
)

// This is a modified version of: https://gist.github.com/paragtokopedia/3a27c2a3ab2c1ac76d456b77e7f98c22

// DynamicQueryBuilder represents query builder.
type DynamicQueryBuilder string

// Expression represents an expression.
type Expression struct {
	Key   string
	Exp   string
	Value interface{}
}

// NewExp instantiates a new Expression.
func (dqb DynamicQueryBuilder) NewExp(key string, assignment string, value interface{}) Expression {
	return Expression{Key: key, Exp: assignment, Value: value}
}

func componentToString(c interface{}) DynamicQueryBuilder {
	switch v := c.(type) {
	case Expression:
		return DynamicQueryBuilder(c.(Expression).ToString())
	case string, *string:
		return DynamicQueryBuilder(c.(string))
	case DynamicQueryBuilder:
		return v
	default:
		return ""
	}
}

// And performs AND operation.
func (dqb DynamicQueryBuilder) And(component ...interface{}) DynamicQueryBuilder {
	return dqb.getOperationExpression("AND", component...)
}

// OR performs OR operation
func (dqb DynamicQueryBuilder) OR(component ...interface{}) DynamicQueryBuilder {
	return dqb.getOperationExpression("OR", component...)
}

func (dqb DynamicQueryBuilder) getOperationExpression(operation string, component ...interface{}) DynamicQueryBuilder {
	if len(component) == 0 {
		return ""
	}
	if len(component) == 1 {
		return componentToString(component[0])
	}
	clauses := make([]string, 0)
	for _, v := range component {
		value := componentToString(v)
		if value != "" {
			clauses = append(clauses, ""+string(value)+"")
		}
	}

	if len(clauses) > 0 {
		return DynamicQueryBuilder("( " + strings.Join(clauses, " "+operation+" ") + ")")
	}

	return ""
}

// Limit performs Limit operation.
func (dqb DynamicQueryBuilder) Limit(offset int, length int) DynamicQueryBuilder {
	query := string(dqb)
	query += " LIMIT " + strconv.Itoa(length) + " OFFSET " + strconv.Itoa(offset)
	return DynamicQueryBuilder(query)
}

// CopyQuery copies query as string.
func (dqb DynamicQueryBuilder) CopyQuery(dest *string) DynamicQueryBuilder {
	*dest = dqb.ToString()
	return dqb
}

// BindSQL binds built query into the main sql.
func (dqb DynamicQueryBuilder) BindSQL(sql string) string {
	if dqb != "" && dqb != "( )" {
		index := strings.Index(dqb.ToString(), "LIMIT")
		if index == 1 {
			return sql + dqb.ToString()
		}

		return sql + " WHERE " + string(dqb)
	}
	return sql
}

// ToString converts this query builder into string.
func (dqb DynamicQueryBuilder) ToString() string {
	return string(dqb)
}

// ToString converts this expression into string.
func (e Expression) ToString() string {
	if e.Value == nil {
		return ""
	}
	var val, clause string
	switch e.Value.(type) {
	case int, int16, int32, int64:
		val = strconv.Itoa(e.Value.(int))
		clause = e.Key + e.Exp + e.getReplaceExp()
	case float32, float64:
		val = fmt.Sprintf("%f", e.Value)
		clause = e.Key + e.Exp + e.getReplaceExp()
	case bool:
		val = fmt.Sprintf("%t", e.Value)
		clause = e.Key + e.Exp + e.getReplaceExp()
	case time.Time:
		val = fmt.Sprintf("%t", e.Value)
		clause = e.Key + e.Exp + e.getReplaceExp()
	case string:
		val = e.Value.(string)
		if strings.TrimSpace(val) == "" {
			return ""
		}
		val = template.HTMLEscapeString(val)
		clause = e.Key + e.Exp + e.getReplaceExp()
	}
	return fmt.Sprintf(clause, val)
}

// HasLikeOperator checks whether this expression contains LIKE operator.
func (e Expression) HasLikeOperator() bool {
	return e.Exp == "LIKE"
}

func (e Expression) getReplaceExp() string {
	switch e.Value.(type) {
	case int, int64, int32, int16:
		return "%s"
	case float32, float64:
		return "%s"
	case bool:
		return "%s"
	}
	if e.HasLikeOperator() {
		return "'%%%s%%'"
	}
	return "'%s'"
}
