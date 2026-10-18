package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lazyora/internal/settings"
)

func fillConnection(s *screen, alias, mode string) {
	for i, value := range map[int]string{connectionHost: "db.example", connectionPort: "1522", connectionUser: "scott", connectionPassword: "secret-only-in-memory", connectionName: "ORCL", connectionAlias: alias} {
		s.connectForm.GetFormItem(i).(*tview.InputField).SetText(value)
	}
	index := 0
	if mode == "sid" {
		index = 1
	}
	s.connectForm.GetFormItem(connectionMode).(*choiceField).SetCurrentOption(index)
}

func TestConnectionProfilesPersistUpdateAndClearPassword(t *testing.T) {
	s := shortcutScreen(t)
	s.connectionForm()
	fillConnection(s, " 개발 DB ", "sid")
	s.key(tcell.NewEventKey(tcell.KeyCtrlS, 0, tcell.ModCtrl))
	if len(s.jobs) != 0 {
		t.Fatal("saving a profile attempted a database connection")
	}
	path := settings.ConnectionsPath(s.prefsPath)
	profiles, err := settings.LoadConnections(path)
	if err != nil || len(profiles) != 1 || profiles[0].Alias != "개발 DB" || profiles[0].Mode != "sid" || profiles[0].Port != 1522 {
		t.Fatalf("profile was not saved: %+v %v", profiles, err)
	}
	data, err := os.ReadFile(path)
	if err != nil || strings.Contains(string(data), "secret-only-in-memory") || strings.Contains(strings.ToLower(string(data)), "password") {
		t.Fatal("password was written to disk")
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0600 {
		t.Fatalf("connection file permissions: %v", info.Mode())
	}
	if s.connectForm.GetFormItem(connectionPassword).(*tview.InputField).GetText() != "secret-only-in-memory" {
		t.Fatal("saving discarded the password needed for connecting")
	}
	s.connectForm.GetFormItem(connectionHost).(*tview.InputField).SetText("updated.example")
	s.key(tcell.NewEventKey(tcell.KeyRune, 's', tcell.ModAlt))
	profiles, _ = settings.LoadConnections(path)
	if len(profiles) != 1 || profiles[0].Host != "updated.example" {
		t.Fatal("saving the same alias did not update the profile")
	}
	fillConnection(s, "운영 DB", "service")
	s.connectForm.GetButton(2).InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), func(tview.Primitive) {})
	profiles, _ = settings.LoadConnections(path)
	if len(profiles) != 2 {
		t.Fatal("second alias did not create another profile")
	}
	s.closeConnection()
	// Reopen from disk with a password supplied in the environment. Selecting a
	// saved profile must not reuse that password for a different database.
	t.Setenv("LAZYORA_PASSWORD", "environment-secret")
	s.connectionForm()
	s.connectProfiles.GetFormItem(0).(*choiceField).SetCurrentOption(1)
	if s.connectForm.GetFormItem(connectionPassword).(*tview.InputField).GetText() != "" || s.connectForm.GetFormItem(connectionHost).(*tview.InputField).GetText() != "updated.example" {
		t.Fatal("saved profile was not restored or password was reused")
	}
	index, _ := s.connectForm.GetFormItem(connectionMode).(*choiceField).GetCurrentOption()
	if index != 1 {
		t.Fatal("saved SID mode was not restored")
	}
	s.connectForm.GetFormItem(connectionPassword).(*tview.InputField).SetText("another-secret")
	s.settingsForm()
	s.prefsForm.GetFormItem(0).(*choiceField).SetCurrentOption(1)
	s.savePrefs()
	if s.connectForm.GetFormItem(connectionPassword).(*tview.InputField).GetText() != "another-secret" {
		t.Fatal("language change discarded connection inputs")
	}
	s.connectProfiles.GetFormItem(0).(*choiceField).SetCurrentOption(2)
	if s.connectForm.GetFormItem(connectionPassword).(*tview.InputField).GetText() != "" {
		t.Fatal("switching aliases reused a password")
	}
	index, _ = s.connectForm.GetFormItem(connectionMode).(*choiceField).GetCurrentOption()
	if index != 0 {
		t.Fatal("saved Service Name mode was not restored")
	}
}

func TestConnectionProfileSaveFailuresKeepInputsAndFile(t *testing.T) {
	s := shortcutScreen(t)
	s.connectionForm()
	fillConnection(s, "", "service")
	s.saveConnection()
	if !strings.Contains(s.connectForm.GetTitle(), "alias") {
		t.Fatal("empty alias was accepted")
	}
	fillConnection(s, "dev", "service")
	s.saveConnection()
	path := settings.ConnectionsPath(s.prefsPath)
	before, _ := os.ReadFile(path)
	s.connectForm.GetFormItem(connectionPort).(*tview.InputField).SetText("0")
	s.saveConnection()
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("invalid input overwrote the saved profile")
	}
	s.closeConnection()
	// Simulate a write failure after a valid profile file was loaded.
	s.connectionForm()
	fillConnection(s, "dev", "service")
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	s.saveConnection()
	if !strings.Contains(s.connectForm.GetTitle(), "Could not save") || s.connectForm.GetFormItem(connectionPassword).(*tview.InputField).GetText() != "secret-only-in-memory" {
		t.Fatal("write failure was not reported or discarded inputs")
	}
	s.closeConnection()
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{broken"), 0600); err != nil {
		t.Fatal(err)
	}
	s.connectionForm()
	fillConnection(s, "dev", "service")
	s.saveConnection()
	after, _ = os.ReadFile(path)
	if string(after) != "{broken" || !strings.Contains(s.connectForm.GetTitle(), "Could not save") {
		t.Fatal("corrupt profiles were silently overwritten")
	}
	s.closeConnection()
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	s.connectionForm()
	fillConnection(s, "dev", "service")
	s.saveConnection()
	if s.connectForm.GetFormItem(connectionPassword).(*tview.InputField).GetText() != "secret-only-in-memory" {
		t.Fatal("save error discarded inputs")
	}
	if page, _ := s.pages.GetFrontPage(); page != "connect" {
		t.Fatal("failed save closed the form")
	}
}

