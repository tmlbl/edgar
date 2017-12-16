// Package edgar contains utilities for retrieving and parsing data from the
// SEC EDGAR online database.
package edgar

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

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
		info, err := parse13fheader(string(body))
		if err != nil {
			return nil, err
		}
		table.ReportInfo = info
		tables = append(tables, *table)
	}
	return tables, nil
}

func extractID(guid string) string {
	i := strings.Index(guid, "accession-number")
	return guid[i+len("accession-number")+1:]
}

type InfoTable struct {
	Issuer       string       `xml:"nameOfIssuer"`
	ClassTitle   string       `xml:"titleOfClass"`
	CUSIP        string       `xml:"cusip"`
	Value        int          `xml:"value"`
	PositionInfo PositionInfo `xml:"shrsOrPrnAmt"`
}

// PositionInfo represents SH / PRN
// Shares or principal amt
type PositionInfo struct {
	Amount int    `xml:"sshPrnamt"`
	Type   string `xml:"sshPrnamtType"`
}

type InformationTable struct {
	ReportInfo *ReportInfo
	InfoTable  []InfoTable `xml:"infoTable"`
}

// Position is the structure we ultimately want to store. It contains data
// relevant to after-the-fact analysis of position data.
type Position struct {
	DocumentID     string `gorm:"primary_key"`
	CompanyID      int
	CUSIP          string `gorm:"primary_key"`
	Value          int
	PositionAmount int
	PositionType   string
	DateObserved   time.Time
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

type ReportInfo struct {
	AccessionNumber string
	DateFiled       time.Time
	CompanyIRS      int
}

const dateFormat = "20060102"

func extractval(ln string) string {
	return strings.TrimSpace(strings.Split(ln, ":")[1])
}

func parse13fheader(data string) (*ReportInfo, error) {
	info := ReportInfo{}
	lines := strings.Split(data, "\n")

	for _, ln := range lines {
		// Document ID
		if strings.Contains(ln, "ACCESSION NUMBER:") {
			info.AccessionNumber = extractval(ln)
		}
		// Filing date
		if strings.Contains(ln, "FILED AS OF DATE:") {
			str := extractval(ln)
			t, err := time.Parse(dateFormat, str)
			if err != nil {
				return nil, err
			}
			info.DateFiled = t
		}
		// IRS number of filer
		if strings.Contains(ln, "IRS NUMBER:") {
			num, err := strconv.Atoi(extractval(ln))
			if err != nil {
				return nil, err
			}
			info.CompanyIRS = num
		}
	}

	return &info, nil
}

func ToPositionList(table *InformationTable) []Position {
	ps := []Position{}

	for _, t := range table.InfoTable {
		p := Position{
			DocumentID:     table.ReportInfo.AccessionNumber,
			CompanyID:      table.ReportInfo.CompanyIRS,
			CUSIP:          t.CUSIP,
			Value:          t.Value,
			PositionAmount: t.PositionInfo.Amount,
			PositionType:   t.PositionInfo.Type,
			DateObserved:   table.ReportInfo.DateFiled,
		}
		ps = append(ps, p)
	}

	return ps
}
