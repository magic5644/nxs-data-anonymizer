package mssql

import (
	"fmt"

	gmssql "gorm.io/driver/sqlserver"
	"gorm.io/gorm"
)

type MSSQL struct {
	client *gorm.DB
}

type Settings struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
}

func Connect(s Settings) (MSSQL, error) {

	dsn := fmt.Sprintf("sqlserver://gorm:%s:%s@%s:%s?database=%s", s.User, s.Password, s.Host, s.Port, s.Database)
	client, err := gorm.Open(gmssql.Open(dsn), &gorm.Config{})
	if err != nil {
		return MSSQL{}, err
	}

	return MSSQL{
		client: client,
	}, nil
}

func (m *MSSQL) Close() error {
	db, _ := m.client.DB()
	return db.Close()
}

func (m *MSSQL) DBCleanup() error {

	tables, err := m.client.Migrator().GetTables()
	if err != nil {
		return fmt.Errorf("drop tables, get tables: %w", err)
	}

	for _, t := range tables {
		if err := m.client.Migrator().DropTable(t); err != nil {
			return fmt.Errorf("drop tables, table `%s`: %w", t, err)
		}
	}

	return nil
}
