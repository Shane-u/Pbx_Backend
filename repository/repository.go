package repository

import (
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRobotRepository(dsn string) (*Repository, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database")
		return nil, err

	}
	if err = db.AutoMigrate(&ChatRobot{}); err != nil {
		log.Fatal("failed to migrate database")
		return nil, err
	}
	log.Println("database connected successfully")
	return &Repository{db: db}, nil
}
