package main

import (
	"log"

	"github.com/anonutopia/gowaves"
	"gorm.io/gorm"
)

var conf *Config

var db *gorm.DB

var anoteAddress string

var anc *gowaves.WavesNodeClient

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	conf = initConfig()

	db = initDb()

	anc = initAnote()

	initBsc()
}
