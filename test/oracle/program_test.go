package oracle_test

import (
	"context"
	"lazyora/internal/oracle"
	"os"
	"strings"
	"testing"
	"time"
)

func TestOracleProgramIntegration(t *testing.T) {
	if os.Getenv("LAZYORA_INTEGRATION") != "1" {
		t.Skip("run make -C test/oracle test")
	}
	portText := os.Getenv("ORACLE_PORT")
	if portText == "" {
		portText = "1521"
	}
	port, err := oracle.ParsePort(portText)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	s, err := oracle.Open(ctx, oracle.Config{Host: "127.0.0.1", Port: port, User: "lazyora", Password: os.Getenv("APP_USER_PASSWORD"), Mode: "service", Name: "FREEPDB1"})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	objects, err := s.Objects(ctx, "LAZYORA")
	if err != nil {
		t.Fatal(err)
	}
	var function oracle.Object
	for _, object := range objects {
		if !strings.HasPrefix(object.Name, "BROWSER_") {
			continue
		}
		if object.HasSource() {
			source, sourceErr := s.Source(ctx, object)
			if sourceErr != nil || !strings.Contains(source, object.Name) {
				t.Fatalf("%s source: %s %v", object.Kind, source, sourceErr)
			}
			result, compileErr := s.Compile(ctx, object, "")
			if compileErr != nil || result.Object.Status != "VALID" || len(result.Diagnostics) != 0 {
				t.Fatalf("%s recompile: %+v %v", object.Kind, result, compileErr)
			}
		}
		if object.Name == "BROWSER_FUNCTION" {
			function = object
		}
	}
	if function.Name == "" {
		t.Fatal("run browser-fixture first")
	}
	args, err := s.Arguments(ctx, function)
	if err != nil || len(args) != 2 || args[0].Position != 0 || args[1].Name != "P_VALUE" {
		t.Fatalf("function arguments: %+v %v", args, err)
	}
	args, err = s.Arguments(ctx, oracle.Object{Owner: "LAZYORA", Name: "BROWSER_PROCEDURE", Kind: "PROCEDURE"})
	if err != nil || len(args) != 2 || args[1].Direction != "OUT" {
		t.Fatalf("procedure arguments: %+v %v", args, err)
	}
	args, err = s.Arguments(ctx, oracle.Object{Owner: "LAZYORA", Name: "BROWSER_PACKAGE", Kind: "PACKAGE"})
	if err != nil || len(args) < 4 {
		t.Fatalf("package overload arguments: %+v %v", args, err)
	}
	original, err := s.Source(ctx, function)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := s.Compile(context.Background(), function, original); err != nil {
			t.Errorf("restore fixture: %v", err)
		}
	}()
	invalid := strings.Replace(original, "RETURN p_value;", "RETURN missing_browser_name;", 1)
	if invalid == original {
		t.Fatal("fixture body not found")
	}
	result, compileErr := s.Compile(ctx, function, invalid)
	if result.Object.Status != "INVALID" || len(result.Diagnostics) == 0 || result.Diagnostics[0].Line == 0 || result.Diagnostics[0].Position == 0 {
		t.Fatalf("expected compilation diagnostics: %+v %v", result, compileErr)
	}
	result, err = s.Compile(ctx, function, original)
	if err != nil || !result.Applied || result.Object.Status != "VALID" || len(result.Diagnostics) != 0 {
		t.Fatalf("repair: %+v %v", result, err)
	}
}
