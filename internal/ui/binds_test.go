package ui

import (
	"testing"

	"github.com/leechis7/tuivory/go"
)

func TestBindPromptCancelAndSubmitSnapshot(t *testing.T) {
	s := fileTestScreen(t)
	s.editor.SetText("SELECT :name, :NAME FROM dual;", false)
	s.executeSQL()
	if s.bindForm == nil || s.bindForm.GetFormItemCount() != 3 || len(s.jobs) != 0 {
		t.Fatal("did not prompt once per name")
	}
	s.bindCancel()
	if len(s.jobs) != 0 || len(s.sqlHistory) != 0 || s.bindValues != nil {
		t.Fatal("cancel submitted SQL or saved values")
	}
	s.executeSQL()
	s.bindForm.GetFormItem(1).(*tuivory.InputField).SetText("private value")
	s.editor.SetText("DROP TABLE other;", false)
	s.key(tuivory.NewEventKey(tuivory.KeyEnter, 0, tuivory.ModCtrl))
	j := <-s.jobs
	if j.text != "SELECT :name, :NAME FROM dual;" || j.bindings["NAME"].Text != "private value" {
		t.Fatal("bind input lost submitted snapshot")
	}
	if s.sqlHistory[0] != j.text {
		t.Fatal("history did not retain placeholders")
	}
	s.busy = false
	s.editor.SetText(j.text, false)
	s.executeSQL()
	if s.bindForm.GetFormItem(1).(*tuivory.InputField).GetText() != "private value" {
		t.Fatal("same tab/SQL did not reuse value")
	}
	s.bindCancel()
	s.clearBindValues()
	if s.bindValues != nil || s.bindSQL != "" {
		t.Fatal("clear retained values")
	}
}

func TestBindInvalidTypeAndNull(t *testing.T) {
	s := fileTestScreen(t)
	s.editor.SetText("SELECT :n FROM dual;", false)
	s.executeSQL()
	s.bindForm.GetFormItem(0).(*choiceField).SetCurrentOption(1)
	s.bindForm.GetFormItem(1).(*tuivory.InputField).SetText("invalid")
	s.bindSubmit()
	if len(s.jobs) != 0 || s.bindForm == nil {
		t.Fatal("invalid value submitted")
	}
	s.bindForm.GetFormItem(2).(*tuivory.Checkbox).SetChecked(true)
	s.bindSubmit()
	if j := <-s.jobs; !j.bindings["N"].Null {
		t.Fatal("NULL value not submitted")
	}
}
