package ui

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/credentials"
	"lazyora/internal/exportdata"
	"lazyora/internal/oracle"
	"lazyora/internal/settings"
	"lazyora/internal/sqlfile"
)

type job struct {
	query         *oracle.QuerySpec
	path          string
	dataset       exportdata.Dataset
	exportOptions exportdata.Options
	revision      exportdata.Revision
	bindings      map[string]oracle.BindValue
	ctx           context.Context
	action        string
	config        oracle.Config
	resolution    oracle.ReconnectResolution
	text          string
	object        oracle.Object
	node          *tview.TreeNode
}

type screen struct {
	resultQuery            *oracle.QuerySpec
	exportForm             *tview.Form
	exportInfo             *tview.TextView
	exportFocus            tview.Primitive
	exportSubmit           func()
	resultValues           [][]oracle.Cell
	resultTruncated        bool
	planTree               *tview.TreeView
	planText               *tview.TextView
	planFocus              tview.Primitive
	bindForm               *tview.Form
	bindSQL                string
	bindValues             map[string]oracle.BindValue
	bindSubmit, bindCancel func()
	scriptReport           oracle.ScriptReport
	scriptList             *tview.List
	scriptResult           *tview.Table
	scriptText             *tview.TextView
	scriptFocus            tview.Primitive
	tabs                   []*sqlTab
	activeTab              int
	tabBar                 *tview.Flex
	document               *sqlfile.Document
	fileDialog             *fileDialog
	documentModal          *tview.Modal

	sqlConfirmText  *tview.TextView
	sqlConfirmFocus tview.Primitive

	vault                                                         *credentials.Store
	reloadConnectionProfiles                                      func() error
	vaultWorkDone                                                 chan struct{}
	vaultBusy                                                     bool
	masterForm, vaultForm                                         *tview.Form
	masterSubmit, masterCancel, closeVault, clearConnectionSecret func()
	vaultFocus                                                    tview.Primitive

	sqlHistory                               []string
	historyList                              *tview.List
	historyFocus                             tview.Primitive
	searchForm                               *tview.Form
	searchFocus                              tview.Primitive
	searchAction                             func(string)
	findText, replaceText                    string
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
	browserPanel                             *tview.Flex
	browserForm                              *tview.Form
	browserUsers, browserTypes               *choiceField
	browserSchemas                           []string
	browserObjects                           []oracle.Object
	browserKind                              int
	object                                   oracle.Object
	objectInfo                               *tview.TextView
	objectFocus                              tview.Primitive
	objectTable                              *tview.Table
	objectColumns                            []oracle.Column
	pendingChanges                           bool
	transactionUncertain                     bool
	pendingWork                              oracle.PendingWork
	pendingTable                             *tview.Table
	pendingSQL                               *tview.TextView
	pendingFocus                             tview.Primitive
	exitDialog                               *tview.Modal
	reconnectDialog                          *tview.Modal
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
	s.initDocument()
	s.editor.SetMouseCapture(func(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
		if action == tview.MouseRightClick {
			s.app.SetFocus(s.editor)
			s.showContextMenu()
			return tview.MouseConsumed, nil
		}
		return action, event
	})
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
		page, pageRoot := s.pages.GetFrontPage()
		if page == "sql_confirm" {
			pageRoot.MouseHandler()(action, event, func(p tview.Primitive) { s.app.SetFocus(p) })
			return tview.MouseConsumed, nil
		}
		var modal *tview.Modal
		if page == "master" || page == "vault" {
			var form *tview.Form
			if page == "master" {
				form = s.masterForm
			} else {
				form = s.vaultForm
			}
			if !s.vaultBusy && form != nil {
				form.MouseHandler()(action, event, func(p tview.Primitive) { s.app.SetFocus(p) })
			}
			return tview.MouseConsumed, nil
		}
		if page == "export" && s.busy {
			button := s.exportForm.GetButton(1)
			x, y := event.Position()
			if button.InRect(x, y) {
				button.MouseHandler()(action, event, func(p tview.Primitive) { s.app.SetFocus(p) })
			}
			return tview.MouseConsumed, nil
		}
		if page == "file" || page == "script" || page == "bind" || page == "plan" || page == "export" {
			pageRoot.MouseHandler()(action, event, func(p tview.Primitive) { s.app.SetFocus(p) })
			return tview.MouseConsumed, nil
		}
		if page == "file_unsaved" {
			modal = s.documentModal
		}
		if page == "exit" {
			modal = s.exitDialog
		}
		if page == "reconnect" {
			modal = s.reconnectDialog
		}
		if page == "program_confirm" {
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
		if s.vaultBusy {
			return nil, action
		}
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
	s.initTabs()
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
	if !demo {
		s.connectionForm()
	}
	if ready != nil {
		ready(s)
	}
	err := s.app.Run()
	if s.vaultWorkDone != nil {
		<-s.vaultWorkDone
	}
	s.lockPasswords()
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
	if j.action != "connect" && j.action != "export" && !s.connected {
		s.setStatus("connect_first")
		return
	}
	s.busy = true
	if j.action == "export" && j.query != nil {
		j.ctx, s.cancel = context.WithCancel(context.Background())
	} else {
		j.ctx, s.cancel = context.WithTimeout(context.Background(), 60*time.Second)
	}
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
		var script oracle.ScriptReport
		var plan []oracle.PlanStep
		var exported int64
		var err error
		switch j.action {
		case "export":
			progressAt := time.Time{}
			options := j.exportOptions
			options.Progress = func(count int64) {
				if count != 1 && time.Since(progressAt) < 250*time.Millisecond {
					return
				}
				progressAt = time.Now()
				s.app.QueueUpdateDraw(func() {
					if s.exportInfo != nil {
						s.exportInfo.SetText(fmt.Sprintf(s.tr("export_progress"), count))
					}
				})
			}
			if j.query != nil {
				options.Unlimited = true
				if session == nil {
					err = fmt.Errorf("connect before exporting the full result")
					break
				}
				exported, err = exportdata.SaveStream(j.ctx, j.path, j.revision, func() (exportdata.Source, error) {
					rows, openErr := session.OpenQuery(j.ctx, j.query)
					if openErr != nil {
						return exportdata.Source{}, openErr
					}
					if !slices.Equal(rows.Columns, j.dataset.Columns) {
						rows.Close()
						return exportdata.Source{}, fmt.Errorf("result columns changed; run the query again before exporting")
					}
					return exportdata.Source{Columns: rows.Columns, Next: rows.Next, Close: rows.Close}, nil
				}, options)
			} else {
				exported, err = exportdata.SaveStream(j.ctx, j.path, j.revision, func() (exportdata.Source, error) { return exportdata.FromDataset(j.dataset), nil }, options)
			}
		case "connect":
			session, err = oracle.Reconnect(j.ctx, session, j.config, j.resolution)
		case "execute":
			result, err = session.Execute(j.ctx, j.text, j.bindings)
		case "script":
			script, err = session.Script(j.ctx, j.text, j.bindings)
		case "plan":
			plan, err = session.Explain(j.ctx, j.text, j.bindings)
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
				s.transactionUncertain = session.TransactionUncertain()
				s.pendingWork = session.PendingWork()
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
			if j.action == "script" {
				s.scriptReport = script
				for _, step := range script.Steps {
					if len(step.Result.Columns) > 0 {
						s.render(step.Result)
					}
				}
				s.showScriptReport()
				s.setStatus("script_done", len(script.Steps), script.Total)
			}
			if err != nil {
				s.setStatus("error", err)
				if j.action == "export" && s.exportInfo != nil {
					s.exportInfo.SetText(s.tr("export_error") + "\n" + err.Error())
				}
				if j.action == "columns" || j.action == "object_info" {
					s.renderObject(nil)
					s.objectInfo.SetText(s.objectInfo.GetText(false) + "\n" + s.tr("browser_error") + ": " + err.Error())
				}
				if j.action == "objects" {
					s.browser.SetRoot(j.node).SetCurrentNode(nil)
					j.node.ClearChildren().AddChild(s.treeMessage("browser_error"))
				}
				if j.action == "source" {
					s.sourceInfo.SetText(s.tr("browser_error") + ": " + err.Error())
				}
				if j.action == "arguments" {
					s.argumentInfo.SetText(s.tr("browser_error") + ": " + err.Error())
				}
			} else {
				switch j.action {
				case "export":
					s.closeExport()
					if j.query != nil {
						s.setStatus("export_full_saved", exported, j.path)
					} else {
						s.setStatus("export_saved", exported, j.path)
					}
				case "plan":
					s.showPlan(j.text, plan)
					s.setStatus("plan_loaded", len(plan))
				case "connect":
					s.clearBindValues()
					s.connected = true
					s.demo = false
					s.table.Clear()
					s.resultRows = nil
					s.resultHeaders = nil
					s.resultValues = nil
					s.resultTruncated = false
					s.resultSelection = resultSelection{}
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
	s.resultQuery = result.Query
	s.resultTruncated = result.Truncated
	s.resultValues = make([][]oracle.Cell, len(result.Rows))
	for i, row := range result.Rows {
		s.resultValues[i] = make([]oracle.Cell, len(row))
		for j, value := range row {
			cell := oracle.Cell{Text: value, Type: "VARCHAR2"}
			if i < len(result.Values) && j < len(result.Values[i]) {
				cell = result.Values[i][j]
			}
			s.resultValues[i][j] = cell
		}
	}
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
