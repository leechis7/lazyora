package ui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/oracle"
)

type job struct {
	ctx    context.Context
	action string
	config oracle.Config
	text   string
}

type screen struct {
	app                             *tview.Application
	pages                           *tview.Pages
	editor                          *tview.TextArea
	table                           *tview.Table
	status                          *tview.TextView
	jobs                            chan job
	done                            chan struct{}
	busy, quitting, demo, connected bool
	cancel                          context.CancelFunc
}

func Run(demo bool) error {
	return run(demo, nil, nil)
}

func run(demo bool, terminal tcell.Screen, ready func(*screen)) error {
	s := &screen{app: tview.NewApplication(), jobs: make(chan job, 1), done: make(chan struct{}), demo: demo}
	if terminal != nil {
		s.app.SetScreen(terminal)
	}
	s.editor = tview.NewTextArea().SetText("SELECT sysdate AS server_time FROM dual;", false)
	s.editor.SetBorder(true).SetTitle(" SQL — one statement or PL/SQL block ")
	s.table = tview.NewTable().SetBorders(false).SetSelectable(true, true).SetFixed(1, 0)
	s.table.SetBorder(true).SetTitle(" Results ")
	s.status = tview.NewTextView().SetText("Disconnected. F2 opens connection settings.")
	help := tview.NewTextView().SetText("F2 Connect | F4 Run | F5 Commit | F6 Rollback | F8 Cancel | Tab Focus | Ctrl+Q Quit")
	root := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(help, 1, 0, false).AddItem(s.editor, 0, 2, true).
		AddItem(s.table, 0, 3, false).AddItem(s.status, 2, 0, false)
	s.pages = tview.NewPages().AddPage("main", root, true, true)
	s.app.SetRoot(s.pages, true).SetFocus(s.editor).SetInputCapture(s.key)
	go s.worker()
	if demo {
		s.render(oracle.Result{Columns: []string{"SERVER_TIME", "MODE"}, Rows: [][]string{{"2026-10-14 12:00:00", "Demo — no database connected"}}})
		s.status.SetText("Demo mode. F2 connects to a real Oracle database; no statements run in demo mode.")
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

func (s *screen) key(event *tcell.EventKey) *tcell.EventKey {
	// Let the connection form handle its own Tab key.
	page, _ := s.pages.GetFrontPage()
	switch event.Key() {
	case tcell.KeyCtrlQ, tcell.KeyCtrlC:
		if s.busy {
			s.quitting = true
			s.cancel()
			s.status.SetText("Cancelling operation before exit…")
		} else {
			s.app.Stop()
		}
		return nil
	case tcell.KeyF8:
		if s.cancel != nil {
			s.cancel()
		}
		return nil
	case tcell.KeyEscape:
		if page == "connect" {
			s.pages.RemovePage("connect")
			s.app.SetFocus(s.editor)
			return nil
		}
	case tcell.KeyF2:
		if !s.busy {
			s.connectionForm()
		}
		return nil
	}
	if page != "main" {
		return event
	}
	switch event.Key() {
	case tcell.KeyF4:
		s.submit(job{action: "execute", text: s.editor.GetText()})
		return nil
	case tcell.KeyF5:
		s.submit(job{action: "commit"})
		return nil
	case tcell.KeyF6:
		s.submit(job{action: "rollback"})
		return nil
	case tcell.KeyTab:
		if s.app.GetFocus() == s.editor {
			s.app.SetFocus(s.table)
		} else {
			s.app.SetFocus(s.editor)
		}
		return nil
	}
	return event
}

func env(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func (s *screen) connectionForm() {
	form := tview.NewForm()
	mode := "service"
	form.AddInputField("Host", env("LAZYORA_HOST", "localhost"), 40, nil, nil).
		AddInputField("Port", env("LAZYORA_PORT", "1521"), 8, nil, nil).
		AddInputField("Username", os.Getenv("LAZYORA_USER"), 40, nil, nil).
		AddPasswordField("Password", os.Getenv("LAZYORA_PASSWORD"), 40, '*', nil).
		AddDropDown("Connection type", []string{"Service Name", "SID"}, 0, func(_ string, index int) {
			mode = "service"
			if index == 1 {
				mode = "sid"
			}
		}).AddInputField("Service Name / SID", os.Getenv("LAZYORA_NAME"), 40, nil, nil)
	form.AddButton("Connect", func() {
		port, err := oracle.ParsePort(form.GetFormItem(1).(*tview.InputField).GetText())
		if err != nil {
			form.SetTitle(" " + err.Error() + " ")
			return
		}
		config := oracle.Config{Host: strings.TrimSpace(form.GetFormItem(0).(*tview.InputField).GetText()), Port: port,
			User: form.GetFormItem(2).(*tview.InputField).GetText(), Password: form.GetFormItem(3).(*tview.InputField).GetText(),
			Mode: mode, Name: strings.TrimSpace(form.GetFormItem(5).(*tview.InputField).GetText())}
		if _, err = config.DSN(); err != nil {
			form.SetTitle(" " + err.Error() + " ")
			return
		}
		s.pages.RemovePage("connect")
		s.app.SetFocus(s.editor)
		s.submit(job{action: "connect", config: config})
	}).AddButton("Cancel", func() { s.pages.RemovePage("connect"); s.app.SetFocus(s.editor) })
	form.SetBorder(true).SetTitle(" Oracle connection — reconnect rolls back previous session ")
	s.pages.AddPage("connect", form, true, true)
	s.app.SetFocus(form)
}

func (s *screen) submit(j job) {
	if s.busy {
		return
	}
	if j.action != "connect" && !s.connected {
		s.status.SetText("Connect first with F2.")
		return
	}
	s.busy = true
	j.ctx, s.cancel = context.WithTimeout(context.Background(), 60*time.Second)
	s.status.SetText("Running " + j.action + "… F8 cancels. Oracle DDL can implicitly commit.")
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
				s.status.SetText("Error: " + tview.Escape(err.Error()))
			} else {
				switch j.action {
				case "connect":
					s.connected = true
					s.demo = false
					s.table.Clear()
					s.status.SetText(tview.Escape(fmt.Sprintf("Connected: %s@%s:%d/%s (%s). Manual commit; DDL implicitly commits.", j.config.User, j.config.Host, j.config.Port, j.config.Name, j.config.Mode)))
				case "execute":
					s.render(result)
					if len(result.Columns) > 0 {
						message := fmt.Sprintf("%d rows · %s", len(result.Rows), result.Elapsed.Round(time.Millisecond))
						if result.Truncated {
							message += " · limited to 1000 rows"
						}
						s.status.SetText(message)
					} else {
						s.status.SetText(fmt.Sprintf("Statement completed · affected rows: %d · %s · F5 Commit / F6 Rollback", result.Affected, result.Elapsed.Round(time.Millisecond)))
					}
				default:
					s.status.SetText(j.action + " completed.")
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
		s.table.SetCell(0, col, tview.NewTableCell(tview.Escape(name)).SetTextColor(tcell.ColorYellow).SetSelectable(false))
	}
	for row, values := range result.Rows {
		for col, value := range values {
			s.table.SetCell(row+1, col, tview.NewTableCell(tview.Escape(value)).SetMaxWidth(80))
		}
	}
}
