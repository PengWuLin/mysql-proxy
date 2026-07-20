package proxy

import (
	"github.com/go-mysql-org/go-mysql/client"
	"github.com/go-mysql-org/go-mysql/mysql"
	"github.com/go-mysql-org/go-mysql/server"
)

// Ensure Handler implements server.Handler.
var _ server.Handler = (*Handler)(nil)

// Handler implements server.Handler by forwarding all commands to a backend
// MySQL connection.
type Handler struct {
	backend *client.Conn
}

// NewHandler creates a Handler that forwards to the given backend connection.
func NewHandler(backend *client.Conn) *Handler {
	return &Handler{backend: backend}
}

func (h *Handler) UseDB(dbName string) error {
	return h.backend.UseDB(dbName)
}

func (h *Handler) HandleQuery(query string) (*mysql.Result, error) {
	return h.backend.Execute(query)
}

func (h *Handler) HandleFieldList(table string, fieldWildcard string) ([]*mysql.Field, error) {
	return h.backend.FieldList(table, fieldWildcard)
}

func (h *Handler) HandleStmtPrepare(query string) (int, int, any, error) {
	stmt, err := h.backend.Prepare(query)
	if err != nil {
		return 0, 0, nil, err
	}
	return stmt.ParamNum(), stmt.ColumnNum(), stmt, nil
}

func (h *Handler) HandleStmtExecute(context any, query string, args []any) (*mysql.Result, error) {
	stmt := context.(*client.Stmt)
	return stmt.Execute(args...)
}

func (h *Handler) HandleStmtClose(context any) error {
	stmt := context.(*client.Stmt)
	return stmt.Close()
}

func (h *Handler) HandleOtherCommand(cmd byte, data []byte) error {
	return nil
}
