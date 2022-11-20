package database

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	_ "github.com/jackc/pgx/v4/stdlib" // postgres driver
)

type DB struct {
	host     string
	port     int
	user     string
	password string
	dbname   string
	sslmode  string
	sql      *sqlx.DB
}

func New(host string, port int, user, password, dbname string, sslmode string) *DB {
	return &DB{
		host:     host,
		port:     port,
		user:     user,
		password: password,
		dbname:   dbname,
		sslmode:  sslmode,
	}
}

func (d *DB) Connect() error {
	psqlconn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.host, d.port, d.user, d.password, d.dbname, d.sslmode,
	)
	db, err := sqlx.Open("pgx", psqlconn)
	if err != nil {
		return err
	}
	d.sql = db

	if err := d.sql.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}
	d.sql.SetMaxIdleConns(10)
	d.sql.SetConnMaxIdleTime(5 * time.Minute)
	return nil
}

func (d *DB) Close() error {
	return d.sql.Close()
}
