package proxy

import (
	"testing"
)

func TestHandlerUseDB(t *testing.T) {
	conn := requireMySQL(t)
	h := NewHandler(conn)

	if err := h.UseDB("mysql"); err != nil {
		t.Fatalf("UseDB: %v", err)
	}
}

func TestHandlerQuery(t *testing.T) {
	conn := requireMySQL(t)
	h := NewHandler(conn)

	result, err := h.HandleQuery("SELECT 1 AS n")
	if err != nil {
		t.Fatalf("HandleQuery: %v", err)
	}
	if result.Resultset == nil {
		t.Fatal("expected resultset")
	}
	val, _ := result.Resultset.GetString(0, 0)
	if val != "1" {
		t.Errorf("got %q, want 1", val)
	}
}

func TestHandlerQueryInsert(t *testing.T) {
	conn := requireMySQL(t)
	h := NewHandler(conn)

	h.HandleQuery("CREATE DATABASE IF NOT EXISTS test_proxy")
	h.UseDB("test_proxy")
	h.HandleQuery("DROP TABLE IF EXISTS t")
	h.HandleQuery("CREATE TABLE t (id INT PRIMARY KEY, name VARCHAR(64))")
	defer func() {
		h.HandleQuery("DROP TABLE IF EXISTS t")
	}()

	result, err := h.HandleQuery("INSERT INTO t VALUES (1, 'hello')")
	if err != nil {
		t.Fatalf("INSERT: %v", err)
	}
	if result.AffectedRows != 1 {
		t.Errorf("affected = %d, want 1", result.AffectedRows)
	}

	sel, err := h.HandleQuery("SELECT id, name FROM t")
	if err != nil {
		t.Fatalf("SELECT: %v", err)
	}
	id, _ := sel.Resultset.GetString(0, 0)
	name, _ := sel.Resultset.GetString(0, 1)
	if id != "1" || name != "hello" {
		t.Errorf("got row (%q, %q), want (1, hello)", id, name)
	}
}

func TestHandlerFieldList(t *testing.T) {
	conn := requireMySQL(t)
	h := NewHandler(conn)

	h.UseDB("mysql")
	fields, err := h.HandleFieldList("user", "")
	if err != nil {
		t.Fatalf("HandleFieldList: %v", err)
	}
	if len(fields) == 0 {
		t.Error("expected at least one field")
	}
}

func TestHandlerStmtPrepareExecuteClose(t *testing.T) {
	conn := requireMySQL(t)
	h := NewHandler(conn)

	h.HandleQuery("CREATE DATABASE IF NOT EXISTS test_proxy")
	h.UseDB("test_proxy")
	h.HandleQuery("DROP TABLE IF EXISTS t")
	h.HandleQuery("CREATE TABLE t (id INT PRIMARY KEY)")
	defer func() {
		h.HandleQuery("DROP TABLE IF EXISTS t")
	}()

	params, columns, ctx, err := h.HandleStmtPrepare("INSERT INTO t VALUES (?)")
	if err != nil {
		t.Fatalf("HandleStmtPrepare: %v", err)
	}
	if params != 1 {
		t.Errorf("params = %d, want 1", params)
	}
	if columns != 0 {
		t.Errorf("columns = %d, want 0", columns)
	}

	result, err := h.HandleStmtExecute(ctx, "", []any{int64(42)})
	if err != nil {
		t.Fatalf("HandleStmtExecute: %v", err)
	}
	if result.AffectedRows != 1 {
		t.Errorf("affected = %d, want 1", result.AffectedRows)
	}

	if err := h.HandleStmtClose(ctx); err != nil {
		t.Fatalf("HandleStmtClose: %v", err)
	}
}

func TestHandlerOtherCommand(t *testing.T) {
	conn := requireMySQL(t)
	h := NewHandler(conn)

	if err := h.HandleOtherCommand(0x00, nil); err != nil {
		t.Errorf("HandleOtherCommand should return nil, got %v", err)
	}
}
