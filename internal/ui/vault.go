package ui

import (
	"bytes"
	"errors"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/credentials"
	"lazyora/internal/settings"
)

func (s *screen) passwordStore() error {
	if s.vault != nil {
		return nil
	}
	store, err := credentials.Open(credentials.Path(s.prefsPath, settings.ConnectionsPath(s.prefsPath)))
	if err != nil {
		return err
	}
	s.vault = store
	return nil
}
func (s *screen) lockPasswords() {
	if s.masterForm != nil {
		for i := 1; i < s.masterForm.GetFormItemCount(); i++ {
			s.masterForm.GetFormItem(i).(*tview.InputField).SetText("")
		}
	}
	if s.vault != nil {
		s.vault.Lock()
	}
	s.vault = nil
	if s.clearConnectionSecret != nil {
		s.clearConnectionSecret()
	}
}
func (s *screen) vaultError(err error) string {
	switch {
	case errors.Is(err, credentials.ErrUnlock):
		return s.tr("vault_wrong")
	case errors.Is(err, credentials.ErrWeak):
		return s.tr("vault_weak")
	case errors.Is(err, credentials.ErrChanged), errors.Is(err, credentials.ErrBusy):
		return s.tr("vault_changed")
	default:
		return s.tr("vault_error")
	}
}
func (s *screen) ensurePasswords(done func()) {
	if err := s.passwordStore(); err != nil {
		s.showVaultError(err)
		return
	}
	if s.vault.Unlocked() {
		done()
		return
	}
	mode := "unlock"
	if !s.vault.Exists() {
		mode = "create"
	}
	s.masterDialog(mode, done)
}
func (s *screen) showVaultError(err error) {
	if s.connectForm != nil {
		s.connectForm.SetTitle(s.vaultError(err))
	}
	if s.vaultForm != nil {
		s.vaultForm.SetTitle(s.vaultError(err))
	}
}

func (s *screen) masterDialog(mode string, done func()) {
	if s.masterForm != nil || s.vaultBusy {
		return
	}
	focus := s.app.GetFocus()
	form := tview.NewForm().SetItemPadding(0)
	s.masterForm = form
	form.AddTextView("", s.tr("vault_master_hint"), 0, 4, true, false)
	form.AddPasswordField(s.tr("master_password"), "", 40, '*', nil)
	if mode != "unlock" {
		form.AddPasswordField(s.tr("master_confirm"), "", 40, '*', nil)
	}
	closeDialog := func() {
		for i := 1; i < form.GetFormItemCount(); i++ {
			form.GetFormItem(i).(*tview.InputField).SetText("")
		}
		s.pages.RemovePage("master")
		s.masterForm = nil
		s.masterSubmit = nil
		s.masterCancel = nil
		s.app.SetFocus(focus)
	}
	s.masterCancel = closeDialog
	s.masterSubmit = func() {
		if s.vaultBusy {
			return
		}
		master := []byte(form.GetFormItem(1).(*tview.InputField).GetText())
		if mode != "unlock" {
			confirmation := []byte(form.GetFormItem(2).(*tview.InputField).GetText())
			matches := bytes.Equal(master, confirmation)
			clear(confirmation)
			if !matches {
				clear(master)
				form.SetTitle(s.tr("master_mismatch"))
				return
			}
			if err := credentials.CheckMaster(master); err != nil {
				clear(master)
				form.SetTitle(s.vaultError(err))
				return
			}
		}
		for i := 1; i < form.GetFormItemCount(); i++ {
			form.GetFormItem(i).(*tview.InputField).SetText("")
		}
		s.vaultBusy = true
		form.SetTitle(s.tr("vault_working"))
		// No UI state is read or written by the KDF worker. While it runs, input
		// and mouse actions are blocked; cleanup waits for it before locking keys.
		store := s.vault
		workDone := make(chan struct{})
		s.vaultWorkDone = workDone
		go func() {
			var err error
			switch mode {
			case "create":
				err = store.Create(master)
			case "change":
				err = store.ChangeMaster(master)
			default:
				err = store.Unlock(master)
			}
			clear(master)
			close(workDone)
			s.app.QueueUpdateDraw(func() {
				s.vaultBusy = false
				if err != nil {
					form.SetTitle(s.vaultError(err))
					focusForm(s, form, 1)
					return
				}
				closeDialog()
				done()
			})
		}()
	}
	form.AddButton(s.tr("proceed"), s.masterSubmit).AddButton(s.tr("cancel"), closeDialog)
	for i := 0; i < form.GetButtonCount(); i++ {
		clickable(form.GetButton(i))
	}
	form.SetBorder(true).SetTitle(s.tr("vault_" + mode))
	styleForm(form)
	s.pages.AddPage("master", form, true, true)
	focusForm(s, form, 1)
}

