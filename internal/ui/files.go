package ui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/sqlfile"
)

type fileDialog struct {
	form            *tview.Form
	list            *tview.List
	info            *tview.TextView
	directory, mode string
	focus           tview.Primitive
	apply, close    func()
}

func (s *screen) initDocument() {
	if s.document == nil {
		s.document = &sqlfile.Document{Text: s.editor.GetText()}
	}
	s.colorEditor(s.editor, s.refreshDocumentTitle)
	s.refreshDocumentTitle()
}
func (s *screen) refreshDocumentTitle() {
	title := s.tr("sql_title")
	if s.document != nil {
		name := s.tr("file_untitled")
		if s.document.Path != "" {
			name = filepath.Base(s.document.Path)
		}
		mark := ""
		if s.document.Dirty(s.editor.GetText()) {
			mark = " *"
		}
		title += "[" + tview.Escape(name) + mark + "] "
	}
	s.editor.SetTitle(title)
	s.refreshTabs()
}
func (s *screen) newDocument() {
	if len(s.tabs) > 0 {
		s.newTab()
		return
	}
	s.initDocument()
	s.guardDocument(func() {
		s.document = &sqlfile.Document{}
		s.editor.SetText("", false)
		s.refreshDocumentTitle()
		s.app.SetFocus(s.editor)
	})
}

func (s *screen) closeDocument() {
	if len(s.tabs) == 0 {
		s.newDocument()
		return
	}
	s.closeTab()
}
func (s *screen) guardDocument(next func()) {
	if s.document == nil || !s.document.Dirty(s.editor.GetText()) {
		next()
		return
	}
	if s.documentModal != nil {
		return
	}
	focus := s.app.GetFocus()
	modal := tview.NewModal().SetText(s.tr("file_unsaved")).AddButtons([]string{s.tr("save"), s.tr("file_discard"), s.tr("cancel")}).SetFocus(2)
	modal.SetBackgroundColor(palette.background)
	modal.SetTextColor(palette.text).SetButtonStyle(buttonStyle()).SetButtonActivatedStyle(buttonFocusStyle())
	styleModal(modal)
	modal.SetMouseCapture(func(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
		if action == tview.MouseLeftDoubleClick {
			action = tview.MouseLeftClick
		}
		return action, event
	})
	s.documentModal = modal
	modal.SetDoneFunc(func(index int, _ string) {
		s.pages.RemovePage("file_unsaved")
		s.documentModal = nil
		s.app.SetFocus(focus)
		switch index {
		case 0:
			s.saveDocument(false, next)
		case 1:
			next()
		}
	})
	s.pages.AddPage("file_unsaved", modal, true, true)
	s.app.SetFocus(modal)
}
func (s *screen) saveDocument(as bool, next func()) {
	s.initDocument()
	if as || s.document.Path == "" {
		s.showFileDialog("save", next)
		return
	}
	s.writeDocument(s.document.Path, s.document.Snapshot, next)
}
func (s *screen) fileError(err error) {
	message := fmt.Sprintf(s.tr("file_error"), err)
	if s.fileDialog != nil {
		s.fileDialog.info.SetText(message)
	}
	s.setStatus("file_error", err)
}
func (s *screen) writeDocument(path string, expected sqlfile.Snapshot, next func()) {
	text := s.editor.GetText()
	if path == s.document.Path && expected == s.document.Snapshot && !s.document.Dirty(text) {
		current, err := sqlfile.Inspect(path)
		if err == nil && current == expected && current.Exists {
			s.finishDocumentSave(s.document, next)
			return
		}
	}
	doc, err := sqlfile.Save(path, text, s.document.Format, expected)
	if errors.Is(err, sqlfile.ErrChanged) {
		current, inspectErr := sqlfile.Inspect(path)
		if inspectErr != nil {
			s.fileError(inspectErr)
			return
		}
		s.showProgramConfirm(fmt.Sprintf(s.tr("file_external_confirm"), tview.Escape(path)), func() { s.writeDocument(path, current, next) })
		return
	}
	if err != nil {
		s.fileError(err)
		return
	}
	s.finishDocumentSave(doc, next)
}

func (s *screen) finishDocumentSave(doc *sqlfile.Document, next func()) {
	s.document = doc
	s.refreshDocumentTitle()
	s.setStatus("file_saved", doc.Path)
	if s.fileDialog != nil {
		s.fileDialog.close()
	}
	if next != nil {
		next()
	}
}

