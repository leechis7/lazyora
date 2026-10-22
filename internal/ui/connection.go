package ui

import (
	"errors"
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
	connectionSavePassword
	connectionEnvironment
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
	form.GetFormItem(connectionSavePassword).(*tview.Checkbox).SetLabel(s.tr("save_password"))
	environmentChoice := form.GetFormItem(connectionEnvironment).(*choiceField)
	environmentIndex, _ := environmentChoice.GetCurrentOption()
	environmentChoice.SetLabel(s.tr("environment"))
	environmentChoice.SetOptions([]string{s.tr("environment_none"), s.tr("environment_dev"), s.tr("environment_test"), s.tr("environment_prod")}, nil).SetCurrentOption(environmentIndex)
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
	if s.connectForm != nil {
		s.connectForm.GetFormItem(connectionPassword).(*tview.InputField).SetText("")
	}
	s.clearConnectionSecret = nil
	s.reloadConnectionProfiles = nil
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
	form := tview.NewForm().SetItemPadding(1)
	s.connectForm = form
	mode := "service"
	autofilled := false
	var invalidateSecret func()
	form.AddInputField("", "", 40, nil, nil).
		AddInputField("", env("LAZYORA_HOST", "localhost"), 40, nil, nil).
		AddInputField("", env("LAZYORA_PORT", "1521"), 8, nil, nil).
		AddInputField("", os.Getenv("LAZYORA_USER"), 40, nil, nil).
		AddPasswordField("", os.Getenv("LAZYORA_PASSWORD"), 40, '*', nil).
		AddFormItem(s.newChoice("", []string{"Service Name", "SID"}, 0, func(_ string, index int) {
			if invalidateSecret != nil {
				invalidateSecret()
			}
			mode = "service"
			if index == 1 {
				mode = "sid"
			}
		})).AddInputField("", os.Getenv("LAZYORA_NAME"), 40, nil, nil)
	form.AddCheckbox(s.tr("save_password"), false, nil)
	environments := []string{"", "dev", "test", "prod"}
	form.AddFormItem(s.newChoice(s.tr("environment"), []string{s.tr("environment_none"), s.tr("environment_dev"), s.tr("environment_test"), s.tr("environment_prod")}, 0, nil))
	password := form.GetFormItem(connectionPassword).(*tview.InputField)
	invalidateSecret = func() {
		if autofilled {
			password.SetText("")
			autofilled = false
		}
	}
	for _, index := range []int{connectionHost, connectionPort, connectionUser, connectionName} {
		form.GetFormItem(index).(*tview.InputField).SetChangedFunc(func(string) { invalidateSecret() })
	}
	password.SetChangedFunc(func(string) { autofilled = false })
	s.clearConnectionSecret = func() { password.SetText(""); autofilled = false }
	path := settings.ConnectionsPath(s.prefsPath)
	profiles, loadErr := settings.LoadConnections(path)
	options := []string{s.tr("new_connection")}
	for _, profile := range profiles {
		options = append(options, profile.Alias)
	}
	selector := tview.NewForm().AddFormItem(s.newChoice("", options, 0, nil))
	s.connectProfiles = selector
	selector.SetBorder(true)
	s.styleForm(selector)
	selectProfile := func(_ string, index int) {
		// Always discard the previous password, including environment defaults.
		form.GetFormItem(connectionSavePassword).(*tview.Checkbox).SetChecked(false)
		form.GetFormItem(connectionEnvironment).(*choiceField).SetCurrentOption(0)
		form.GetFormItem(connectionPassword).(*tview.InputField).SetText("")
		if index <= 0 {
			for i, value := range map[int]string{connectionAlias: "", connectionHost: env("LAZYORA_HOST", "localhost"), connectionPort: env("LAZYORA_PORT", "1521"), connectionUser: os.Getenv("LAZYORA_USER"), connectionName: os.Getenv("LAZYORA_NAME")} {
				form.GetFormItem(i).(*tview.InputField).SetText(value)
			}
			form.GetFormItem(connectionMode).(*choiceField).SetCurrentOption(0)
			return
		}
		p := profiles[index-1]
		for i, environment := range environments {
			if environment == p.Environment {
				form.GetFormItem(connectionEnvironment).(*choiceField).SetCurrentOption(i)
			}
		}
		for i, value := range map[int]string{connectionHost: p.Host, connectionPort: strconv.Itoa(p.Port), connectionUser: p.User, connectionName: p.Name, connectionAlias: p.Alias} {
			form.GetFormItem(i).(*tview.InputField).SetText(value)
		}
		modeIndex := 0
		if p.Mode == "sid" {
			modeIndex = 1
		}
		form.GetFormItem(connectionMode).(*choiceField).SetCurrentOption(modeIndex)
		if p.SavedPassword {
			form.GetFormItem(connectionSavePassword).(*tview.Checkbox).SetChecked(true)
			s.ensurePasswords(func() {
				value, found, err := s.vault.Get(p.ID, p.Binding())
				if err != nil {
					s.showVaultError(err)
					return
				}
				if !found {
					form.SetTitle(s.tr("vault_missing"))
					return
				}
				password.SetText(value)
				autofilled = true
			})
		}
	}
	selector.GetFormItem(0).(*choiceField).SetSelectedFunc(selectProfile)
	s.connectionOptions = func() {
		dropdown := selector.GetFormItem(0).(*choiceField)
		index, _ := dropdown.GetCurrentOption()
		options = []string{s.tr("new_connection")}
		for _, p := range profiles {
			options = append(options, p.Alias)
		}
		dropdown.SetSelectedFunc(nil).SetOptions(options, nil).SetCurrentOption(index).SetSelectedFunc(selectProfile)
	}
	s.reloadConnectionProfiles = func() error {
		next, err := settings.LoadConnections(path)
		if err != nil {
			return err
		}
		profiles = next
		s.connectionOptions()
		form.GetFormItem(connectionSavePassword).(*tview.Checkbox).SetChecked(false)
		return nil
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
		if s.busy || s.reconnectDialog != nil {
			return
		}
		config, ok := readConfig()
		if !ok {
			return
		}
		environment, _ := form.GetFormItem(connectionEnvironment).(*choiceField).GetCurrentOption()
		s.pendingIdentity = settings.Connection{Alias: strings.TrimSpace(form.GetFormItem(connectionAlias).(*tview.InputField).GetText()), Environment: environments[environment]}
		s.requestReconnect(config)
	}
	saveProfile := func() {
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
		environment, _ := form.GetFormItem(connectionEnvironment).(*choiceField).GetCurrentOption()
		profile := settings.Connection{Environment: environments[environment], Alias: alias, Host: config.Host, Port: config.Port, User: config.User, Mode: config.Mode, Name: config.Name}
		next := append([]settings.Connection(nil), profiles...)
		index := len(next)
		for i, p := range next {
			if p.Alias == alias {
				index = i
				break
			}
		}
		if index < len(next) {
			profile.ID = next[index].ID
		}
		if profile.ID == "" {
			var err error
			profile.ID, err = settings.NewConnectionID()
			if err != nil {
				s.showVaultError(err)
				return
			}
		}
		savePassword := form.GetFormItem(connectionSavePassword).(*tview.Checkbox).IsChecked()
		// Keep the public profile marked as not saved until encrypted persistence
		// succeeds. A failure can leave updated public metadata, never plaintext.
		profile.SavedPassword = false
		if index == len(next) {
			next = append(next, profile)
		} else {
			next[index] = profile
		}
		writer, err := settings.BeginConnectionsUpdate(path, profiles)
		if err != nil {
			s.showConnectionSaveError(err)
			return
		}
		defer writer.Close()
		if err := writer.Save(next); err != nil {
			form.SetTitle(" " + fmt.Sprintf(s.tr("connection_save_error"), err) + " ")
			return
		}
		if savePassword {
			if err := s.vault.Put(profile.ID, profile.Binding(), config.Password); err != nil {
				if restoreErr := writer.Save(profiles); restoreErr != nil {
					form.SetTitle(s.tr("vault_profile_error"))
				} else {
					s.showVaultError(err)
				}
				return
			}
			next[index].SavedPassword = true
			if err := writer.Save(next); err != nil {
				profiles = next
				form.SetTitle(s.tr("vault_profile_error"))
				return
			}
		} else if index < len(profiles) && profiles[index].SavedPassword {
			if s.vault != nil && s.vault.Unlocked() {
				if err := s.vault.Delete(profile.ID); err != nil {
					if restoreErr := writer.Save(profiles); restoreErr != nil {
						form.SetTitle(s.tr("vault_profile_error"))
					} else {
						s.showVaultError(err)
					}
					return
				}
			}
		}
		profiles = next
		// Updating the selection after save must keep the typed password in memory.
		dropdown := selector.GetFormItem(0).(*choiceField)
		s.connectionOptions()
		dropdown.SetSelectedFunc(nil).SetCurrentOption(index + 1).SetSelectedFunc(selectProfile)
		form.GetFormItem(connectionAlias).(*tview.InputField).SetText(alias)
		message := "connection_saved"
		if savePassword {
			message = "connection_secret_saved"
		}
		form.SetTitle(" " + s.tr(message) + " ")
	}
	s.saveConnection = func() {
		if loadErr != nil {
			saveProfile()
			return
		}
		if form.GetFormItem(connectionSavePassword).(*tview.Checkbox).IsChecked() {
			config, ok := readConfig()
			if !ok {
				return
			}
			if config.Password == "" {
				form.SetTitle(s.tr("vault_password_required"))
				return
			}
			if strings.TrimSpace(form.GetFormItem(connectionAlias).(*tview.InputField).GetText()) == "" {
				form.SetTitle(s.tr("alias_required"))
				return
			}
			s.ensurePasswords(saveProfile)
		} else {
			alias := strings.TrimSpace(form.GetFormItem(connectionAlias).(*tview.InputField).GetText())
			for _, p := range profiles {
				if p.Alias == alias && p.SavedPassword {
					s.ensurePasswords(saveProfile)
					return
				}
			}
			saveProfile()
		}
	}
	form.AddButton("", s.connectAction).AddButton("", s.closeConnection).AddButton("", s.saveConnection)
	s.clickable(form.GetButton(0))
	s.clickable(form.GetButton(1))
	s.clickable(form.GetButton(2))
	form.SetBorder(true)
	s.styleForm(form)
	s.localizeConnection()
	if loadErr != nil {
		form.SetTitle(" " + fmt.Sprintf(s.tr("connection_load_error"), loadErr) + " ")
	}
	gap := s.newBox().SetBackgroundColor(s.colors().surface)
	content := s.newFlex().SetDirection(tview.FlexRow).AddItem(selector, 5, 0, false).AddItem(gap, 1, 0, false).AddItem(form, 0, 1, true)
	root := s.formPopup(content, 90, 30, func(width, height int) {
		spacing := 1
		if height < 29 {
			spacing = 0
		}
		form.SetItemPadding(spacing)
		profileHeight := 5
		if height < 24 {
			profileHeight = 3
			selector.SetBorderPadding(0, 0, 2, 2)
			form.SetBorderPadding(0, 0, 2, 2)
		} else {
			selector.SetBorderPadding(1, 1, 2, 2)
			form.SetBorderPadding(1, 1, 2, 2)
		}
		content.ResizeItem(selector, profileHeight, 0)
		gapHeight := 1
		if height < 20 {
			gapHeight = 0
		}
		content.ResizeItem(gap, gapHeight, 0)
		if width < 72 {
			form.GetButton(0).SetLabel(s.tr("connect"))
			form.GetButton(1).SetLabel(s.tr("cancel"))
			form.GetButton(2).SetLabel(s.tr("save"))
		} else {
			form.GetButton(0).SetLabel(s.tr("connect") + " (Alt+C)")
			form.GetButton(1).SetLabel(s.tr("cancel") + " (Esc)")
			form.GetButton(2).SetLabel(s.tr("save") + " (Alt+S)")
		}
	})
	s.pages.AddPage("connect", root, true, true)
	s.app.SetFocus(form)
}

func (s *screen) showConnectionSaveError(err error) {
	message := fmt.Sprintf(s.tr("connection_save_error"), err)
	if errors.Is(err, settings.ErrConnectionsChanged) {
		message = s.tr("connections_changed")
	} else if errors.Is(err, settings.ErrConnectionsBusy) {
		message = s.tr("connections_busy")
	}
	s.connectForm.SetTitle(" " + message + " ")
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
