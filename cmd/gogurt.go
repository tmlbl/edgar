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
		"host=localhost port=5431 sslmode=disable user=postgres dbname=edgar")
	if err != nil {
		log.Fatal(err)
	}
	db = handle

	db.AutoMigrate(&edgar.Position{})
	db.AutoMigrate(&edgar.Security{})

	for {
		// Pull the latest 13F filings
		tables, err := edgar.Latest13F()
		if err != nil {
			log.Println("Error getting latest 13Fs:", err)
			continue
		}
		fmt.Printf("Retrieved %d 13F filings.\n", len(tables))

		// Get list of unique CIKs from document list
		cikmap := map[string]int{}
		for _, t := range tables {
			cikmap[t.ReportInfo.CompanyCIK] = 1
		}

		for cik := range cikmap {
			// For each company, pull all of their past 13F filings as well
			ctables, err := edgar.Company13Fs(cik)
			if err != nil {
				log.Println("Error retrieving past filing:", err)
				continue
			}
			for _, ct := range ctables {
				insertIfMissing(ct)
			}
		}

		// Insert the newest documents into the db
		for _, t := range tables {
			insertIfMissing(t)
		}

		time.Sleep(time.Second * time.Duration(pullInterval))
	}
}

func insertIfMissing(t edgar.InformationTable) {
	err := db.Where("document_id = ?",
		t.ReportInfo.AccessionNumber).Find(&edgar.Position{}).Error
	// If not in database, process the document
	if err == gorm.ErrRecordNotFound {
		start := time.Now()
		tx := db.Begin()

		positions := edgar.ToPositionList(&t)
		for pix, p := range positions {
			// Look up the security. First check if we already have it.
			err := db.Where("cus_ip = ?", p.CUSIP).Find(&edgar.Security{}).Error
			if err == gorm.ErrRecordNotFound {
				// fmt.Println("Looking up position of type", p.PositionType)
				sec, err := edgar.LookupCUSIP(edgar.FidelityStock, p.CUSIP)
				if err != nil {
					fmt.Printf("Error looking up CUSIP %s for company %s in document %s: %s\n",
						p.CUSIP, t.InfoTable[pix].Issuer, p.DocumentID, err)
				} else {
					// fmt.Println("Got security", sec)
					tx.Save(&sec)
				}
			}
			// Save the position
			tx.Save(&p)
		}
		tx.Commit()
		elapsed := time.Now().Sub(start)
		fmt.Printf("Saved %d records in %s\n",
			len(positions), elapsed)
	}
}
