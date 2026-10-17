package ui

import (
	"context"
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/oracle"
	"lazyora/internal/settings"
)

type job struct {
	ctx    context.Context
	action string
	config oracle.Config
	text   string
	object oracle.Object
	node   *tview.TreeNode
}

type screen struct {
	rootLayout, menuLayout                   *tview.Flex
	chromeHidden                             bool
	splitOffsets                             panelOffsets
	menuButtons                              []*tview.Button
	menuList                                 *tview.List
	menuFocus                                tview.Primitive
	menuIndex                                int
	describeForm                             *tview.Form
	describeFocus                            tview.Primitive
	describeAction                           func()
	app                                      *tview.Application
	terminal                                 tcell.Screen
	resultRows                               [][]string
	resultHeaders                            []string
	resultSelection                          resultSelection
	clipboardText                            string
	copyFocus                                tview.Primitive
	copyText                                 *tview.TextView
	maximized                                string
	fullscreen                               bool
	pages                                    *tview.Pages
	editor                                   *tview.TextArea
	table                                    *tview.Table
	status                                   *tview.TextView
	footer                                   *tview.TextView
	footerVersion                            *tview.TextView
	footerBar                                *tview.Flex
	buttons                                  []*tview.Button
	footerKeys                               []footerKey
	resultsFocused                           bool
	limitedColors                            bool
	browser                                  *tview.TreeView
	browserHidden                            bool
	mainLayout, queryLayout                  *tview.Flex
	browserOwner                             string
	object                                   oracle.Object
	objectInfo                               *tview.TextView
	objectFocus                              tview.Primitive
	objectTable                              *tview.Table
	objectColumns                            []oracle.Column
	pendingChanges                           bool
	exitDialog                               *tview.Modal
	exitFocus                                tview.Primitive
	sourceEditor                             *tview.TextArea
	sourceInfo                               *tview.TextView
	sourceDiagnostics                        *tview.Table
	sourceBase                               string
	sourceLoaded                             bool
	programModal                             *tview.Modal
	programFocus, argumentFocus              tview.Primitive
	argumentTable                            *tview.Table
	argumentInfo                             *tview.TextView
	jobs                                     chan job
	done                                     chan struct{}
	busy, quitting, demo, connected          bool
	cancel                                   context.CancelFunc
	helpFocus, connectFocus, prefsFocus      tview.Primitive
	connectForm, connectProfiles, prefsForm  *tview.Form
	connectAction, saveConnection, savePrefs func()
	connectionOptions                        func()
	prefs                                    settings.Config
	prefsPath, statusKey                     string
	statusArgs                               []any
}

func Run(demo bool) error { return run(demo, nil, nil) }

