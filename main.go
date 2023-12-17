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

	// price := int64(5)
	// dataTransaction("%s__nodePrice", nil, &price, nil)

	// tier := int64(10)
	// dataTransaction("%s__nodeTier", nil, &tier, nil)

	initBsc()
}
