package proxy

import (
	"testing"

	"github.com/go-mysql-org/go-mysql/client"
)

// mysqlAvailable skips the test if MySQL is not reachable.
func mysqlAvailable(t *testing.T) {
	t.Helper()
	conn, err := client.Connect("127.0.0.1:3306", "root", "", "")
	if err != nil {
		t.Skipf("mysql not available: %v", err)
	}
	conn.Close()
}

// requireMySQL returns a backend connection or skips the test.
func requireMySQL(t *testing.T) *client.Conn {
	t.Helper()
	conn, err := client.Connect("127.0.0.1:3306", "root", "", "")
	if err != nil {
		t.Skipf("mysql not available: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}
