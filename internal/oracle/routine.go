package oracle

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	go_ora "github.com/sijms/go-ora/v3"
)

// Routine describes one dictionary signature, including a function's position-0 return.
type Routine struct {
	Object           Object
	Member, Overload string
	Arguments        []RoutineArgument
}
type RoutineArgument struct {
	Argument
	PLS, TypeOwner, TypeName, TypeSubname string
	Defaulted                             bool
}
type RoutineInput struct {
	Value   BindValue
	Default bool
}
type RoutineResult struct {
	Routine string
	Scalars Result
	Cursors []NamedResult
	Output  Output
}
type NamedResult struct {
	Name   string
	Result Result
	Null   bool
}

func (s *Session) Routines(ctx context.Context, object Object) ([]Routine, error) {
	if !object.HasArguments() {
		return nil, errors.New("select a procedure, function or package")
	}
	filter := "package_name IS NULL AND object_name = :2"
	packaged := object.Kind == "PACKAGE" || object.Kind == "PACKAGE BODY"
	if packaged {
		filter = "package_name = :2"
	}
	query := `SELECT object_name, NVL(overload, '-'), NVL(argument_name, '-'),
position, data_level, NVL(data_type, '-'), NVL(in_out, '-'),
NVL(pls_type, '-'), type_owner, type_name, type_subname, %s
FROM all_arguments WHERE owner = :1 AND ` + filter + ` ORDER BY object_name, overload, sequence`
	rows, err := s.dictionaryQuery(ctx, fmt.Sprintf(query, "defaulted"), object.Owner, object.Name)
	// DEFAULTED is absent on older Oracle 10g dictionary views. Never retry
	// permission/network errors as if they meant "no defaults".
	if err != nil && strings.Contains(err.Error(), "ORA-00904") {
		rows, err = s.dictionaryQuery(ctx, fmt.Sprintf(query, "'N'"), object.Owner, object.Name)
	}
	if err != nil {
		return nil, err
	}
	var routines []Routine
	for rows.Next() {
		var a RoutineArgument
		var owner, name, subname, defaulted sql.NullString
		if err = rows.Scan(&a.Member, &a.Overload, &a.Name, &a.Position, &a.Depth, &a.DataType, &a.Direction, &a.PLS, &owner, &name, &subname, &defaulted); err != nil {
			rows.Close()
			return nil, err
		}
		a.TypeOwner, a.TypeName, a.TypeSubname = owner.String, name.String, subname.String
		a.Defaulted = defaulted.String == "Y"
		if len(routines) == 0 || routines[len(routines)-1].Member != a.Member || routines[len(routines)-1].Overload != a.Overload {
			routines = append(routines, Routine{Object: object, Member: a.Member, Overload: a.Overload})
		}
		// 10g represents no-argument procedures with an otherwise empty row.
		if a.Name != "-" || a.Position == 0 && a.DataType != "-" {
			routines[len(routines)-1].Arguments = append(routines[len(routines)-1].Arguments, a)
		}
	}
	err = errors.Join(rows.Err(), rows.Close())
	if err != nil {
		return nil, err
	}
	if !packaged {
		if len(routines) == 0 && object.Kind == "PROCEDURE" {
			routines = append(routines, Routine{Object: object, Member: object.Name, Overload: "-"})
		}
		return routines, nil
	}
	// Newer Oracle omits no-argument procedures from ALL_ARGUMENTS, even
	// when another overload with arguments exists. Read each signature rather
	// than DISTINCT member names. 10.1 lacks OVERLOAD on ALL_PROCEDURES, but its
	// ALL_ARGUMENTS still includes no-argument placeholder rows.
	memberQuery := `SELECT procedure_name, NVL(overload, '-') FROM all_procedures
WHERE owner = :1 AND object_name = :2 AND procedure_name IS NOT NULL ORDER BY procedure_name, overload`
	members, err := s.dictionaryQuery(ctx, memberQuery, object.Owner, object.Name)
	if err != nil && strings.Contains(err.Error(), "ORA-00904") {
		members, err = s.dictionaryQuery(ctx, `SELECT DISTINCT procedure_name, '-' FROM all_procedures
WHERE owner = :1 AND object_name = :2 AND procedure_name IS NOT NULL ORDER BY procedure_name`, object.Owner, object.Name)
	}
	if err != nil {
		return nil, err
	}
	defer members.Close()
	for members.Next() {
		var name, overload string
		if err = members.Scan(&name, &overload); err != nil {
			return nil, err
		}
		found := false
		for _, r := range routines {
			if r.Member == name && (r.Overload == overload || overload == "-") {
				found = true
				break
			}
		}
		if !found {
			routines = append(routines, Routine{Object: object, Member: name, Overload: overload})
		}
	}
	return routines, members.Err()
}

