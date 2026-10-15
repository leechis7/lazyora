package ui

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/rivo/tview"
	"lazyora/internal/oracle"
	"lazyora/internal/settings"
)

func env(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

var connectionLabels = []string{"host", "port", "username", "password", "connection_type", "service_name", "saved_connections", "alias"}

func (s *screen) localizeConnection() {
	form := s.connectForm
	for i, key := range connectionLabels {
		if i == 4 || i == 6 {
			form.GetFormItem(i).(*tview.DropDown).SetLabel(s.tr(key))
		} else {
			form.GetFormItem(i).(*tview.InputField).SetLabel(s.tr(key))
		}
	}
	form.GetButton(0).SetLabel(s.tr("connect") + " (Ctrl+Enter / Alt+C)")
	form.GetButton(1).SetLabel(s.tr("cancel") + " (Esc)")
	form.GetButton(2).SetLabel(s.tr("save") + " (Ctrl+S / Alt+S)")
	if s.connectionOptions != nil {
		s.connectionOptions()
	}
	form.SetTitle(s.tr("connection_title"))
}

func (s *screen) closeConnection() {
	s.pages.RemovePage("connect")
	s.connectForm, s.connectAction, s.saveConnection = nil, nil, nil
	s.connectionOptions = nil
	if s.connectFocus != nil {
		s.app.SetFocus(s.connectFocus)
	} else {
		s.app.SetFocus(s.editor)
	}
	s.connectFocus = nil
}

func (s *screen) connectionForm() {
	s.connectFocus = s.app.GetFocus()
	form := tview.NewForm()
	s.connectForm = form
	mode := "service"
	form.AddInputField("", env("LAZYORA_HOST", "localhost"), 40, nil, nil).
		AddInputField("", env("LAZYORA_PORT", "1521"), 8, nil, nil).
		AddInputField("", os.Getenv("LAZYORA_USER"), 40, nil, nil).
		AddPasswordField("", os.Getenv("LAZYORA_PASSWORD"), 40, '*', nil).
		AddDropDown("", []string{"Service Name", "SID"}, 0, func(_ string, index int) {
			mode = "service"
			if index == 1 {
				mode = "sid"
			}
		}).AddInputField("", os.Getenv("LAZYORA_NAME"), 40, nil, nil)
	path := settings.ConnectionsPath(s.prefsPath)
	profiles, loadErr := settings.LoadConnections(path)
	options := []string{s.tr("new_connection")}
	for _, profile := range profiles {
		options = append(options, profile.Alias)
	}
	form.AddDropDown("", options, 0, nil).AddInputField("", "", 40, nil, nil)
	selectProfile := func(_ string, index int) {
		// Always discard the previous password, including environment defaults.
		form.GetFormItem(3).(*tview.InputField).SetText("")
		if index <= 0 {
			form.GetFormItem(7).(*tview.InputField).SetText("")
			return
		}
		p := profiles[index-1]
		for i, value := range map[int]string{0: p.Host, 1: strconv.Itoa(p.Port), 2: p.User, 5: p.Name, 7: p.Alias} {
			form.GetFormItem(i).(*tview.InputField).SetText(value)
		}
		modeIndex := 0
		if p.Mode == "sid" {
			modeIndex = 1
		}
		form.GetFormItem(4).(*tview.DropDown).SetCurrentOption(modeIndex)
	}
	form.GetFormItem(6).(*tview.DropDown).SetSelectedFunc(selectProfile)
	s.connectionOptions = func() {
		dropdown := form.GetFormItem(6).(*tview.DropDown)
		index, _ := dropdown.GetCurrentOption()
		options = []string{s.tr("new_connection")}
		for _, p := range profiles {
			options = append(options, p.Alias)
		}
		dropdown.SetOptions(options, nil).SetCurrentOption(index).SetSelectedFunc(selectProfile)
	}
	readConfig := func() (oracle.Config, bool) {
		port, err := oracle.ParsePort(form.GetFormItem(1).(*tview.InputField).GetText())
		if err != nil {
			form.SetTitle(" " + s.tr("port_error") + " ")
			return oracle.Config{}, false
		}
		config := oracle.Config{Host: strings.TrimSpace(form.GetFormItem(0).(*tview.InputField).GetText()), Port: port,
			User: form.GetFormItem(2).(*tview.InputField).GetText(), Password: form.GetFormItem(3).(*tview.InputField).GetText(),
			Mode: mode, Name: strings.TrimSpace(form.GetFormItem(5).(*tview.InputField).GetText())}
		if _, err = config.DSN(); err != nil {
			form.SetTitle(" " + s.tr("required") + " ")
			return oracle.Config{}, false
		}
		return config, true
	}
	s.connectAction = func() {
		config, ok := readConfig()
		if !ok {
			return
		}
		s.closeConnection()
		s.submit(job{action: "connect", config: config})
	}
	s.saveConnection = func() {
		if loadErr != nil {
			form.SetTitle(" " + fmt.Sprintf(s.tr("connection_save_error"), loadErr) + " ")
			return
		}
		config, ok := readConfig()
		if !ok {
			return
		}
		alias := strings.TrimSpace(form.GetFormItem(7).(*tview.InputField).GetText())
		if alias == "" {
			form.SetTitle(" " + s.tr("alias_required") + " ")
			return
		}
		profile := settings.Connection{Alias: alias, Host: config.Host, Port: config.Port, User: config.User, Mode: config.Mode, Name: config.Name}
		next := append([]settings.Connection(nil), profiles...)
		index := len(next)
		for i, p := range next {
			if p.Alias == alias {
				index = i
				break
			}
		}
		if index == len(next) {
			next = append(next, profile)
		} else {
			next[index] = profile
		}
		if err := settings.SaveConnections(path, next); err != nil {
			form.SetTitle(" " + fmt.Sprintf(s.tr("connection_save_error"), err) + " ")
			return
		}
		profiles = next
		// Updating the selection after save must keep the typed password in memory.
		dropdown := form.GetFormItem(6).(*tview.DropDown)
		s.connectionOptions()
		dropdown.SetSelectedFunc(nil).SetCurrentOption(index + 1).SetSelectedFunc(selectProfile)
		form.GetFormItem(7).(*tview.InputField).SetText(alias)
		form.SetTitle(" " + s.tr("connection_saved") + " ")
	}
	form.AddButton("", s.connectAction).AddButton("", s.closeConnection).AddButton("", s.saveConnection)
	clickable(form.GetButton(0))
	clickable(form.GetButton(1))
	clickable(form.GetButton(2))
	form.SetBorder(true)
	s.localizeConnection()
	if loadErr != nil {
		form.SetTitle(" " + fmt.Sprintf(s.tr("connection_load_error"), loadErr) + " ")
	}
	s.pages.AddPage("connect", form, true, true)
	s.app.SetFocus(form)
}
