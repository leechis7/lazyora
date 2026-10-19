package oracle

import (
	"context"
	"fmt"
	"strings"
)

const MaxScriptStatements = 100

type ScriptStep struct {
	Statement Statement
	Result    Result
	Error     string
}
type ScriptReport struct {
	Steps []ScriptStep
	Total int
}

// ScriptStatements preflights the complete input before executing any part.
// SQL*Plus directives/substitution and transaction options are intentionally
// unsupported rather than silently ignored or converted into another action.
func ScriptStatements(text string) ([]Statement, error) {
	statements, err := Statements(text)
	if err != nil {
		return nil, err
	}
	if len(statements) == 0 {
		return nil, fmt.Errorf("enter a SQL script")
	}
	if len(statements) > MaxScriptStatements {
		return nil, fmt.Errorf("scripts support at most %d statements", MaxScriptStatements)
	}
	for _, stmt := range statements {
		tokens, _ := sqlTokens(stmt.Text)
		word := tokens[0].word
		switch word {
		case "SELECT", "WITH", "INSERT", "UPDATE", "DELETE", "MERGE", "CREATE", "ALTER", "DROP", "TRUNCATE", "RENAME", "COMMENT", "GRANT", "REVOKE", "BEGIN", "DECLARE", "SAVEPOINT", "CALL", "LOCK", "EXPLAIN", "ANALYZE", "AUDIT", "NOAUDIT", "PURGE", "FLASHBACK":
		case "SET":
			if len(tokens) < 2 || (tokens[1].word != "TRANSACTION" && tokens[1].word != "CONSTRAINTS") {
				return nil, fmt.Errorf("line %d: SQL*Plus SET is not supported", stmt.Line)
			}
		case "COMMIT", "ROLLBACK":
			words := tokens
			if words[len(words)-1].word == ";" {
				words = words[:len(words)-1]
			}
			if len(words) > 2 || (len(words) == 2 && words[1].word != "WORK") {
				return nil, fmt.Errorf("line %d: only plain COMMIT/ROLLBACK is supported", stmt.Line)
			}
		default:
			return nil, fmt.Errorf("line %d: unsupported script command %q (SQL*Plus is not supported)", stmt.Line, word)
		}
		for _, t := range tokens {
			if t.word == "&" {
				return nil, fmt.Errorf("line %d: SQL*Plus substitution variables are not supported", stmt.Line)
			}
		}
	}
	return statements, nil
}

// Script runs on the same session and transaction. Stop on first error/cancel;
// successful preceding DML stays pending unless the script explicitly resolves it.
func (s *Session) Script(ctx context.Context, text string) (ScriptReport, error) {
	statements, err := ScriptStatements(text)
	report := ScriptReport{Total: len(statements)}
	if err != nil {
		return report, err
	}
	for index, stmt := range statements {
		step := ScriptStep{Statement: stmt}
		err = ctx.Err()
		if err == nil {
			word, _ := leadingWord(stmt.Text)
			if word == "COMMIT" || word == "ROLLBACK" {
				err = s.Finish(word == "COMMIT")
			} else {
				step.Result, err = s.Execute(ctx, stmt.Text)
			}
		}
		if err != nil {
			step.Error = err.Error()
		}
		report.Steps = append(report.Steps, step)
		if err != nil {
			return report, fmt.Errorf("statement %d (line %d): %w", index+1, stmt.Line, err)
		}
	}
	return report, nil
}

func (r ScriptReport) Summary() string {
	var lines []string
	for i, step := range r.Steps {
		status := "OK"
		if step.Error != "" {
			status = step.Error
		}
		lines = append(lines, fmt.Sprintf("%d (line %d): %s — rows %d, affected %d", i+1, step.Statement.Line, status, len(step.Result.Rows), step.Result.Affected))
	}
	return strings.Join(lines, "\n")
}
