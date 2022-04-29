package mysql

import (
	"github.com/code-and-chill/auth-api/pkg/filter"
	"testing"
)

func TestDynamicQueryBuilder_GenerateQuery(t *testing.T) {
	tests := []struct {
		name  string
		query string
		args  filter.Filter
		want  string
	}{
		{
			name:  "Generates a correct SQL",
			query: "select * from application a",
			args: filter.Filter{
				"start_date": "2018-01-01",
				"end_date":   "2019-01-01",
				"status":     "some_status",
				"id":         "2",
			},
			want: "select * from application a WHERE ( a.created_at>='2018-01-01 ' AND a.created_at<='2019-01-01 ' AND a.id >= 0 AND a.status IN ('pending','complete','initiated') AND ( a.status='some_status' OR a.id=2))",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dqb DynamicQueryBuilder

			dqb = dqb.And(
				dqb.NewExp("a.created_at", ">=", tt.args["start_date"]+" "+tt.args["start_time"]),
				dqb.NewExp("a.created_at", "<=", tt.args["end_date"]+" "+tt.args["end_time"]),
				"a.id >= 0",
				"a.status IN ('pending','complete','initiated')",
				dqb.OR(
					dqb.NewExp("a.status", "=", tt.args["status"]),
					dqb.NewExp("a.id", "=", tt.args.GetInt("id")),
				),
			)

			if got := dqb.BindSQL(tt.query); got != tt.want {
				t.Errorf("DynamicQueryBuilder.BindSQL() = %v, want %v", got, tt.want)
			}
		})
	}
}