func (a RoutineArgument) InputKind() (string, error) {
	if a.Depth != 0 {
		return "", errors.New("record/collection arguments are not supported")
	}
	switch a.DataType {
	case "VARCHAR2", "VARCHAR", "CHAR", "NVARCHAR2", "NCHAR", "LONG":
		return "text", nil
	case "NUMBER", "FLOAT", "BINARY_INTEGER", "PLS_INTEGER", "BINARY_FLOAT", "BINARY_DOUBLE":
		return "number", nil
	case "DATE":
		return "date", nil
	case "TIMESTAMP":
		return "timestamp", nil
	case "PL/SQL BOOLEAN", "BOOLEAN":
		return "boolean", nil
	case "RAW":
		return "raw", nil
	case "REF CURSOR":
		return "cursor", nil
	default:
		return "", fmt.Errorf("unsupported argument type: %s", a.DataType)
	}
}
func (a RoutineArgument) declaration() (string, error) {
	kind, err := a.InputKind()
	if err != nil {
		return "", err
	}
	switch kind {
	case "text":
		if a.DataType == "NCHAR" || a.DataType == "NVARCHAR2" {
			return "NVARCHAR2(16383)", nil
		}
		return "VARCHAR2(32767)", nil
	case "number":
		switch a.PLS {
		case "PLS_INTEGER", "BINARY_INTEGER", "BINARY_FLOAT", "BINARY_DOUBLE":
			return a.PLS, nil
		}
		switch a.DataType {
		case "BINARY_FLOAT", "BINARY_DOUBLE", "PLS_INTEGER", "BINARY_INTEGER":
			return a.DataType, nil
		}
		return "NUMBER", nil
	case "date":
		return "DATE", nil
	case "timestamp":
		return "TIMESTAMP(9)", nil
	case "boolean":
		return "BOOLEAN", nil
	case "raw":
		return "RAW(32767)", nil
	case "cursor":
		// For strong REF CURSOR arguments ALL_ARGUMENTS may name the returned
		// table/record rather than the declared package cursor type. A weak host
		// cursor is compatible; Oracle validates the actual returned row shape.
		return "SYS_REFCURSOR", nil
	}
	return "", errors.New("unsupported argument")
}

func routineValue(a RoutineArgument, input RoutineInput) (any, error) {
	kind, err := a.InputKind()
	if err != nil {
		return nil, err
	}
	if input.Value.Null {
		return nil, nil
	}
	if len(input.Value.Text) > 32767 {
		return nil, errors.New("argument exceeds 32767 bytes")
	}
	switch kind {
	case "boolean":
		switch strings.ToLower(strings.TrimSpace(input.Value.Text)) {
		case "true", "1":
			return "true", nil
		case "false", "0":
			return "false", nil
		default:
			return nil, errors.New("BOOLEAN: use true, false or NULL")
		}
	case "raw":
		if len(input.Value.Text) > 32766 || len(input.Value.Text)%2 != 0 || strings.IndexFunc(input.Value.Text, func(r rune) bool { return !strings.ContainsRune("0123456789abcdefABCDEF", r) }) >= 0 {
			return nil, errors.New("RAW: use hexadecimal pairs (max 16383 bytes)")
		}
		return input.Value.Text, nil
	case "cursor":
		return nil, errors.New("IN / IN OUT cursors must initially be NULL")
	default:
		return (BindValue{Kind: kind, Text: input.Value.Text}).Value()
	}
}

type routineOutput struct {
	argument RoutineArgument
	text     *sql.NullString
	cursor   *go_ora.RefCursor
	opened   *int
}
type routineCall struct {
	text    string
	args    []any
	outputs []routineOutput
}

