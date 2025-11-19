package main

import (
	"fmt"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Image struct {
	ID          uint64     `gorm:"primaryKey;autoIncrement"`
	Key         string     `gorm:"type:varchar(1024);index"`
	URL         string     `gorm:"type:text"`
	ContentType *string    `gorm:"type:varchar(255)"`
	Size        *int64
	ETag        *string    `gorm:"type:varchar(128)"`
	CreatedAt   time.Time  `gorm:"autoCreateTime"`
}

func newDB() (*gorm.DB, error) {
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5432"
	}
	user := os.Getenv("DB_USER")
	pass := os.Getenv("DB_PASSWORD")
	name := os.Getenv("DB_DATABASE")
	ssl := os.Getenv("DB_SSLMODE")
	if ssl == "" {
		ssl = "disable"
	}
	tz := os.Getenv("DB_TIMEZONE")
	if tz == "" {
		tz = "Asia/Tokyo"
	}

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
		host, port, user, pass, name, ssl, tz,
	)

	return gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
}

func autoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&Image{})
}
