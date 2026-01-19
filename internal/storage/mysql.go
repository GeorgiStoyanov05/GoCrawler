package storage

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

type MySQLStorage struct {
	DB *sql.DB
}

func NewMySQLStorage(user, pass, host, dbname string) (*MySQLStorage, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true",
		user, pass, host, dbname,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &MySQLStorage{DB: db}, nil
}
