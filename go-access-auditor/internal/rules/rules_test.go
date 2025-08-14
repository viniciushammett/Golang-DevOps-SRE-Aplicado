package rules

import "testing"

func TestRuleSet(t *testing.T) {
	rs := New([]Rule{
		{Name:"rm-root", Regex:`(?i)\brm\s+-rf\s+/(?:\s|$)`},
		{Name:"drop-db", Regex:`(?i)\bdrop\s+database\b`},
	})
	tests := []struct{
		line string
		want bool
	}{
		{"rm -rf /", true},
		{"DROP DATABASE foo", true},
		{"echo ok", false},
	}
	for _, tt := range tests {
		got, _ := rs.Match(tt.line)
		if got != tt.want {
			t.Fatalf("line=%q got=%v want=%v", tt.line, got, tt.want)
		}
	}
}