func (s *screen) openDocument(path string) {
	doc, err := sqlfile.Read(path)
	if err != nil {
		s.fileError(err)
		return
	}
	if s.fileDialog != nil {
		s.fileDialog.close()
	}
	s.document = doc
	s.editor.SetText(doc.Text, false)
	s.refreshDocumentTitle()
	s.app.SetFocus(s.editor)
	s.setStatus("file_opened", path)
}
func (s *screen) showFileDialog(mode string, next func()) {
	if s.fileDialog != nil || s.busy {
		return
	}
	s.initDocument()
	directory, err := os.Getwd()
	if err != nil {
		s.fileError(err)
		return
	}
	if s.document.Path != "" {
		directory = filepath.Dir(s.document.Path)
	}
	path := directory + string(os.PathSeparator)
	if mode == "save" {
		path = filepath.Join(directory, "query.sql")
		if s.document.Path != "" {
			path = s.document.Path
		}
	}
	dialog := &fileDialog{directory: directory, mode: mode, focus: s.app.GetFocus()}
	s.fileDialog = dialog
	dialog.form = tview.NewForm().SetItemPadding(0).AddInputField(s.tr("file_path"), path, 0, nil, nil)
	styleForm(dialog.form)
	dialog.list = tview.NewList().ShowSecondaryText(false).SetSelectedStyle(focusStyle())
	dialog.list.SetBackgroundColor(palette.background)
	dialog.list.SetMainTextColor(palette.text)
	dialog.list.SetBorder(true)
	styleBox(dialog.list.Box)
	dialog.info = tview.NewTextView().SetWrap(true).SetText(s.tr("file_picker_hint"))
	dialog.info.SetBackgroundColor(palette.background)
	dialog.info.SetTextColor(palette.text)
	dialog.close = func() { s.pages.RemovePage("file"); s.fileDialog = nil; s.app.SetFocus(dialog.focus) }
	pathField := dialog.form.GetFormItem(0).(*tview.InputField)
	var browse func(string)
	browse = func(dir string) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			s.fileError(err)
			return
		}
		dialog.directory = dir
		pathField.SetText(dir + string(os.PathSeparator))
		dialog.list.Clear().SetTitle(tview.Escape(dir))
		parent := filepath.Dir(dir)
		if parent != dir {
			dialog.list.AddItem("../", "", 0, func() { browse(parent) })
		}
		sort.SliceStable(entries, func(i, j int) bool {
			if entries[i].IsDir() != entries[j].IsDir() {
				return entries[i].IsDir()
			}
			return entries[i].Name() < entries[j].Name()
		})
		for _, entry := range entries {
			if entry.Type()&os.ModeSymlink != 0 {
				continue
			}
			name := entry.Name()
			destination := filepath.Join(dir, name)
			if entry.IsDir() {
				dialog.list.AddItem(tview.Escape(name)+"/", "", 0, func() { browse(destination) })
			} else if entry.Type().IsRegular() {
				dialog.list.AddItem(tview.Escape(name), "", 0, func() { pathField.SetText(destination); s.app.SetFocus(dialog.form) })
			}
		}
		dialog.info.SetText(s.tr("file_picker_hint"))
	}
	dialog.apply = func() {
		path, err := sqlfile.Resolve(pathField.GetText(), dialog.directory)
		if err != nil {
			s.fileError(err)
			return
		}
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			browse(path)
			return
		}
		if mode == "open" {
			// Save As may need its own picker. Retain the target path while
			// closing this picker before the unsaved-document prompt.
			if s.document.Dirty(s.editor.GetText()) {
				dialog.close()
			}
			s.guardDocument(func() { s.openDocument(path) })
			return
		}
		expected, err := sqlfile.Inspect(path)
		if err != nil {
			s.fileError(err)
			return
		}
		write := func() { s.writeDocument(path, expected, next) }
		if expected.Exists {
			s.showProgramConfirm(fmt.Sprintf(s.tr("file_overwrite_confirm"), tview.Escape(path)), write)
		} else {
			write()
		}
	}
	apply := clickable(tview.NewButton(s.tr("file_" + mode)).SetSelectedFunc(dialog.apply))
	cancel := clickable(tview.NewButton(s.tr("cancel")).SetSelectedFunc(dialog.close))
	row := tview.NewFlex().AddItem(apply, 0, 1, false).AddItem(cancel, 0, 1, false)
	dialog.form.SetBorder(true)
	content := tview.NewFlex().SetDirection(tview.FlexRow).AddItem(dialog.form, 5, 0, true).AddItem(dialog.list, 0, 1, false).AddItem(dialog.info, 3, 0, false).AddItem(row, 1, 0, false)
	root := s.formPopup(content, 96, 30, nil)
	browse(directory)
	pathField.SetText(path)
	s.pages.AddPage("file", root, true, true)
	s.app.SetFocus(dialog.form)
}
func (s *screen) fileKey(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyCtrlC, tcell.KeyCtrlQ:
		return nil
	case tcell.KeyF1:
		s.showHelp()
		return nil
	case tcell.KeyEscape:
		s.fileDialog.close()
		return nil
	case tcell.KeyTab, tcell.KeyBacktab:
		s.dialogTab(event, s.fileDialog.form, s.fileDialog.list)
		return nil
	case tcell.KeyEnter:
		if event.Modifiers()&(tcell.ModCtrl|tcell.ModAlt) != 0 || s.fileDialog.form.HasFocus() {
			s.fileDialog.apply()
			return nil
		}
	case tcell.KeyCtrlS:
		if s.fileDialog.mode == "save" {
			s.fileDialog.apply()
			return nil
		}
	}
	return event
}
