package settings

import (
	"path/filepath"
	"testing"
)

func TestConnectionEnvironmentIsPublicAndDoesNotChangeCredentialBinding(t *testing.T) {
	p := Connection{Alias: "primary", Host: "localhost", Port: 1521, User: "scott", Mode: "service", Name: "db", Environment: "prod"}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	binding := p.Binding()
	p.Environment = "dev"
	p.Alias = "renamed"
	if p.Binding() != binding {
		t.Fatal("cosmetic labels invalidated encrypted credentials")
	}
	path := filepath.Join(t.TempDir(), "connections.json")
	writer, err := BeginConnectionsUpdate(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := writer.Save([]Connection{p}); err != nil {
		t.Fatal(err)
	}
	writer.Close()
	loaded, err := LoadConnections(path)
	if err != nil || loaded[0].Environment != "dev" {
		t.Fatal("environment not saved", err)
	}
	p.Environment = "invalid"
	if p.Validate() == nil {
		t.Fatal("unknown environment accepted")
	}
}
