package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/tmlbl/edgar"
)

var pullInterval int64 = 10

func main() {
	flag.Int64Var(&pullInterval, "interval", 10,
		"Seconds to pause between data pulls")
	flag.Parse()

	for {
		tables, err := edgar.Latest13F()
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Printf("Pulled %d tables\n", len(tables))
		time.Sleep(time.Second * time.Duration(pullInterval))
	}
}