func buildRoutineCall(r Routine, inputs map[int]RoutineInput) (routineCall, error) {
	var call routineCall
	if !r.Object.HasArguments() || r.Object.Owner == "" || r.Object.Name == "" || r.Member == "" {
		return call, errors.New("invalid routine")
	}
	if len(r.Arguments) > MaxBindVariables {
		return call, fmt.Errorf("at most %d routine arguments are supported", MaxBindVariables)
	}
	var declarations, initializers, actuals, outputs, cleanup []string
	returned := ""
	cursorCount := 0
	positions := map[int]bool{}
	for i, a := range r.Arguments {
		if positions[a.Position] {
			return call, errors.New("record/collection argument metadata is not supported")
		}
		positions[a.Position] = true
		decl, err := a.declaration()
		if err != nil {
			return call, fmt.Errorf("%s: %w", a.Name, err)
		}
		input := a.Position > 0 && (a.Direction == "IN" || a.Direction == "IN/OUT")
		output := a.Position == 0 || a.Direction == "OUT" || a.Direction == "IN/OUT"
		if !input && !output {
			return call, errors.New("unknown argument direction")
		}
		value, ok := inputs[a.Position]
		if input && !ok {
			return call, fmt.Errorf("missing input %s", a.Name)
		}
		if input && value.Default {
			if a.Direction != "IN" || !a.Defaulted {
				return call, fmt.Errorf("%s has no known IN default", a.Name)
			}
			if r.Overload != "" && r.Overload != "-" {
				return call, errors.New("supply explicit values for overloaded routines; omitting defaults can select a different signature")
			}
			continue
		}
		local := fmt.Sprintf("v%d", i)
		declarations = append(declarations, local+" "+decl+";")
		kind, _ := a.InputKind()
		if input {
			v, err := routineValue(a, value)
			if err != nil {
				return call, fmt.Errorf("%s: %w", a.Name, err)
			}
			if kind != "cursor" {
				name := fmt.Sprintf("LAZYORA_I%d", i)
				expression := ":" + name
				if kind == "boolean" {
					expression = "CASE " + expression + " WHEN 'true' THEN TRUE WHEN 'false' THEN FALSE ELSE NULL END"
				}
				if kind == "raw" {
					expression = "HEXTORAW(" + expression + ")"
				}
				initializers = append(initializers, local+" := "+expression+";")
				call.args = append(call.args, sql.Named(name, v))
			}
		}
		if a.Position == 0 {
			returned = local
		} else {
			actuals = append(actuals, quoteIdentifier(a.Name)+" => "+local)
		}
		if !output {
			continue
		}
		out := routineOutput{argument: a}
		name := fmt.Sprintf("LAZYORA_O%d", i)
		expression := local
		switch kind {
		case "number":
			expression = "TO_CHAR(" + local + ", 'TM9', 'NLS_NUMERIC_CHARACTERS=''.,''')"
		case "date":
			expression = "TO_CHAR(" + local + ", 'YYYY-MM-DD HH24:MI:SS')"
		case "timestamp":
			expression = "TO_CHAR(" + local + ", 'YYYY-MM-DD HH24:MI:SS.FF9')"
		case "boolean":
			expression = "CASE WHEN " + local + " THEN 'true' WHEN NOT " + local + " THEN 'false' ELSE NULL END"
		case "raw":
			expression = "RAWTOHEX(" + local + ")"
		case "cursor":
			cursorCount++
			if cursorCount > 4 {
				return call, errors.New("at most 4 REF CURSOR outputs are supported")
			}
			out.cursor = new(go_ora.RefCursor)
			out.opened = new(int)
			flag := fmt.Sprintf("LAZYORA_C%d", i)
			// go-ora cannot decode a NULL REF CURSOR (ORA-01001). Return an empty
			// open cursor plus a separate null flag, then close it through sql.Rows.
			adapter := fmt.Sprintf("c%d", i)
			declarations = append(declarations, adapter+" SYS_REFCURSOR;")
			expression = adapter
			outputs = append(outputs, "IF "+local+"%ISOPEN THEN :"+flag+" := 1; "+adapter+" := "+local+"; ELSE :"+flag+" := 0; OPEN "+adapter+" FOR SELECT NULL AS LAZYORA_EMPTY FROM dual WHERE 1=0; END IF;")
			call.args = append(call.args, sql.Named(flag, sql.Out{Dest: out.opened}))
			for _, cursorLocal := range []string{adapter, local} {
				cleanup = append(cleanup, "BEGIN IF "+cursorLocal+"%ISOPEN THEN CLOSE "+cursorLocal+"; END IF; EXCEPTION WHEN INVALID_CURSOR THEN NULL; END;")
			}
		}
		outputs = append(outputs, ":"+name+" := "+expression+";")
		if kind == "cursor" {
			call.args = append(call.args, sql.Named(name, sql.Out{Dest: out.cursor}))
		} else {
			out.text = new(sql.NullString)
			call.args = append(call.args, sql.Named(name, go_ora.Out{Dest: out.text, Size: 32767}))
		}
		call.outputs = append(call.outputs, out)
	}
	target := QualifiedObject(r.Object)
	if r.Object.Kind == "PACKAGE" || r.Object.Kind == "PACKAGE BODY" {
		target += "." + quoteIdentifier(r.Member)
	} else if r.Member != r.Object.Name {
		return call, errors.New("routine member does not match standalone object")
	}
	statement := target + "(" + strings.Join(actuals, ", ") + ");"
	if returned != "" {
		statement = returned + " := " + statement
	}
	// An inner exception handler closes server cursors opened before a failure.
	body := strings.Join(initializers, "\n") + "\n" + statement + "\n" + strings.Join(outputs, "\n")
	if len(cleanup) > 0 {
		body = "BEGIN\n" + body + "\nEXCEPTION WHEN OTHERS THEN\n" + strings.Join(cleanup, "\n") + "\nRAISE; END;"
	}
	if len(declarations) > 0 {
		call.text = "DECLARE\n" + strings.Join(declarations, "\n") + "\nBEGIN\n" + body + "\nEND;"
	} else {
		call.text = "BEGIN\n" + body + "\nEND;"
	}
	return call, nil
}

