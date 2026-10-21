package ui

import (
	"encoding/csv"
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/oracle"
)

func (s *screen) showDefinition(section string) {
	if s.busy {
		return
	}
	page, _ := s.pages.GetFrontPage()
	if page != "object" && page != "definition" {
		return
	}
	if page != "definition" {
		s.definitionObject = s.object
		s.definitionCopyDetail = false
		s.definitionCache = make(map[string]oracle.ObjectDefinition)
		s.definitionInfo = tview.NewTextView().SetWrap(true)
		s.definitionInfo.SetBackgroundColor(palette.background)
		s.definitionInfo.SetTextColor(palette.text)
		s.definitionRoot = tview.NewFlex().SetDirection(tview.FlexRow)
		s.pages.AddPage("definition", s.definitionRoot, true, true)
	}
	s.definitionSection = section
	if data, ok := s.definitionCache[section]; ok {
		s.renderDefinition(data, nil)
		return
	}
	s.renderDefinition(oracle.ObjectDefinition{}, nil)
	s.definitionInfo.SetText(oracle.QualifiedObject(s.definitionObject) + " · " + s.tr(section) + "\n" + s.tr("browser_loading"))
	if s.demo && !s.connected {
		data := oracle.ObjectDefinition{}
		if section == "ddl" {
			data.DDL = "CREATE TABLE " + oracle.QualifiedObject(s.definitionObject) + " (\"ID\" NUMBER);"
		}
		s.acceptDefinition(s.definitionObject, section, data, nil)
		return
	}
	s.submit(job{action: "definition", object: s.definitionObject, text: section})
}
func (s *screen) closeDefinition() {
	if s.busy {
		s.action("cancel")
		return
	}
	s.pages.RemovePage("definition")
	s.definitionCache = nil
	s.definitionRoot = nil
	s.definitionInfo, s.definitionText, s.definitionTable = nil, nil, nil
	s.definitionRows, s.definitionDetails = nil, nil
	s.definitionDDL = ""
	if s.objectTable != nil {
		s.app.SetFocus(s.objectTable)
	}
}
func (s *screen) acceptDefinition(object oracle.Object, section string, data oracle.ObjectDefinition, err error) {
	if s.definitionRoot == nil || object != s.definitionObject {
		return
	}
	if err == nil {
		s.definitionCache[section] = data
		s.setStatus("definition_loaded", s.tr(section))
	}
	if section == s.definitionSection {
		s.renderDefinition(data, err)
	}
}
func quoteColumns(columns []string) string {
	quoted := make([]string, len(columns))
	for i, name := range columns {
		quoted[i] = "\"" + strings.ReplaceAll(name, "\"", "\"\"") + "\""
	}
	return strings.Join(quoted, ", ")
}
func (s *screen) renderDefinition(data oracle.ObjectDefinition, err error) {
	s.definitionDDL = data.DDL
	message := oracle.QualifiedObject(s.definitionObject) + " · " + s.tr(s.definitionSection) + "\n" + s.tr("definition_note")
	if data.Limited {
		message += "\n" + s.tr("definition_limited")
	}
	if err != nil {
		message += "\n" + s.tr("definition_error") + ": " + err.Error()
	}
	s.definitionInfo.SetText(message)
	if s.definitionText == nil {
		s.definitionText = tview.NewTextView().SetWrap(false).SetScrollable(true)
	}
	s.definitionText.SetFocusFunc(func() { s.definitionCopyDetail = true })
	s.definitionText.SetText("").ScrollToBeginning()
	s.definitionText.SetBorder(true).SetTitle(s.tr("object_details"))
	styleBox(s.definitionText.Box)
	s.definitionText.SetTextColor(palette.text)
	if s.definitionTable == nil {
		s.definitionTable = tview.NewTable().SetFixed(1, 0).SetSelectable(true, false).SetSelectedStyle(focusStyle())
	}
	s.definitionTable.SetFocusFunc(func() { s.definitionCopyDetail = false })
	s.definitionTable.Clear()
	s.definitionTable.SetBorder(true).SetTitle(s.tr(s.definitionSection))
	styleBox(s.definitionTable.Box)
	s.definitionRows, s.definitionDetails = nil, nil
	var headers []string
	if s.definitionSection == "constraints" {
		headers = []string{"object_name", "object_type", "object_status", "definition_columns", "definition_reference", "definition_validated"}
		for _, c := range data.Constraints {
			reference := c.ReferenceOwner + "." + c.ReferenceName
			if c.ReferenceTable != "" {
				reference = oracle.QualifiedObject(oracle.Object{Owner: c.ReferenceOwner, Name: c.ReferenceTable}) + " (" + quoteColumns(c.ReferenceColumns) + ")"
			}
			if c.ReferenceOwner == "" {
				reference = ""
			}
			s.definitionRows = append(s.definitionRows, []string{c.Name, c.Kind, c.Status, quoteColumns(c.Columns), reference, c.Validated})
			detail := fmt.Sprintf("%s: %s\n%s: %s\n%s: %s\n%s: %s\n%s: %s / %s\n%s: %s\n%s:\n%s", s.tr("object_name"), c.Name, s.tr("definition_columns"), quoteColumns(c.Columns), s.tr("definition_reference"), reference, s.tr("definition_delete_rule"), c.DeleteRule, s.tr("definition_deferred"), c.Deferrable, c.Deferred, s.tr("definition_validated"), c.Validated, s.tr("definition_condition"), c.Condition)
			s.definitionDetails = append(s.definitionDetails, detail)
		}
	} else if s.definitionSection == "indexes" {
		headers = []string{"owner", "object_name", "object_type", "definition_unique", "object_status", "definition_columns"}
		for _, index := range data.Indexes {
			var cols, expressions []string
			for _, col := range index.Columns {
				label := quoteColumns([]string{col.Name}) + " " + col.Direction
				if col.Expression != "" {
					label = col.Expression + " " + col.Direction
					expressions = append(expressions, fmt.Sprintf("%d: %s", col.Position, col.Expression))
				}
				cols = append(cols, label)
			}
			s.definitionRows = append(s.definitionRows, []string{index.Owner, index.Name, index.Kind, index.Uniqueness, index.Status, strings.Join(cols, ", ")})
			s.definitionDetails = append(s.definitionDetails, oracle.QualifiedObject(oracle.Object{Owner: index.TableOwner, Name: index.TableName})+"\n"+strings.Join(cols, ", ")+"\n\n"+strings.Join(expressions, "\n"))
		}
	}
	for col, label := range headers {
		s.definitionTable.SetCell(0, col, tview.NewTableCell(s.tr(label)).SetSelectable(false).SetStyle(tcell.StyleDefault.Foreground(palette.accent).Background(palette.surface).Bold(true)))
	}
	for row, values := range s.definitionRows {
		for col, value := range values {
			s.definitionTable.SetCell(row+1, col, tview.NewTableCell(tview.Escape(value)).SetTextColor(palette.text))
		}
	}
	s.definitionTable.SetSelectionChangedFunc(func(row, col int) {
		if row > 0 && row <= len(s.definitionDetails) {
			s.definitionText.SetText(s.definitionDetails[row-1]).ScrollToBeginning()
		}
	})
	var selectors, buttons *tview.Flex
	if s.definitionRoot.GetItemCount() > 1 {
		selectors = s.definitionRoot.GetItem(1).(*tview.Flex)
		buttons = s.definitionRoot.GetItem(s.definitionRoot.GetItemCount() - 1).(*tview.Flex)
	}
	s.definitionRoot.Clear().AddItem(s.definitionInfo, 5, 0, false)
	if selectors == nil {
		selectors = tview.NewFlex()
		for i, section := range []string{"ddl", "constraints", "indexes"} {
			selectors.AddItem(clickable(tview.NewButton(fmt.Sprintf("%d %s", i+1, s.tr(section))).SetSelectedFunc(func() { s.showDefinition(section) })), 0, 1, false)
		}
	}
	s.definitionRoot.AddItem(selectors, 1, 0, false)
	if s.definitionSection == "ddl" {
		s.definitionText.SetText(data.DDL)
		s.definitionRoot.AddItem(s.definitionText, 0, 1, true)
	} else {
		s.definitionRoot.AddItem(s.definitionTable, 0, 2, true).AddItem(s.definitionText, 0, 1, false)
		if len(s.definitionRows) > 0 {
			s.definitionTable.Select(1, 0)
		}
	}
	if data.DDL == "" && len(s.definitionRows) == 0 {
		if err != nil {
			s.definitionText.SetText(err.Error())
		} else {
			s.definitionText.SetText(s.tr("definition_empty"))
		}
	}
	if buttons == nil {
		buttons = tview.NewFlex().AddItem(clickable(tview.NewButton(s.tr("copy_selection")+" (Ctrl+C)").SetSelectedFunc(s.copyDefinition)), 0, 1, false).
			AddItem(clickable(tview.NewButton(s.tr("reload")+" (Ctrl+R)").SetSelectedFunc(s.refreshDefinition)), 0, 1, false).
			AddItem(clickable(tview.NewButton(s.tr("close")+" (Esc)").SetSelectedFunc(s.closeDefinition)), 0, 1, false)
	}
	s.definitionRoot.AddItem(buttons, 1, 0, false)
	if page, _ := s.pages.GetFrontPage(); page == "definition" {
		if s.definitionSection == "ddl" {
			s.app.SetFocus(s.definitionText)
		} else {
			s.app.SetFocus(s.definitionTable)
		}
	}
}
func (s *screen) refreshDefinition() {
	if s.busy {
		return
	}
	delete(s.definitionCache, s.definitionSection)
	s.showDefinition(s.definitionSection)
}
func (s *screen) copyDefinition() {
	if s.definitionSection == "ddl" || s.definitionCopyDetail || s.definitionText.HasFocus() {
		if s.definitionSection == "ddl" {
			s.sendClipboard(s.definitionDDL)
		} else {
			s.sendClipboard(s.definitionText.GetText(false))
		}
		return
	}
	row, _ := s.definitionTable.GetSelection()
	if row > 0 && row <= len(s.definitionRows) {
		var out strings.Builder
		w := csv.NewWriter(&out)
		w.Comma = '\t'
		_ = w.Write(s.definitionRows[row-1])
		w.Flush()
		s.sendClipboard(strings.TrimSuffix(out.String(), "\n"))
	}
}
func (s *screen) definitionKey(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyF1:
		s.showHelp()
		return nil
	case tcell.KeyEscape:
		s.closeDefinition()
		return nil
	case tcell.KeyCtrlC:
		s.copyDefinition()
		return nil
	case tcell.KeyCtrlR:
		s.refreshDefinition()
		return nil
	case tcell.KeyTab, tcell.KeyBacktab:
		content := []tview.Primitive{s.definitionText}
		if s.definitionSection != "ddl" {
			content = []tview.Primitive{s.definitionTable, s.definitionText}
		}
		selectors := s.definitionRoot.GetItem(1).(*tview.Flex)
		for i := 0; i < selectors.GetItemCount(); i++ {
			content = append(content, selectors.GetItem(i))
		}
		s.dialogTab(event, content...)
		return nil
	case tcell.KeyRune:
		if event.Modifiers() == 0 && event.Rune() >= '1' && event.Rune() <= '3' {
			s.showDefinition([]string{"ddl", "constraints", "indexes"}[event.Rune()-'1'])
			return nil
		}
	}
	return event
}
