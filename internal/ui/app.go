package ui

import (
	"context"
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
}

type screen struct {
	app                                      *tview.Application
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
	jobs                                     chan job
	done                                     chan struct{}
	busy, quitting, demo, connected          bool
	cancel                                   context.CancelFunc
	helpFocus, connectFocus, prefsFocus      tview.Primitive
	connectForm, prefsForm                   *tview.Form
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
	s.app.SetScreen(terminal)
	s.editor = tview.NewTextArea().SetText("SELECT sysdate AS server_time FROM dual;", false)
	s.editor.SetBorder(true)
	s.table = tview.NewTable().SetBorders(false).SetSelectable(true, true).SetFixed(1, 0)
	s.table.SetBorder(true)
	s.status = tview.NewTextView()
	root := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(s.editor, 0, 2, true).
		AddItem(s.table, 0, 3, false).AddItem(s.status, 2, 0, false)
	s.pages = tview.NewPages().AddPage("main", root, true, true)
	s.footer = tview.NewTextView().SetWrap(false)
	s.styleScreen()
	layout := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(s.pages, 0, 1, true).AddItem(s.footer, 1, 0, false).AddItem(s.toolbar(), 1, 0, false)
	s.app.SetRoot(layout, true).SetFocus(s.editor).SetInputCapture(s.key).EnableMouse(prefs.Mouse)
	s.app.SetBeforeDrawFunc(func(tcell.Screen) bool { s.refreshFooter(); return false })
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
	s.setStatus("running", j.action)
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
		case "commit":
			err = session.Finish(true)
		case "rollback":
			err = session.Finish(false)
		}
		s.app.QueueUpdateDraw(func() {
			s.cancel()
			s.cancel = nil
			s.busy = false
			if err != nil {
				s.setStatus("error", err)
			} else {
				switch j.action {
				case "connect":
					s.connected = true
					s.demo = false
					s.table.Clear()
					s.setStatus("connected", j.config.User, j.config.Host, j.config.Port, j.config.Name, j.config.Mode)
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
				}
			}
			if s.quitting {
				s.app.Stop()
			}
		})
	}
}

func (s *screen) render(result oracle.Result) {
	s.table.Clear()
	for col, name := range result.Columns {
		s.table.SetCell(0, col, tview.NewTableCell(tview.Escape(name)).SetStyle(tcell.StyleDefault.Foreground(palette.accent).Background(palette.surface).Bold(true)).SetSelectable(false))
	}
	for row, values := range result.Rows {
		for col, value := range values {
			s.table.SetCell(row+1, col, tview.NewTableCell(tview.Escape(value)).SetTextColor(palette.text).SetMaxWidth(80))
		}
	}
}
