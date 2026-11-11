package oracle

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestActualPlanPermissionErrorKeepsSuccessfulStatement(t *testing.T) {
	s, mock := mockSession(t)
	mock.ExpectBegin()
	mock.ExpectExec(`^/\* lazyora-plan-[a-f0-9]{24} \*/ UPDATE items SET n = 1$`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("FROM v\\$sql WHERE").WithArgs(sqlmock.AnyArg()).WillReturnError(errors.New("ORA-00942: table or view does not exist"))
	result, err := s.ExecuteWithActualPlan(context.Background(), "UPDATE items SET n = 1", nil)
	if err != nil || result.Affected != 1 || result.ActualPlan == nil || !errors.Is(result.ActualPlan.Err, ErrActualPlanPrivilege) {
		t.Fatalf("execution and plan outcomes were not separated: %+v, %v", result, err)
	}
	pending := s.PendingWork().Statements
	if !s.PendingChanges() || len(pending) != 1 || pending[0].SQL != "UPDATE items SET n = 1" {
		t.Fatalf("pending statement changed: %+v", pending)
	}
	mock.ExpectRollback()
	if err := s.Finish(false); err != nil {
		t.Fatal(err)
	}
}
