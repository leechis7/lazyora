package oracle

import "testing"

func TestPrepare(t *testing.T) {
	for _, tc := range []struct {
		input, want string
		query       bool
	}{
		{"SELECT ';' FROM dual;", "SELECT ';' FROM dual", true},
		{"-- note\n/* note */ SELECT 1 FROM dual;", "-- note\n/* note */ SELECT 1 FROM dual", true},
		{"WITH x AS (SELECT 1 n FROM dual) SELECT * FROM x;", "WITH x AS (SELECT 1 n FROM dual) SELECT * FROM x", true},
		{"BEGIN NULL; END;\n/", "BEGIN NULL; END;", false},
		{"DECLARE n NUMBER; BEGIN n := 1; END;", "DECLARE n NUMBER; BEGIN n := 1; END;", false},
		{"UPDATE t SET n = 1;", "UPDATE t SET n = 1", false},
	} {
		got, query, err := Prepare(tc.input)
		if err != nil || got != tc.want || query != tc.query {
			t.Errorf("Prepare(%q) = %q, %v, %v", tc.input, got, query, err)
		}
	}
	for _, input := range []string{"", "-- comment", "/* open"} {
		if _, _, err := Prepare(input); err == nil {
			t.Errorf("expected error for %q", input)
		}
	}
}