func run(demo bool, terminal tcell.Screen, ready func(*screen)) error {
	if terminal == nil {
		var err error
		terminal, err = newCompatibleScreen()
		if err != nil {
			return err
		}
	}
	prefs := settings.Default()
	path, prefsErr := settings.Path()
	if prefsErr == nil {
		prefs, prefsErr = settings.Load(path)
	}
	s := &screen{app: tview.NewApplication(), jobs: make(chan job, 1), done: make(chan struct{}), demo: demo, prefs: prefs, prefsPath: path}
	s.terminal = terminal
	s.app.SetScreen(terminal)
	s.limitedColors = terminal.Colors() > 0 && terminal.Colors() <= 16
	s.editor = tview.NewTextArea().SetText("SELECT sysdate AS server_time FROM dual;", false)
	s.editor.SetBorder(true)
	s.table = tview.NewTable().SetBorders(false).SetSelectable(true, true).SetFixed(1, 0)
	s.table.SetBorder(true)
	s.table.SetMouseCapture(s.resultMouse)
	s.editor.SetClipboard(s.sendClipboard, func() string { return s.clipboardText })
	s.status = tview.NewTextView()
	s.queryLayout = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(s.editor, 0, 2, true).
		AddItem(s.table, 0, 3, false).AddItem(s.status, 2, 0, false)
	s.mainLayout = tview.NewFlex().AddItem(s.newBrowser(), 0, 1, false).AddItem(s.queryLayout, 0, 3, true)
	s.pages = tview.NewPages().AddPage("main", s.mainLayout, true, true)
	s.pages.SetMouseCapture(func(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
		modal := s.exitDialog
		if s.programModal != nil {
			modal = s.programModal
		}
		if modal != nil {
			// Modal only covers its centered box; consume the rest of the page
			// too, so clicks cannot edit SQL or open dialogs behind it.
			if !s.busy {
				modal.MouseHandler()(action, event, func(p tview.Primitive) { s.app.SetFocus(p) })
			}
			return tview.MouseConsumed, nil
		}
		if s.menuList != nil {
			x, y := event.Position()
			if s.menuList.InRect(x, y) {
				s.menuList.MouseHandler()(action, event, func(p tview.Primitive) { s.app.SetFocus(p) })
			} else if action == tview.MouseLeftClick {
				s.closeMenu()
			}
			return tview.MouseConsumed, nil
		}
		return action, event
	})
	s.footer = tview.NewTextView().SetWrap(false)
	s.styleScreen()
	s.menuLayout = s.menuBar()
	layout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(s.menuLayout, 1, 0, false).AddItem(s.pages, 0, 1, true).AddItem(s.footer, 1, 0, false).AddItem(s.toolbar(), 1, 0, false)
	s.rootLayout = layout
	s.app.SetRoot(layout, true).SetFocus(s.editor).SetInputCapture(s.key).EnableMouse(prefs.Mouse)
	s.app.SetBeforeDrawFunc(func(tcell.Screen) bool {
		s.syncChrome()
		s.refreshFooter()
		s.refreshBorders()
		return false
	})
	lastClickX, lastClickY := -1, -1
	s.app.SetMouseCapture(func(event *tcell.EventMouse, action tview.MouseAction) (*tcell.EventMouse, tview.MouseAction) {
		if !s.prefs.Mouse {
			return nil, action
		}
		// tview groups quick clicks even when they target different controls.
		// Preserve double clicks for text selection only at the same position.
		if action == tview.MouseLeftClick || action == tview.MouseLeftDoubleClick {
			x, y := event.Position()
			if x != lastClickX || y != lastClickY {
				action = tview.MouseLeftClick
			}
			lastClickX, lastClickY = x, y
		}
		return event, action
	})
	s.refreshLabels()
	s.setStatus("disconnected")
	go s.worker()
	if demo {
		s.showDemoBrowser()
		s.render(oracle.Result{Columns: []string{"SERVER_TIME", "MODE"}, Rows: [][]string{{"2026-10-14 12:00:00", s.tr("demo_cell")}}})
		s.setStatus("demo")
	}
	if prefsErr != nil {
		s.setStatus("settings_load_error", prefsErr)
	}
	if ready != nil {
		ready(s)
	}
	err := s.app.Run()
	if s.cancel != nil {
		s.cancel()
	}
	close(s.jobs)
	<-s.done
	return err
}

func (s *screen) submit(j job) {
	if s.busy {
		return
	}
	if j.action != "connect" && !s.connected {
		s.setStatus("connect_first")
		return
	}
	s.busy = true
	j.ctx, s.cancel = context.WithTimeout(context.Background(), 60*time.Second)
	if j.action == "schemas" || j.action == "objects" || j.action == "columns" || j.action == "object_info" || j.action == "source" || j.action == "arguments" {
		s.setStatus("browser_loading")
	} else {
		s.setStatus("running", j.action)
	}
	s.jobs <- j
}

