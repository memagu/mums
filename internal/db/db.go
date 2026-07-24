package db

import (
	"database/sql"
	"fmt"
	"sync"

	"github.com/labstack/echo/v4"
	_ "modernc.org/sqlite"
)

const ctxKeyDB = "db"

var schemas = []string{
	SchemaUserCredentials,
	SchemaUserProfiles,
	SchemaUserAccounts,
	SchemaUserAccountRoleMappings,
	SchemaPhaddergrupps,
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
}

type queryer interface {
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

type DB struct {
	*sql.DB
	sync.RWMutex
	subscribers map[int64]chan DBEvent
	nextID      int64
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
