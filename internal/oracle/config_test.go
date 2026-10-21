package oracle

import (
	"net/url"
	"testing"
)

func TestConnectionModes(t *testing.T) {
	for _, mode := range []string{"service", "sid"} {
		c := Config{Host: "127.0.0.1", Port: 1521, User: "scott", Password: "p@ss:/?#&", Mode: mode, Name: "ORCL"}
		dsn, err := c.DSN()
		if err != nil {
			t.Fatal(err)
		}
		u, err := url.Parse(dsn)
		if err != nil {
			t.Fatal(err)
		}
		if u.Query().Get("LOB FETCH") != "STREAM" || u.Query().Get("LOB READ") != "NONE" {
			t.Fatal("LOB values must use explicit streaming")
		}
		password, _ := u.User.Password()
		if password != c.Password {
			t.Fatal("password was not round-tripped")
		}
		if mode == "sid" && (u.Query().Get("SID") != "ORCL" || u.Path != "/") {
			t.Fatal("SID connection includes a service name")
		}
		if mode == "service" && (u.Path != "/ORCL" || u.Query().Get("SID") != "") {
			t.Fatal("invalid service name connection")
		}
	}
}

func TestInvalidConfig(t *testing.T) {
	for _, c := range []Config{{}, {Host: "db", User: "u", Name: "n", Port: 0, Mode: "service"}, {Host: "db", User: "u", Name: "n", Port: 1521, Mode: "unknown"}} {
		if _, err := c.DSN(); err == nil {
			t.Fatal("expected validation error")
		}
	}
}