// ValidateRoutineInputs performs exactly the same validation as execution,
// without opening a transaction or contacting Oracle.
func ValidateRoutineInputs(r Routine, inputs map[int]RoutineInput) error {
	_, err := buildRoutineCall(r, inputs)
	return err
}

func (s *Session) RunRoutine(ctx context.Context, routine Routine, inputs map[int]RoutineInput) (result RoutineResult, err error) {
	result.Routine = QualifiedObject(routine.Object)
	if routine.Object.Kind == "PACKAGE" || routine.Object.Kind == "PACKAGE BODY" {
		result.Routine += "." + quoteIdentifier(routine.Member)
	}
	if s.finishErr != nil {
		return result, s.finishErr
	}
	call, err := buildRoutineCall(routine, inputs)
	if err != nil {
		return result, err
	}
	s.ClosePage()
	s.clearLOBs()
	if err = s.resetOutput(ctx); err != nil {
		return result, err
	}
	if s.tx == nil {
		s.tx, err = s.conn.BeginTx(context.Background(), nil)
		if err != nil {
			return result, err
		}
	}
	if s.outputEnabled {
		defer func() { result.Output = s.collectOutput() }()
	}
	started := time.Now()
	s.pending = true
	_, err = s.tx.ExecContext(ctx, call.text, call.args...)
	// Record the routine identity only. Never persist IN values or generated SQL.
	identity := QualifiedObject(routine.Object)
	if routine.Object.Kind == "PACKAGE" || routine.Object.Kind == "PACKAGE BODY" {
		identity += "." + quoteIdentifier(routine.Member)
	}
	s.recordStatement(started, "BEGIN "+identity+"(...); END;", -1, err)
	if err != nil {
		return result, err
	}
	result.Scalars = Result{Columns: []string{"Argument", "Direction", "Type", "Value"}, Affected: -1}
	// Every successfully returned cursor is initialized. Close rows (driver owns
	// the underlying cursor) exactly once, including empty/null/truncated results.
	for i, out := range call.outputs {
		if out.cursor != nil {
			named := NamedResult{Name: out.argument.Name, Null: *out.opened == 0}
			if out.argument.Position == 0 {
				named.Name = "RETURN"
			}
			named.Result, err = s.readRoutineCursor(ctx, s.tx, out.cursor)
			result.Cursors = append(result.Cursors, named)
			if err != nil {
				for _, remaining := range call.outputs[i+1:] {
					if remaining.cursor != nil {
						_ = remaining.cursor.Close()
					}
				}
				return result, err
			}
		} else {
			name := out.argument.Name
			if out.argument.Position == 0 {
				name = "RETURN"
			}
			value := Cell{Text: out.text.String, Type: "VARCHAR2", Null: !out.text.Valid}
			display := value.Text
			if value.Null {
				display = "NULL"
			}
			row := []string{name, out.argument.Direction, out.argument.DataType, display}
			result.Scalars.Rows = append(result.Scalars.Rows, row)
			result.Scalars.Values = append(result.Scalars.Values, []Cell{{Text: name, Type: "VARCHAR2"}, {Text: out.argument.Direction, Type: "VARCHAR2"}, {Text: out.argument.DataType, Type: "VARCHAR2"}, value})
		}
	}
	result.Scalars.Elapsed = time.Since(started)
	return result, nil
}
func (s *Session) readRoutineCursor(ctx context.Context, tx *sql.Tx, cursor *go_ora.RefCursor) (result Result, err error) {
	result.Affected = -1
	rows, err := go_ora.WrapRefCursor(ctx, tx, cursor)
	if err != nil {
		return result, errors.Join(err, cursor.Close())
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	result.Columns, err = rows.Columns()
	if err != nil {
		return result, err
	}
	types, err := rows.ColumnTypes()
	if err != nil {
		return result, err
	}
	stopFetch, err := s.pageFetchContext(ctx)
	if err != nil {
		return result, err
	}
	defer stopFetch()
	ctx = context.WithValue(ctx, fetchWatchKey{}, true)
	for rows.Next() {
		if len(result.Rows) == MaxRows {
			result.Truncated = true
			break
		}
		cells, scanErr := s.scanCells(ctx, rows, types, true)
		if scanErr != nil {
			return result, scanErr
		}
		row := make([]string, len(cells))
		for i, cell := range cells {
			row[i] = cell.Text
		}
		result.Rows = append(result.Rows, row)
		result.Values = append(result.Values, cells)
	}
	return result, rows.Err()
}
