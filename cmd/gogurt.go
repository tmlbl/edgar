package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/tmlbl/edgar"

	_ "github.com/jinzhu/gorm/dialects/postgres"
)

var pullInterval int64 = 10

var db *gorm.DB

func main() {
	flag.Int64Var(&pullInterval, "interval", 10,
		"Seconds to pause between data pulls")
	flag.Parse()

	handle, err := gorm.Open("postgres",
		"host=localhost port=5432 sslmode=disable user=postgres dbname=edgar")
	if err != nil {
		log.Fatal(err)
	}
	db = handle

	db.AutoMigrate(&edgar.Position{})

	for {
		tables, err := edgar.Latest13F()
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Printf("Pulled %d tables\n", len(tables))
		for _, t := range tables {
			start := time.Now()
			tx := db.Begin()
			positions := edgar.ToPositionList(&t)
			for _, p := range positions {
				tx.Save(&p)
			}
			tx.Commit()
			elapsed := time.Now().Sub(start)
			fmt.Printf("Saved %d records in %s\n",
				len(positions), elapsed)
		}
		time.Sleep(time.Second * time.Duration(pullInterval))
	}
}
