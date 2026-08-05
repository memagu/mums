package db

import (
	"database/sql"
	"fmt"
	"sync"

	"github.com/labstack/echo/v4"
	_ "modernc.org/sqlite"

	"github.com/memagu/mums/internal/config"
)

const ctxKeyDB = "db"

var schemas = []string{
	SchemaUserCredentials,
	SchemaUserProfiles,
	SchemaUserAccounts,
	SchemaUserAccountRoleMappings,
	SchemaPhaddergrupper,
	SchemaPhaddergruppMappings,
	SchemaPhaddergruppInvites,
	SchemaMums,
}

var indexes = []string{
	IndexPhaddergruppMappingsOnPhaddergruppID,
	IndexPhaddergruppInvitesOnPhaddergruppID,
	IndexMumsOnPhaddergruppID,
	IndexMumsOnUserAccountID,
}

type execer interface {
	Exec(query string, args ...any) (sql.Result, error)
	Emit(DBEvent)
}

type Execer = execer

type queryer interface {
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
	Emit(DBEvent)
}

type Queryer = queryer

type DB struct {
	*sql.DB
	sync.RWMutex
	subscribers map[int64]chan DBEvent
	nextID      int64
}

type txWrapper struct {
	tx     *sql.Tx
	events []DBEvent
}

func (tw *txWrapper) Emit(e DBEvent) {
	tw.events = append(tw.events, e)
}

func (tw *txWrapper) Exec(query string, args ...any) (sql.Result, error) {
	return tw.tx.Exec(query, args...)
}

func (tw *txWrapper) Query(query string, args ...any) (*sql.Rows, error) {
	return tw.tx.Query(query, args...)
}

func (tw *txWrapper) QueryRow(query string, args ...any) *sql.Row {
	return tw.tx.QueryRow(query, args...)
}

func NewDB(dbFilePath string) (*DB, error) {
	sqlDB, err := sql.Open("sqlite", dbFilePath)
	if err != nil {
		return nil, err
	}
	if err = sqlDB.Ping(); err != nil {
		return nil, err
	}
	_, err = sqlDB.Exec(`PRAGMA journal_mode = WAL;`)
	if err != nil {
		return nil, fmt.Errorf("enabling WAL failed: %w", err)
	}
	_, err = sqlDB.Exec(`PRAGMA foreign_keys = ON;`)
	if err != nil {
		return nil, fmt.Errorf("enabling foreign_keys failed: %w", err)
	}
	_, err = sqlDB.Exec(fmt.Sprintf(`PRAGMA busy_timeout = %d;`, config.DB.BusyTimeout.Milliseconds()))
	if err != nil {
		return nil, fmt.Errorf("enabling busy_timeout failed: %w", err)
	}

	for _, schema := range schemas {
		if _, err := sqlDB.Exec(schema); err != nil {
			return nil, fmt.Errorf("failed to create schema: %w", err)
		}
	}
	for _, index := range indexes {
		if _, err := sqlDB.Exec(index); err != nil {
			return nil, fmt.Errorf("failed to create index: %w", err)
		}
	}

	db := &DB{
		DB:          sqlDB,
		subscribers: make(map[int64]chan DBEvent),
	}

	return db, nil
}

func WithTx(db *DB, fn func(e execer) error) error {
	tx, err := db.DB.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	tw := &txWrapper{tx: tx}
	if err := fn(tw); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	for _, event := range tw.events {
		db.Emit(event)
	}

	return nil
}

func DBMiddleware(db *DB) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(ctxKeyDB, db)

			return next(c)
		}
	}
}

func GetDB(c echo.Context) *DB {
	db, ok := c.Get(ctxKeyDB).(*DB)
	if !ok {
		panic("ctxKeyDB is not set in context, was DBMiddleware not applied?")
	}
	return db
}
