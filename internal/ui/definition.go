package ui

import (
	"encoding/csv"
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/oracle"
)

var definitionSections = []string{"ddl", "constraints", "indexes", "grants", "dependencies", "dependents"}

type definitionLocation struct {
	object  oracle.Object
	section string
	cache   map[string]oracle.ObjectDefinition
	row     int
}

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
		s.definitionTrail = nil
		s.definitionSelectors = nil
		s.definitionCache = make(map[string]oracle.ObjectDefinition)
		s.definitionInfo = s.newTextView().SetWrap(true)
		s.definitionInfo.SetBackgroundColor(s.colors().background)
		s.definitionInfo.SetTextColor(s.colors().text)
		s.definitionRoot = s.newFlex().SetDirection(tview.FlexRow)
		s.pages.AddPage("definition", s.definitionRoot, true, true)
	}
	s.definitionSection = section
	if data, ok := s.definitionCache[section]; ok {
		s.renderDefinition(data, nil)
		return
	}
	s.renderDefinition(oracle.ObjectDefinition{}, nil)
	s.definitionInfo.SetText(oracle.QualifiedObject(s.definitionObject) + " · " + s.tr(section) + "\n" + s.tr("browser_loading"))
	s.definitionText.SetText(s.tr("browser_loading"))
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
	s.definitionTargets, s.definitionTrail, s.definitionSelectors = nil, nil, nil
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
	message := oracle.QualifiedObject(s.definitionObject) + " · " + s.definitionObject.Kind + " · " + s.definitionObject.Status + " · " + s.tr(s.definitionSection) + "\n" + s.tr("definition_note")
	if s.definitionSection == "grants" || s.definitionSection == "dependencies" || s.definitionSection == "dependents" {
		message += "\n" + s.tr("relations_scope")
	}
	if data.Limited {
		message += "\n" + s.tr("definition_limited")
	}
	if err != nil {
		message += "\n" + s.tr("definition_error") + ": " + err.Error()
	}
	s.definitionInfo.SetText(message)
	if s.definitionText == nil {
		s.definitionText = s.newTextView().SetWrap(false).SetScrollable(true)
	}
	s.definitionText.SetFocusFunc(func() { s.definitionCopyDetail = true })
	s.definitionText.SetText("").ScrollToBeginning()
	s.definitionText.SetBorder(true).SetTitle(s.tr("object_details"))
	s.styleBox(s.definitionText.Box)
	s.definitionText.SetTextColor(s.colors().text)
	if s.definitionTable == nil {
		s.definitionTable = tview.NewTable().SetFixed(1, 0).SetSelectable(true, false).SetSelectedStyle(s.focusStyle())
	}
	s.definitionTable.SetFocusFunc(func() { s.definitionCopyDetail = false })
	s.definitionTable.Clear()
	s.definitionTable.SetBorder(true).SetTitle(s.tr(s.definitionSection))
	s.styleBox(s.definitionTable.Box)
	s.definitionRows, s.definitionDetails = nil, nil
	s.definitionTargets = nil
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
	} else if s.definitionSection == "grants" {
		headers = []string{"grant_grantee", "grant_privilege", "definition_columns", "grant_grantable", "grant_grantor"}
		for _, grant := range data.Grants {
			s.definitionRows = append(s.definitionRows, []string{grant.Grantee, grant.Privilege, grant.Column, grant.Grantable, grant.Grantor})
			s.definitionDetails = append(s.definitionDetails, fmt.Sprintf("%s: %s\n%s: %s\n%s: %s\n%s: %s\n%s: %s", s.tr("grant_grantee"), grant.Grantee, s.tr("grant_privilege"), grant.Privilege, s.tr("definition_columns"), grant.Column, s.tr("grant_grantable"), grant.Grantable, s.tr("grant_grantor"), grant.Grantor))
		}
	} else if s.definitionSection == "dependencies" || s.definitionSection == "dependents" {
		headers = []string{"owner", "object_name", "object_type", "object_status", "relation_link", "relation_kind"}
		for _, relation := range data.Relations {
			status := relation.Object.Status
			if !relation.Visible {
				status = s.tr("relation_unknown")
			}
			s.definitionRows = append(s.definitionRows, []string{relation.Object.Owner, relation.Object.Name, relation.Object.Kind, status, relation.Link, relation.Dependency})
			detail := oracle.QualifiedObject(relation.Object) + "\n" + relation.Object.Kind + " · " + status + "\n" + s.tr("relation_kind") + ": " + relation.Dependency
			if relation.Navigable() {
				detail += "\n" + s.tr("relation_open_hint")
			} else {
				detail += "\n" + s.tr("relation_unavailable")
			}
			if relation.Link != "" {
				detail += "\n" + s.tr("relation_link") + ": " + relation.Link
			}
			s.definitionDetails = append(s.definitionDetails, detail)
			s.definitionTargets = append(s.definitionTargets, relation)
		}
	}
	for col, label := range headers {
		s.definitionTable.SetCell(0, col, tview.NewTableCell(s.tr(label)).SetSelectable(false).SetStyle(tcell.StyleDefault.Foreground(s.colors().accent).Background(s.colors().surface).Bold(true)))
	}
	for row, values := range s.definitionRows {
		for col, value := range values {
			s.definitionTable.SetCell(row+1, col, tview.NewTableCell(tview.Escape(value)).SetTextColor(s.colors().text))
		}
	}
	s.definitionTable.SetSelectionChangedFunc(func(row, col int) {
		if row > 0 && row <= len(s.definitionDetails) {
			s.definitionText.SetText(s.definitionDetails[row-1]).ScrollToBeginning()
		}
	})
	s.definitionTable.SetSelectedFunc(func(_, _ int) { s.openDefinitionRelation() })
	var selectors, buttons *tview.Flex
	if s.definitionRoot.GetItemCount() > 1 {
		selectors = s.definitionRoot.GetItem(1).(*tview.Flex)
		buttons = s.definitionRoot.GetItem(s.definitionRoot.GetItemCount() - 1).(*tview.Flex)
	}
	s.definitionRoot.Clear().AddItem(s.definitionInfo, 5, 0, false)
	if selectors == nil {
		selectors = s.newFlex().SetDirection(tview.FlexRow)
		for from := 0; from < len(definitionSections); from += 3 {
			row := s.newFlex()
			for i := from; i < from+3; i++ {
				section := definitionSections[i]
				button := s.clickable(tview.NewButton(fmt.Sprintf("%d %s", i+1, s.tr(section))).SetSelectedFunc(func() { s.showDefinition(section) }))
				s.definitionSelectors = append(s.definitionSelectors, button)
				row.AddItem(button, 0, 1, false)
			}
			selectors.AddItem(row, 1, 0, false)
		}
	}
	s.definitionRoot.AddItem(selectors, 2, 0, false)
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
		buttons = s.newFlex().AddItem(s.clickable(tview.NewButton(s.tr("copy_selection")+" (Ctrl+C)").SetSelectedFunc(s.copyDefinition)), 0, 1, false).
			AddItem(s.clickable(tview.NewButton(s.tr("reload")+" (Ctrl+R)").SetSelectedFunc(s.refreshDefinition)), 0, 1, false).
			AddItem(s.clickable(tview.NewButton(s.tr("relation_open")+" (Alt+O)").SetSelectedFunc(s.openDefinitionRelation)), 0, 1, false).
			AddItem(s.clickable(tview.NewButton(s.tr("relation_back")+" (Alt+Left)").SetSelectedFunc(s.backDefinitionRelation)), 0, 1, false).
			AddItem(s.clickable(tview.NewButton(s.tr("close")+" (Esc)").SetSelectedFunc(s.closeDefinition)), 0, 1, false)
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
	if event.Modifiers()&tcell.ModAlt != 0 {
		if event.Key() == tcell.KeyLeft {
			s.backDefinitionRelation()
			return nil
		}
		if event.Key() == tcell.KeyRune && (event.Rune() == 'o' || event.Rune() == 'O') {
			s.openDefinitionRelation()
			return nil
		}
	}
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
		content = append(content, s.definitionSelectors...)
		s.dialogTab(event, content...)
		return nil
	case tcell.KeyRune:
		if event.Modifiers() == 0 && event.Rune() >= '1' && event.Rune() <= '6' {
			s.showDefinition(definitionSections[event.Rune()-'1'])
			return nil
		}
	}
	return event
}

func (s *screen) openDefinitionRelation() {
	if s.busy {
		return
	}
	row, _ := s.definitionTable.GetSelection()
	if row < 1 || row > len(s.definitionTargets) {
		return
	}
	target := s.definitionTargets[row-1]
	if !target.Navigable() || len(s.definitionTrail) >= 32 {
		s.definitionInfo.SetText(s.definitionInfo.GetText(false) + "\n" + s.tr("relation_unavailable"))
		return
	}
	s.definitionTrail = append(s.definitionTrail, definitionLocation{s.definitionObject, s.definitionSection, s.definitionCache, row})
	s.definitionObject, s.definitionCache = target.Object, make(map[string]oracle.ObjectDefinition)
	s.definitionCopyDetail = false
	s.showDefinition("dependencies")
}

func (s *screen) backDefinitionRelation() {
	if s.busy || len(s.definitionTrail) == 0 {
		return
	}
	last := len(s.definitionTrail) - 1
	location := s.definitionTrail[last]
	s.definitionTrail = s.definitionTrail[:last]
	s.definitionObject, s.definitionCache = location.object, location.cache
	s.definitionCopyDetail = false
	s.showDefinition(location.section)
	s.definitionTable.Select(location.row, 0)
}
