package oracle

import (
	"context"
	"github.com/DATA-DOG/go-sqlmock"
	"testing"
)

func TestDescribeBindsQuotedIdentityWithoutStartingTransaction(t *testing.T) {
	s, mock := mockSession(t)
	mock.ExpectQuery("SELECT owner, object_name, object_type, status").WithArgs(`Mixed'Owner`, `T"name`).WillReturnRows(sqlmock.NewRows([]string{"OWNER", "NAME", "KIND", "STATUS"}).AddRow(`Mixed'Owner`, `T"name`, "VIEW", "VALID")).RowsWillBeClosed()
	object, err := s.FindObject(context.Background(), `"Mixed'Owner"."T""name"`)
	if err != nil || object.Kind != "VIEW" || object.Name != `T"name` {
		t.Fatalf("object=%+v err=%v", object, err)
	}
	if s.tx != nil || s.PendingChanges() {
		t.Fatal("describe changed transaction state")
	}
	for _, bad := range []string{"", "A.B.C", "T; DROP TABLE T", "A@DB"} {
		if _, _, err := ParseObjectReference(bad); err == nil {
			t.Fatalf("accepted %q", bad)
		}
	}
	owner, name, err := ParseObjectReference("my_schema.some_table")
	if err != nil || owner != "MY_SCHEMA" || name != "SOME_TABLE" {
		t.Fatal("unquoted identifiers not normalized")
	}
}
