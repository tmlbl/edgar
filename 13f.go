// Package edgar contains utilities for retrieving and parsing data from the
// SEC EDGAR online database.
package edgar

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/mmcdole/gofeed"
)

var url13f = "https://www.sec.gov/cgi-bin/browse-edgar?" +
	"action=getcurrent&CIK=&type=13F&output=atom"

// Latest13F pulls the latest 13F filings from an EDGAR RSS feed.
// This typically only contains < 10 of the most recent filings.
func Latest13F() ([]InformationTable, error) {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(url13f)
	tables := []InformationTable{}

	if err != nil {
		return nil, err
	}
	for _, it := range feed.Items {
		fmt.Println(it.Link)
		link := strings.Replace(it.Link, "-index.htm", ".txt", 1)
		resp, err := http.Get(link)
		if err != nil {
			return nil, err
		}
		body, _ := ioutil.ReadAll(resp.Body)
		table, err := parse13f(string(body))
		if err != nil {
			return nil, err
		}
		tables = append(tables, *table)
	}
	return tables, nil
}

func extractID(guid string) string {
	i := strings.Index(guid, "accession-number")
	return guid[i+len("accession-number")+1:]
}

type InfoTable struct {
	Issuer     string   `xml:"nameOfIssuer"`
	ClassTitle string   `xml:"titleOfClass"`
	CUSIP      string   `xml:"cusip"`
	Value      int      `xml:"value"`
	Position   Position `xml:"shrsOrPrnAmt"`
}

// Position represents SH / PRN
// Shares or principal amt
type Position struct {
	Amount int    `xml:"sshPrnamt"`
	Type   string `xml:"sshPrnamtType"`
}

type InformationTable struct {
	InfoTable []InfoTable `xml:"infoTable"`
}

func parse13f(data string) (*InformationTable, error) {
	lines := strings.Split(data, "\n")
	cap := false
	xlines := []string{}

	for _, ln := range lines {
		if strings.Index(ln, "informationTable") != -1 {
			cap = true
		}

		if cap {
			xlines = append(xlines, ln)
		}

		if strings.Index(ln, "/informationTable") != -1 {
			cap = false
		}
	}

	xdata := strings.Join(xlines, "\n")
	table := InformationTable{}
	err := xml.Unmarshal([]byte(xdata), &table)
	if err != nil {
		return nil, err
	}
	return &table, nil
}