func TestConnectionProfilePathNeverOverwritesPreferences(t *testing.T) {
	path := filepath.Join(t.TempDir(), "connections.json")
	if settings.ConnectionsPath(path) == path || settings.ConnectionsPath("") != "" {
		t.Fatal("invalid connection profile path")
	}
}

func connectionKey(s *screen, key tcell.Key) {
	event := s.key(tcell.NewEventKey(key, 0, tcell.ModNone))
	if event != nil {
		s.app.GetFocus().InputHandler()(event, func(p tview.Primitive) { s.app.SetFocus(p) })
	}
}

func TestConnectionSectionsAndTabCycle(t *testing.T) {
	s := shortcutScreen(t)
	s.connectionForm()
	if s.connectForm.GetFormItem(0).GetLabel() != s.tr("alias") {
		t.Fatal("alias must be first")
	}
	if s.connectProfiles.GetFormItemCount() != 1 || s.connectForm.GetFormItemCount() != 8 {
		t.Fatal("profile picker not separate from editable fields")
	}
	connectionKey(s, tcell.KeyBacktab)
	if !s.connectProfiles.HasFocus() {
		t.Fatal("Shift+Tab from alias did not reach profiles")
	}
	connectionKey(s, tcell.KeyTab)
	if !s.connectForm.GetFormItem(connectionAlias).HasFocus() {
		t.Fatal("Tab from profiles did not reach alias")
	}
	for i := 0; i < s.connectForm.GetFormItemCount()+s.connectForm.GetButtonCount(); i++ {
		connectionKey(s, tcell.KeyTab)
	}
	if !s.connectProfiles.HasFocus() {
		t.Fatal("forward cycle did not include profiles")
	}
	connectionKey(s, tcell.KeyBacktab)
	if !s.connectForm.GetButton(2).HasFocus() {
		t.Fatal("reverse cycle missed final button")
	}
	s.key(tcell.NewEventKey(tcell.KeyRune, 'r', tcell.ModAlt))
	if !s.connectProfiles.HasFocus() {
		t.Fatal("Alt+R missed profiles")
	}
	s.key(tcell.NewEventKey(tcell.KeyRune, 'a', tcell.ModAlt))
	if !s.connectForm.GetFormItem(connectionAlias).HasFocus() {
		t.Fatal("Alt+A missed alias")
	}
	terminal := tcell.NewSimulationScreen("UTF-8")
	if err := terminal.Init(); err != nil {
		t.Fatal(err)
	}
	defer terminal.Fini()
	terminal.SetSize(80, 24)
	s.pages.SetRect(0, 0, 80, 22)
	s.pages.Draw(terminal)
	_, selectorY, _, selectorHeight := s.connectProfiles.GetRect()
	_, editorY, _, _ := s.connectForm.GetRect()
	if editorY < selectorY+selectorHeight {
		t.Fatal("picker and editor overlap")
	}
	for i := 0; i < s.connectForm.GetButtonCount(); i++ {
		x, y, w, h := s.connectForm.GetButton(i).GetRect()
		if x < 0 || x+w > 80 || y < editorY || y+h > 22 || w == 0 {
			t.Fatalf("button %d offscreen at 80x24: %d,%d %dx%d", i, x, y, w, h)
		}
	}
}

func TestNewConnectionClearsSelectedProfileAndPassword(t *testing.T) {
	t.Setenv("LAZYORA_HOST", "default.example")
	t.Setenv("LAZYORA_PORT", "1521")
	t.Setenv("LAZYORA_USER", "default_user")
	t.Setenv("LAZYORA_NAME", "DEFAULT_SERVICE")
	s := shortcutScreen(t)
	s.connectionForm()
	fillConnection(s, "saved", "sid")
	s.saveConnection()
	s.connectProfiles.GetFormItem(0).(*choiceField).SetCurrentOption(0)
	for index, want := range map[int]string{connectionAlias: "", connectionHost: "default.example", connectionPort: "1521", connectionUser: "default_user", connectionName: "DEFAULT_SERVICE", connectionPassword: ""} {
		if got := s.connectForm.GetFormItem(index).(*tview.InputField).GetText(); got != want {
			t.Fatalf("field %d: %q want %q", index, got, want)
		}
	}
	index, _ := s.connectForm.GetFormItem(connectionMode).(*choiceField).GetCurrentOption()
	if index != 0 {
		t.Fatal("new connection retained saved SID")
	}
}