func (s *screen) worker() {
	defer close(s.done)
	var session *oracle.Session
	defer func() {
		if session != nil {
			session.Close()
		}
	}()
	for j := range s.jobs {
		var result oracle.Result
		var catalog oracle.Catalog
		var objects []oracle.Object
		var columns []oracle.Column
		var object oracle.Object
		var source string
		var arguments []oracle.Argument
		var compilation oracle.Compilation
		var err error
		switch j.action {
		case "connect":
			var next *oracle.Session
			next, err = oracle.Open(j.ctx, j.config)
			if err == nil {
				if session != nil {
					session.Close()
				}
				session = next
			}
		case "execute":
			result, err = session.Execute(j.ctx, j.text)
		case "commit", "exit_commit":
			err = session.Finish(true)
		case "rollback", "exit_rollback":
			err = session.Finish(false)
		case "schemas":
			catalog, err = session.Catalog(j.ctx)
		case "objects":
			objects, err = session.Objects(j.ctx, j.object.Owner)
		case "columns":
			columns, err = session.Columns(j.ctx, j.object)
		case "describe":
			object, err = session.FindObject(j.ctx, j.text)
		case "object_info":
			object, err = session.ObjectInfo(j.ctx, j.object)
		case "source":
			source, err = session.Source(j.ctx, j.object)
		case "arguments":
			arguments, err = session.Arguments(j.ctx, j.object)
		case "compile_source":
			compilation, err = session.Compile(j.ctx, j.object, j.text)
		}
		s.app.QueueUpdateDraw(func() {
			s.cancel()
			s.cancel = nil
			s.busy = false
			if session != nil {
				s.pendingChanges = session.PendingChanges()
			}
			if j.action == "exit_commit" || j.action == "exit_rollback" {
				if err == nil {
					s.app.Stop()
				} else {
					s.exitDialog.SetText(s.tr("exit_failed") + "\n" + tview.Escape(err.Error()))
				}
				return
			}
			if j.action == "compile_source" {
				s.object = compilation.Object
				if compilation.Applied {
					s.sourceBase = j.text
				}
				s.renderDiagnostics(compilation.Diagnostics)
				s.renderObject(s.objectColumns)
				s.refreshSourceTitle()
				message := fmt.Sprintf(s.tr("compile_status"), s.object.Status, len(compilation.Diagnostics))
				if err != nil {
					message += "\n" + err.Error()
				}
				s.sourceInfo.SetText(message)
			}
			if err != nil {
				s.setStatus("error", err)
				if j.action == "columns" || j.action == "object_info" {
					s.renderObject(nil)
					s.objectInfo.SetText(s.objectInfo.GetText(false) + "\n" + s.tr("browser_error") + ": " + err.Error())
				}
				if j.action == "source" {
					s.sourceInfo.SetText(s.tr("browser_error") + ": " + err.Error())
				}
				if j.action == "arguments" {
					s.argumentInfo.SetText(s.tr("browser_error") + ": " + err.Error())
				}
			} else {
				switch j.action {
				case "connect":
					s.connected = true
					s.demo = false
					s.table.Clear()
					s.resetBrowser()
					s.setStatus("connected", j.config.User, j.config.Host, j.config.Port, j.config.Name, j.config.Mode)
					if !s.quitting {
						s.submit(job{action: "schemas"})
					}
				case "execute":
					s.render(result)
					if len(result.Columns) > 0 {
						key := "rows"
						if result.Truncated {
							key = "rows_limited"
						}
						s.setStatus(key, len(result.Rows), result.Elapsed.Round(time.Millisecond))
					} else {
						s.setStatus("executed", result.Affected, result.Elapsed.Round(time.Millisecond))
					}
				case "commit":
					s.setStatus("committed")
				case "rollback":
					s.setStatus("rolled_back")
				case "schemas":
					s.setStatus("schemas_loaded", len(catalog.Schemas))
					if !s.quitting {
						s.populateSchemas(catalog)
					}
				case "objects":
					s.populateObjects(j.node, objects)
					s.setStatus("objects_loaded", j.object.Owner, len(objects))
				case "columns":
					s.renderObject(columns)
					if len(columns) == 0 {
						s.objectInfo.SetText(s.objectInfo.GetText(false) + "\n" + s.tr("columns_empty"))
					}
					s.setStatus("columns_loaded", len(columns))
				case "describe":
					s.openObject(object)
				case "object_info":
					s.object = object
					s.renderObject(nil)
					s.setStatus("object_info_loaded", object.Name)
				case "source":
					s.acceptSource(source)
				case "arguments":
					s.renderArguments(arguments)
				}
			}
			if s.quitting {
				s.quitting = false
				s.requestQuit()
			}
		})
	}
}

func (s *screen) render(result oracle.Result) {
	s.resultHeaders = append([]string(nil), result.Columns...)
	s.resultRows = make([][]string, len(result.Rows))
	for i, row := range result.Rows {
		s.resultRows[i] = append([]string(nil), row...)
	}
	s.resultSelection = resultSelection{}
	s.table.Clear()
	for col, name := range result.Columns {
		s.table.SetCell(0, col, tview.NewTableCell(tview.Escape(name)).SetStyle(tcell.StyleDefault.Foreground(palette.accent).Background(palette.surface).Bold(true)).SetSelectable(false))
	}
	if len(result.Rows) > 0 && len(result.Columns) > 0 {
		s.table.Select(1, 0)
	}
	for row, values := range result.Rows {
		for col, value := range values {
			s.table.SetCell(row+1, col, tview.NewTableCell(tview.Escape(value)).SetTextColor(palette.text).SetMaxWidth(80))
		}
	}
}