func (s *screen) passwordManager() {
	if s.vaultForm != nil || s.vaultBusy {
		return
	}
	s.vaultFocus = s.app.GetFocus()
	form := tview.NewForm().SetItemPadding(0)
	s.vaultForm = form
	form.AddTextView("", s.tr("vault_hint"), 0, 7, true, false)
	refresh := func() {
		state := "vault_locked"
		if s.vault != nil && s.vault.Unlocked() {
			state = "vault_unlocked"
		}
		form.SetTitle(s.tr("passwords") + " — " + s.tr(state))
	}
	closeManager := func() {
		s.pages.RemovePage("vault")
		s.vaultForm = nil
		s.app.SetFocus(s.vaultFocus)
		s.vaultFocus = nil
	}
	s.closeVault = closeManager
	form.SetCancelFunc(closeManager)
	form.AddButton(s.tr("vault_unlock"), func() { s.ensurePasswords(refresh) }).
		AddButton(s.tr("vault_lock"), func() { s.lockPasswords(); refresh() }).
		AddButton(s.tr("vault_change"), func() { s.ensurePasswords(func() { s.masterDialog("change", refresh) }) }).
		AddButton(s.tr("vault_reset"), func() {
			s.showProgramConfirm(s.tr("vault_reset_confirm"), func() {
				// Re-open even a malformed store, preserving its original bytes for the
				// optimistic write check. Reset is explicitly destructive and confirmed.
				store, err := credentials.Open(credentials.Path(s.prefsPath, settings.ConnectionsPath(s.prefsPath)))
				if store == nil {
					s.showVaultError(err)
					return
				}
				if err = store.Reset(); err != nil {
					s.showVaultError(err)
					return
				}
				s.lockPasswords()
				path := settings.ConnectionsPath(s.prefsPath)
				profiles, err := settings.LoadConnections(path)
				if err != nil {
					s.showVaultError(err)
					return
				}
				if len(profiles) > 0 {
					for i := range profiles {
						profiles[i].SavedPassword = false
					}
					if err = settings.SaveConnections(path, profiles); err != nil {
						s.showVaultError(err)
						return
					}
				}
				if s.reloadConnectionProfiles != nil {
					if err = s.reloadConnectionProfiles(); err != nil {
						s.showVaultError(err)
						return
					}
				}
				refresh()
			})
		}).AddButton(s.tr("close"), closeManager)
	for i := 0; i < form.GetButtonCount(); i++ {
		clickable(form.GetButton(i))
	}
	form.SetBorder(true)
	styleForm(form)
	refresh()
	s.pages.AddPage("vault", form, true, true)
	focusForm(s, form, 1)
}

func (s *screen) vaultKey(event *tcell.EventKey, page string) *tcell.EventKey {
	if event.Key() == tcell.KeyF1 {
		s.showHelp()
		return nil
	}
	if event.Key() == tcell.KeyCtrlC {
		return nil
	}
	if event.Key() == tcell.KeyEscape {
		if page == "master" {
			s.masterCancel()
		} else {
			s.closeVault()
		}
		return nil
	}
	if page == "master" && (event.Key() == tcell.KeyCtrlS || (event.Key() == tcell.KeyEnter && event.Modifiers()&(tcell.ModCtrl|tcell.ModAlt) != 0)) {
		s.masterSubmit()
		return nil
	}
	return event
}
