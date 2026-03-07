package database

import (
	"context"
	"database/sql/driver"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/stdlib"
)

func openPostgresConnector(dsn string) (driver.Connector, error) {
	baseDriver := stdlib.GetDefaultDriver()
	driverCtx, ok := baseDriver.(driver.DriverContext)
	if !ok {
		return nil, fmt.Errorf("pgx stdlib driver does not support DriverContext")
	}

	baseConnector, err := driverCtx.OpenConnector(dsn)
	if err != nil {
		return nil, err
	}

	return &postgresConnector{base: baseConnector}, nil
}

type postgresConnector struct {
	base driver.Connector
}

func (c *postgresConnector) Connect(ctx context.Context) (driver.Conn, error) {
	conn, err := c.base.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return &postgresConn{Conn: conn}, nil
}

func (c *postgresConnector) Driver() driver.Driver {
	return c.base.Driver()
}

type postgresConn struct {
	driver.Conn
}

func (c *postgresConn) Prepare(query string) (driver.Stmt, error) {
	stmt, err := c.Conn.Prepare(rewritePlaceholders(query))
	if err != nil {
		return nil, err
	}
	return &postgresStmt{Stmt: stmt}, nil
}

func (c *postgresConn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	preparer, ok := c.Conn.(driver.ConnPrepareContext)
	if !ok {
		return c.Prepare(query)
	}

	stmt, err := preparer.PrepareContext(ctx, rewritePlaceholders(query))
	if err != nil {
		return nil, err
	}
	return &postgresStmt{Stmt: stmt}, nil
}

func (c *postgresConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	execer, ok := c.Conn.(driver.ExecerContext)
	if !ok {
		return nil, driver.ErrSkip
	}
	return execer.ExecContext(ctx, rewritePlaceholders(query), normalizeNamedValues(args))
}

func (c *postgresConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	queryer, ok := c.Conn.(driver.QueryerContext)
	if !ok {
		return nil, driver.ErrSkip
	}
	return queryer.QueryContext(ctx, rewritePlaceholders(query), normalizeNamedValues(args))
}

func (c *postgresConn) Exec(query string, args []driver.Value) (driver.Result, error) {
	execer, ok := c.Conn.(driver.Execer)
	if !ok {
		return nil, driver.ErrSkip
	}
	return execer.Exec(rewritePlaceholders(query), normalizeValues(args))
}

func (c *postgresConn) Query(query string, args []driver.Value) (driver.Rows, error) {
	queryer, ok := c.Conn.(driver.Queryer)
	if !ok {
		return nil, driver.ErrSkip
	}
	return queryer.Query(rewritePlaceholders(query), normalizeValues(args))
}

func (c *postgresConn) CheckNamedValue(namedValue *driver.NamedValue) error {
	namedValue.Value = normalizeValue(namedValue.Value)
	if checker, ok := c.Conn.(driver.NamedValueChecker); ok {
		return checker.CheckNamedValue(namedValue)
	}
	return nil
}

func (c *postgresConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	beginner, ok := c.Conn.(driver.ConnBeginTx)
	if !ok {
		return nil, driver.ErrSkip
	}
	return beginner.BeginTx(ctx, opts)
}

func (c *postgresConn) Ping(ctx context.Context) error {
	pinger, ok := c.Conn.(driver.Pinger)
	if !ok {
		return nil
	}
	return pinger.Ping(ctx)
}

func (c *postgresConn) ResetSession(ctx context.Context) error {
	resetter, ok := c.Conn.(driver.SessionResetter)
	if !ok {
		return nil
	}
	return resetter.ResetSession(ctx)
}

func (c *postgresConn) IsValid() bool {
	validator, ok := c.Conn.(driver.Validator)
	if !ok {
		return true
	}
	return validator.IsValid()
}

type postgresStmt struct {
	driver.Stmt
}

func (s *postgresStmt) Exec(args []driver.Value) (driver.Result, error) {
	return s.Stmt.Exec(normalizeValues(args))
}

func (s *postgresStmt) Query(args []driver.Value) (driver.Rows, error) {
	return s.Stmt.Query(normalizeValues(args))
}

func (s *postgresStmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	execer, ok := s.Stmt.(driver.StmtExecContext)
	if !ok {
		return nil, driver.ErrSkip
	}
	return execer.ExecContext(ctx, normalizeNamedValues(args))
}

func (s *postgresStmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	queryer, ok := s.Stmt.(driver.StmtQueryContext)
	if !ok {
		return nil, driver.ErrSkip
	}
	return queryer.QueryContext(ctx, normalizeNamedValues(args))
}

func normalizeNamedValues(args []driver.NamedValue) []driver.NamedValue {
	if len(args) == 0 {
		return args
	}

	normalized := make([]driver.NamedValue, len(args))
	copy(normalized, args)
	for index := range normalized {
		normalized[index].Value = normalizeValue(normalized[index].Value)
	}
	return normalized
}

func normalizeValues(args []driver.Value) []driver.Value {
	if len(args) == 0 {
		return args
	}

	normalized := make([]driver.Value, len(args))
	for index, arg := range args {
		normalized[index] = normalizeValue(arg)
	}
	return normalized
}

func normalizeValue(value any) any {
	switch typed := value.(type) {
	case bool:
		if typed {
			return int64(1)
		}
		return int64(0)
	case *bool:
		if typed == nil {
			return nil
		}
		if *typed {
			return int64(1)
		}
		return int64(0)
	default:
		return value
	}
}

func rewritePlaceholders(query string) string {
	if query == "" || !strings.Contains(query, "?") {
		return query
	}

	var builder strings.Builder
	builder.Grow(len(query) + 8)

	placeholderIndex := 1
	inSingleQuote := false
	inDoubleQuote := false
	inLineComment := false
	inBlockComment := false

	for index := 0; index < len(query); index++ {
		current := query[index]
		var next byte
		if index+1 < len(query) {
			next = query[index+1]
		}

		switch {
		case inLineComment:
			builder.WriteByte(current)
			if current == '\n' {
				inLineComment = false
			}
			continue
		case inBlockComment:
			builder.WriteByte(current)
			if current == '*' && next == '/' {
				builder.WriteByte(next)
				index++
				inBlockComment = false
			}
			continue
		case inSingleQuote:
			builder.WriteByte(current)
			if current == '\'' {
				if next == '\'' {
					builder.WriteByte(next)
					index++
				} else {
					inSingleQuote = false
				}
			}
			continue
		case inDoubleQuote:
			builder.WriteByte(current)
			if current == '"' {
				inDoubleQuote = false
			}
			continue
		}

		switch current {
		case '\'':
			inSingleQuote = true
			builder.WriteByte(current)
		case '"':
			inDoubleQuote = true
			builder.WriteByte(current)
		case '-':
			if next == '-' {
				inLineComment = true
				builder.WriteByte(current)
				builder.WriteByte(next)
				index++
				continue
			}
			builder.WriteByte(current)
		case '/':
			if next == '*' {
				inBlockComment = true
				builder.WriteByte(current)
				builder.WriteByte(next)
				index++
				continue
			}
			builder.WriteByte(current)
		case '?':
			builder.WriteByte('$')
			builder.WriteString(fmt.Sprintf("%d", placeholderIndex))
			placeholderIndex++
		default:
			builder.WriteByte(current)
		}
	}

	return builder.String()
}
