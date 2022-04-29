package mysql

import (
	"context"
	"fmt"
	"github.com/code-and-chill/auth-api/pkg/logger"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"go.elastic.co/apm/module/apmsql"

	// driver
	_ "go.elastic.co/apm/module/apmsql/mysql"
)

// Result wraps transaction result.
type Result struct {
	Data  interface{}
	Error error
}

// Block is a transaction block.
type Block func(tx *sqlx.Tx, ch chan Result)

// MySQL provides an interface to access MySQL.
type MySQL interface {
	WithTransaction(block Block) (Result, error)
	Get(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	GetNamed(ctx context.Context, dest interface{}, query string, args interface{}) error
	Select(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	SelectNamed(ctx context.Context, dest interface{}, query string, args interface{}) (err error)
	In(ctx context.Context, query string, params map[string]interface{}) (string, []interface{}, error)
	PrepareForWrite(ctx context.Context, query string) (*sqlx.NamedStmt, error)
	PrepareForRead(ctx context.Context, query string) (*sqlx.NamedStmt, error)
	PrepareMode(ctx context.Context, mode Mode) func(string) (*sqlx.NamedStmt, error)
	PrepareBindForWrite(ctx context.Context, query string) (*sqlx.Stmt, error)
	PrepareBindForRead(ctx context.Context, query string) (*sqlx.Stmt, error)
	RebindForWrite(query string) string
	RebindForRead(query string) string
	RebindMode(mode Mode) func(string) string
	Shutdown()
}

type mysql struct {
	master *sqlx.DB
	slave  *sqlx.DB
	logger *logger.Logger
}

// New instantiates a new MySQL.
func New(config Config, logger *logger.Logger) (MySQL, error) {
	var master, slave *sqlx.DB

	master, err := connect(config.Master, logger)
	if err != nil {
		logger.WithField("err", err).Error()
		return nil, errors.WithStack(err)
	}

	if config.Slave != nil {
		var err error
		slave, err = connect(*config.Slave, logger)
		if err != nil {
			logger.WithField("err", err).Error()
			return nil, errors.WithStack(err)
		}
	}
	mysql := &mysql{
		master: master,
		slave:  slave,
		logger: logger,
	}
	return mysql, nil
}

func connect(config ConnectionConfig, logger *logger.Logger) (*sqlx.DB, error) {
	connectionString := buildConnectionString(config)
	logger.WithField("host", config.Host).Debug("initializing mysql connection")
	db, err := apmsql.Open("mysql", connectionString)
	if err != nil {
		logger.WithField("err", err).Error()
		return nil, errors.WithStack(err)
	}
	if err := db.Ping(); err != nil {
		logger.WithField("err", err).Error()
		return nil, errors.WithStack(err)
	}
	logger.WithField("host", config.Host).Debug("connected to mysql")
	db.SetMaxOpenConns(config.ConnectionLimit)
	conn := sqlx.NewDb(db, "mysql")
	conn = conn.Unsafe()
	return conn, nil
}

// Mode represents mysql mode.
type Mode int

const (
	// ModeWrite represents a mode write.
	ModeWrite = Mode(iota + 1)
	// ModeRead represents a mode read.
	ModeRead
)

// GetActiveDB gets the active db for specific mode.
func (m *mysql) GetActiveDB(mode Mode) *sqlx.DB {
	if m.slave == nil {
		return m.master
	}
	switch mode {
	case ModeRead:
		return m.slave
	default:
		return m.master
	}
}

// WithTransaction starts transaction.
func (m *mysql) WithTransaction(block Block) (Result, error) {
	var result Result
	c := make(chan Result)
	tx, err := m.master.Beginx()
	if err != nil {
		m.logger.WithField("err", err).Error()
		return Result{Data: nil, Error: err}, errors.WithStack(err)
	}

	go block(tx, c)

	result = <-c
	if result.Error != nil {
		tx.Tx.Rollback()
	} else {
		tx.Tx.Commit()
	}
	return result, nil
}

// Get gets data from database.
func (m *mysql) Get(ctx context.Context, dest interface{}, query string, args ...interface{}) (err error) {
	return m.GetActiveDB(ModeRead).GetContext(ctx, dest, query, args...)
}

// GetNamed gets single data from database using named parameters.
func (m *mysql) GetNamed(ctx context.Context, dest interface{}, query string, args interface{}) error {
	stmt, err := m.GetActiveDB(ModeRead).PrepareNamedContext(ctx, query)
	if err != nil {
		return errors.WithStack(err)
	}
	defer stmt.Close()
	return stmt.GetContext(ctx, dest, args)
}

// SelectNamed gets multiple data from database using named parameters.
func (m *mysql) SelectNamed(ctx context.Context, dest interface{}, query string, args interface{}) error {
	stmt, err := m.GetActiveDB(ModeRead).PrepareNamedContext(ctx, query)
	if err != nil {
		return errors.WithStack(err)
	}
	defer stmt.Close()
	return stmt.SelectContext(ctx, dest, args)
}

// Select gets multiple data from database.
func (m *mysql) Select(ctx context.Context, dest interface{}, query string, args ...interface{}) (err error) {
	return m.GetActiveDB(ModeRead).SelectContext(ctx, dest, query, args...)
}

// SelectMode selects mode, whether it should be read or write.
func (m *mysql) SelectMode(ctx context.Context, mode Mode) func(dest interface{}, query string, args ...interface{}) (err error) {
	activeDB := m.GetActiveDB(mode)
	return func(dest interface{}, query string, args ...interface{}) (err error) {
		return activeDB.SelectContext(ctx, dest, query, args...)
	}
}

// In Gets data using named parameters and where in query.
func (m *mysql) In(ctx context.Context, query string, params map[string]interface{}) (string, []interface{}, error) {
	query, args, err := sqlx.Named(query, params)
	if err != nil {
		m.logger.WithField("err", err).Error()
		return query, args, errors.WithStack(err)
	}
	return sqlx.In(query, args...)
}

// PrepareForWrite prepares the statements for writing.
func (m *mysql) PrepareForWrite(ctx context.Context, query string) (*sqlx.NamedStmt, error) {
	return m.GetActiveDB(ModeWrite).PrepareNamedContext(ctx, query)
}

// PrepareForRead prepares the statements for reading.
func (m *mysql) PrepareForRead(ctx context.Context, query string) (*sqlx.NamedStmt, error) {
	return m.GetActiveDB(ModeRead).PrepareNamedContext(ctx, query)
}

// PrepareMode prepares statements using selected mode.
func (m *mysql) PrepareMode(ctx context.Context, mode Mode) func(string) (*sqlx.NamedStmt, error) {
	activeDb := m.GetActiveDB(mode)
	return func(query string) (*sqlx.NamedStmt, error) {
		return activeDb.PrepareNamedContext(ctx, query)
	}
}

// PrepareBindForWrite prepares bind for writing.
func (m *mysql) PrepareBindForWrite(ctx context.Context, query string) (*sqlx.Stmt, error) {
	return m.GetActiveDB(ModeWrite).PreparexContext(ctx, query)
}

// PrepareBindForRead prepares bind for reading.
func (m *mysql) PrepareBindForRead(ctx context.Context, query string) (*sqlx.Stmt, error) {
	return m.GetActiveDB(ModeRead).PreparexContext(ctx, query)
}

// RebindForWrite rebinds for writing.
func (m *mysql) RebindForWrite(query string) string {
	return m.GetActiveDB(ModeWrite).Rebind(query)
}

// RebindForRead rebinds for reading.
func (m *mysql) RebindForRead(query string) string {
	return m.GetActiveDB(ModeRead).Rebind(query)
}

// RebindMode rebinds using selected mode.
func (m *mysql) RebindMode(mode Mode) func(string) string {
	activeDB := m.GetActiveDB(mode)
	return func(query string) string {
		return activeDB.Rebind(query)
	}
}

// Shutdown shuts the server down.
func (m *mysql) Shutdown() {
	if m.master != nil {
		m.logger.Debug("closing master mysql database connection")
		m.master.Close()
	}
	if m.slave != nil {
		m.logger.Debug("closing slave mysql database connection")
		m.slave.Close()
	}
}

func buildConnectionString(config ConnectionConfig) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true", config.Username, config.Password,
		config.Host, config.Port, config.Database)
}
