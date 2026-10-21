package oracle

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrEditConflict = errors.New("row changed or was deleted; reload table data before editing")

type EditChange struct {
	ID          uint64
	Row, Column int
	Text        string
	Null        bool
}
type editSnapshot struct {
	id      uint64
	object  Object
	columns []Column
	rows    [][]Cell
	rowIDs  []string
}

// EditableTable creates its own unambiguous single-table query. Arbitrary SQL,
// views, joins, synonyms and result column labels never grant edit capability.
func (s *Session) EditableTable(ctx context.Context, object Object) (Result, error) {
	if s.finishErr != nil {
		return Result{}, s.finishErr
	}
	if object.Kind != "TABLE" || object.Owner == "" || object.Name == "" {
		return Result{}, errors.New("select a base table in the schema browser")
	}
	rows, err := s.dictionaryQuery(ctx, `SELECT COUNT(*) FROM all_tables WHERE owner=:1 AND table_name=:2 AND iot_type IS NULL AND nested='NO' AND temporary='N'`, object.Owner, object.Name)
	if err != nil {
		return Result{}, err
	}
	var count int
	if rows.Next() {
		err = rows.Scan(&count)
	} else {
		err = rows.Err()
	}
	rows.Close()
	if err != nil {
		return Result{}, err
	}
	if count != 1 {
		return Result{}, errors.New("only ordinary persistent heap tables support safe editing")
	}
	columns, err := s.Columns(ctx, object)
	if err != nil {
		return Result{}, err
	}
	if len(columns) == 0 || len(columns) > 64 {
		return Result{}, errors.New("safe editing supports tables with 1–64 columns")
	}
	expr := []string{"ROWIDTOCHAR(t.ROWID)"}
	for i := range columns {
		c := columns[i]
		kind := c.DataType
		if !strings.Contains(kind, "TIME ZONE") {
			kind, _, _ = strings.Cut(kind, "(")
		}
		columns[i].DataType = kind
		c.DataType = kind
		if !editableType(c.DataType) {
			return Result{}, fmt.Errorf("safe editing does not support table column %s (%s)", c.Name, c.DataType)
		}
		expr = append(expr, editExpression("t."+quoteIdentifier(c.Name), c.DataType)+" AS "+quoteIdentifier(c.Name))
	}
	result, err := s.Execute(ctx, "SELECT "+strings.Join(expr, ",")+" FROM "+QualifiedObject(object)+" t WHERE ROWNUM <= 1000")
	if err != nil {
		return result, err
	}
	if len(result.Rows) == 1000 {
		result.Truncated = true
	}
	snapshot := &editSnapshot{object: object, columns: columns}
	s.editSequence++
	snapshot.id = s.editSequence
	result.Columns = result.Columns[1:]
	result.Query = nil
	for i := range result.Rows {
		snapshot.rowIDs = append(snapshot.rowIDs, result.Rows[i][0])
		result.Rows[i] = result.Rows[i][1:]
		result.Values[i] = result.Values[i][1:]
		for j := range columns {
			result.Values[i][j].Type = columns[j].DataType
		}
		snapshot.rows = append(snapshot.rows, append([]Cell(nil), result.Values[i]...))
	}
	s.edit = snapshot
	result.EditID = snapshot.id
	result.EditObject = object
	return result, nil
}
func editableType(kind string) bool {
	return kind == "NUMBER" || kind == "VARCHAR2" || kind == "CHAR" || kind == "DATE" || kind == "RAW" || strings.HasPrefix(kind, "TIMESTAMP(") && !strings.Contains(kind, "TIME ZONE") || kind == "TIMESTAMP"
}
func editExpression(column, kind string) string {
	if kind == "DATE" {
		return "TO_CHAR(" + column + ",'YYYY-MM-DD HH24:MI:SS')"
	}
	if strings.HasPrefix(kind, "TIMESTAMP") {
		return "TO_CHAR(" + column + ",'YYYY-MM-DD HH24:MI:SS.FF9')"
	}
	if kind == "RAW" {
		return "RAWTOHEX(" + column + ")"
	}
	return column
}
func editBind(name, kind string) string {
	if kind == "DATE" {
		return "TO_DATE(" + name + ",'YYYY-MM-DD HH24:MI:SS')"
	}
	if strings.HasPrefix(kind, "TIMESTAMP") {
		return "TO_TIMESTAMP(" + name + ",'YYYY-MM-DD HH24:MI:SS.FF9')"
	}
	if kind == "RAW" {
		return "HEXTORAW(" + name + ")"
	}
	return name
}
func editValue(kind, text string, null bool) (any, error) {
	if null {
		return nil, nil
	}
	if kind == "NUMBER" {
		return (BindValue{Kind: "number", Text: text}).Value()
	}
	if kind == "DATE" {
		if _, err := time.Parse("2006-01-02 15:04:05", text); err != nil {
			return nil, errors.New("DATE requires YYYY-MM-DD HH:MM:SS")
		}
	}
	if strings.HasPrefix(kind, "TIMESTAMP") {
		if _, err := time.Parse("2006-01-02 15:04:05.999999999", text); err != nil {
			return nil, errors.New("TIMESTAMP requires YYYY-MM-DD HH:MM:SS[.fraction]")
		}
	}
	if kind == "RAW" {
		if len(text)%2 != 0 {
			return nil, errors.New("RAW requires even-length hexadecimal")
		}
		for _, r := range text {
			if !strings.ContainsRune("0123456789abcdefABCDEF", r) {
				return nil, errors.New("RAW requires hexadecimal")
			}
		}
	}
	if len(text) > 32767 {
		return nil, errors.New("new cell value is too large for safe editing")
	}
	return text, nil
}
func (s *Session) EditID() uint64 {
	if s.edit == nil {
		return 0
	}
	return s.edit.id
}

