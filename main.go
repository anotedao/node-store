package main

import (
	"log"

	"gorm.io/gorm"
)

var conf *Config

var db *gorm.DB

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	conf = initConfig()

	db = initDb()
}
