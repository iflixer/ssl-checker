package database

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type Service struct {
	DB *gorm.DB
}

func NewService(mysqlURL string) (*Service, error) {

	db, err := gorm.Open(mysql.Open(mysqlURL), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	//db.Logger = logger.Default.LogMode(logger.LogLevel(4))

	return &Service{
		DB: db,
	}, nil
}