// ApplyEdit binds all values and compares every original column plus ROWID.
// A savepoint undoes this attempt on error/conflict, preserving earlier work.
// Success remains uncommitted and invalidates the snapshot (triggers may run).
func (s *Session) ApplyEdit(ctx context.Context, change EditChange) (err error) {
	if s.finishErr != nil {
		return s.finishErr
	}
	snap := s.edit
	if snap == nil || change.ID == 0 || snap.id != change.ID || change.Row < 0 || change.Row >= len(snap.rows) || change.Column < 0 || change.Column >= len(snap.columns) {
		return errors.New("editable result expired; reload table data")
	}
	column := snap.columns[change.Column]
	value, err := editValue(column.DataType, change.Text, change.Null)
	if err != nil {
		return err
	}
	if !column.Nullable && (change.Null || change.Text == "") {
		return errors.New("column does not allow NULL (Oracle treats empty text as NULL)")
	}
	args := []any{sql.Named("new_value", value), sql.Named("row_id", snap.rowIDs[change.Row])}
	conditions := []string{"ROWID = CHARTOROWID(:row_id)"}
	for i, c := range snap.columns {
		old := snap.rows[change.Row][i]
		arg, e := editValue(c.DataType, old.Text, old.Null)
		if e != nil {
			return e
		}
		name := fmt.Sprintf("old_%d", i)
		args = append(args, sql.Named(name, arg))
		expr := editExpression(quoteIdentifier(c.Name), c.DataType)
		conditions = append(conditions, "("+expr+" = :"+name+" OR ("+expr+" IS NULL AND :"+name+" IS NULL))")
	}
	statement := "UPDATE " + QualifiedObject(snap.object) + " SET " + quoteIdentifier(column.Name) + " = " + editBind(":new_value", column.DataType) + " WHERE " + strings.Join(conditions, " AND ")
	var nonce [8]byte
	if _, err = rand.Read(nonce[:]); err != nil {
		return err
	}
	savepoint := fmt.Sprintf("LAZYORA_%X", nonce)
	if _, err = s.tx.ExecContext(ctx, "SAVEPOINT "+savepoint); err != nil {
		return err
	}
	previousPending := s.pending
	s.pending = true
	start := time.Now()
	defer func() {
		s.edit = nil
		if err != nil {
			cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if _, rollbackErr := s.tx.ExecContext(cleanup, "ROLLBACK TO SAVEPOINT "+savepoint); rollbackErr != nil {
				s.finishErr = fmt.Errorf("grid update outcome is uncertain; reconnect: %w", rollbackErr)
				err = errors.Join(err, s.finishErr)
			} else {
				s.pending = previousPending
			}
		}
		s.recordStatement(start, statement, func() int64 {
			if err == nil {
				return 1
			}
			return 0
		}(), err)
	}()
	var locked string
	err = s.tx.QueryRowContext(ctx, "SELECT ROWIDTOCHAR(ROWID) FROM "+QualifiedObject(snap.object)+" WHERE ROWID = CHARTOROWID(:row_id) FOR UPDATE NOWAIT", sql.Named("row_id", snap.rowIDs[change.Row])).Scan(&locked)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrEditConflict
	}
	if err != nil {
		return fmt.Errorf("could not lock row without waiting; reload or retry after the other transaction finishes: %w", err)
	}
	exec, err := s.tx.ExecContext(ctx, statement, args...)
	if err != nil {
		return err
	}
	count, err := exec.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return ErrEditConflict
	}
	return nil
}
