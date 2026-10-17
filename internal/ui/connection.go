package ui

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
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

const (
	connectionAlias = iota
	connectionHost
	connectionPort
	connectionUser
	connectionPassword
	connectionMode
	connectionName
)

var connectionLabels = []string{"alias", "host", "port", "username", "password", "connection_type", "service_name"}

func (s *screen) localizeConnection() {
	form := s.connectForm
	for i, key := range connectionLabels {
		if i == connectionMode {
			form.GetFormItem(i).(*choiceField).SetLabel(s.tr(key))
		} else {
			form.GetFormItem(i).(*tview.InputField).SetLabel(s.tr(key))
		}
	}
	form.GetButton(0).SetLabel(s.tr("connect") + " (Alt+C)")
	form.GetButton(1).SetLabel(s.tr("cancel") + " (Esc)")
	form.GetButton(2).SetLabel(s.tr("save") + " (Alt+S)")
	if s.connectionOptions != nil {
		s.connectionOptions()
	}
	form.SetTitle(s.tr("connection_details"))
	s.connectProfiles.SetTitle(s.tr("saved_connections_title"))
	s.connectProfiles.GetFormItem(0).(*choiceField).SetLabel(s.tr("saved_connections"))
}

func (s *screen) closeConnection() {
	s.pages.RemovePage("connect")
	s.connectForm, s.connectAction, s.saveConnection = nil, nil, nil
	s.connectionOptions = nil
	s.connectProfiles = nil
	if s.connectFocus != nil {
		s.app.SetFocus(s.connectFocus)
	} else {
		s.app.SetFocus(s.editor)
	}
	s.connectFocus = nil
}

func (s *screen) connectionForm() {
	s.connectFocus = s.app.GetFocus()
	form := tview.NewForm().SetItemPadding(0)
	s.connectForm = form
	mode := "service"
	form.AddInputField("", "", 40, nil, nil).
		AddInputField("", env("LAZYORA_HOST", "localhost"), 40, nil, nil).
		AddInputField("", env("LAZYORA_PORT", "1521"), 8, nil, nil).
		AddInputField("", os.Getenv("LAZYORA_USER"), 40, nil, nil).
		AddPasswordField("", os.Getenv("LAZYORA_PASSWORD"), 40, '*', nil).
		AddFormItem(s.newChoice("", []string{"Service Name", "SID"}, 0, func(_ string, index int) {
			mode = "service"
			if index == 1 {
				mode = "sid"
			}
		})).AddInputField("", os.Getenv("LAZYORA_NAME"), 40, nil, nil)
	path := settings.ConnectionsPath(s.prefsPath)
	profiles, loadErr := settings.LoadConnections(path)
	options := []string{s.tr("new_connection")}
	for _, profile := range profiles {
		options = append(options, profile.Alias)
	}
	selector := tview.NewForm().AddFormItem(s.newChoice("", options, 0, nil))
	s.connectProfiles = selector
	selector.SetBorder(true)
	styleForm(selector)
	selectProfile := func(_ string, index int) {
		// Always discard the previous password, including environment defaults.
		form.GetFormItem(connectionPassword).(*tview.InputField).SetText("")
		if index <= 0 {
			for i, value := range map[int]string{connectionAlias: "", connectionHost: env("LAZYORA_HOST", "localhost"), connectionPort: env("LAZYORA_PORT", "1521"), connectionUser: os.Getenv("LAZYORA_USER"), connectionName: os.Getenv("LAZYORA_NAME")} {
				form.GetFormItem(i).(*tview.InputField).SetText(value)
			}
			form.GetFormItem(connectionMode).(*choiceField).SetCurrentOption(0)
			return
		}
		p := profiles[index-1]
		for i, value := range map[int]string{connectionHost: p.Host, connectionPort: strconv.Itoa(p.Port), connectionUser: p.User, connectionName: p.Name, connectionAlias: p.Alias} {
			form.GetFormItem(i).(*tview.InputField).SetText(value)
		}
		modeIndex := 0
		if p.Mode == "sid" {
			modeIndex = 1
		}
		form.GetFormItem(connectionMode).(*choiceField).SetCurrentOption(modeIndex)
	}
	selector.GetFormItem(0).(*choiceField).SetSelectedFunc(selectProfile)
	s.connectionOptions = func() {
		dropdown := selector.GetFormItem(0).(*choiceField)
		index, _ := dropdown.GetCurrentOption()
		options = []string{s.tr("new_connection")}
		for _, p := range profiles {
			options = append(options, p.Alias)
		}
		dropdown.SetOptions(options, nil).SetCurrentOption(index).SetSelectedFunc(selectProfile)
	}
	readConfig := func() (oracle.Config, bool) {
		port, err := oracle.ParsePort(form.GetFormItem(connectionPort).(*tview.InputField).GetText())
		if err != nil {
			form.SetTitle(" " + s.tr("port_error") + " ")
			return oracle.Config{}, false
		}
		config := oracle.Config{Host: strings.TrimSpace(form.GetFormItem(connectionHost).(*tview.InputField).GetText()), Port: port,
			User: form.GetFormItem(connectionUser).(*tview.InputField).GetText(), Password: form.GetFormItem(connectionPassword).(*tview.InputField).GetText(),
			Mode: mode, Name: strings.TrimSpace(form.GetFormItem(connectionName).(*tview.InputField).GetText())}
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
		alias := strings.TrimSpace(form.GetFormItem(connectionAlias).(*tview.InputField).GetText())
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
		dropdown := selector.GetFormItem(0).(*choiceField)
		s.connectionOptions()
		dropdown.SetSelectedFunc(nil).SetCurrentOption(index + 1).SetSelectedFunc(selectProfile)
		form.GetFormItem(connectionAlias).(*tview.InputField).SetText(alias)
		form.SetTitle(" " + s.tr("connection_saved") + " ")
	}
	form.AddButton("", s.connectAction).AddButton("", s.closeConnection).AddButton("", s.saveConnection)
	clickable(form.GetButton(0))
	clickable(form.GetButton(1))
	clickable(form.GetButton(2))
	form.SetBorder(true)
	styleForm(form)
	s.localizeConnection()
	if loadErr != nil {
		form.SetTitle(" " + fmt.Sprintf(s.tr("connection_load_error"), loadErr) + " ")
	}
	root := tview.NewFlex().SetDirection(tview.FlexRow).AddItem(selector, 5, 0, false).AddItem(form, 0, 1, true)
	s.pages.AddPage("connect", root, true, true)
	s.app.SetFocus(form)
}

// Keep the profile picker separate while retaining a single Tab cycle.
func (s *screen) connectionTab(event *tcell.EventKey) bool {
	if event.Key() != tcell.KeyTab && event.Key() != tcell.KeyBacktab {
		return false
	}
	for _, form := range []*tview.Form{s.connectProfiles, s.connectForm} {
		for i := 0; i < form.GetFormItemCount(); i++ {
			if dropdown, ok := form.GetFormItem(i).(*choiceField); ok && dropdown.IsOpen() {
				return false
			}
		}
	}
	backwards := event.Key() == tcell.KeyBacktab
	if s.connectProfiles.HasFocus() {
		index := 0
		if backwards {
			index = s.connectForm.GetFormItemCount() + s.connectForm.GetButtonCount() - 1
		}
		focusForm(s, s.connectForm, index)
		return true
	}
	item, button := s.connectForm.GetFocusedItemIndex()
	if (backwards && item == 0) || (!backwards && button == s.connectForm.GetButtonCount()-1) {
		focusForm(s, s.connectProfiles, 0)
		return true
	}
	return false
}
