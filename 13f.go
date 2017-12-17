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

var urlnew13f = "https://www.sec.gov/cgi-bin/browse-edgar?" +
	"action=getcurrent&CIK=&type=13F&output=atom"

// Latest13F pulls the latest 13F filings from an EDGAR RSS feed.
// This typically only contains < 10 of the most recent filings.
func Latest13F() ([]InformationTable, error) {
	return parsefeed13f(urlnew13f)
}

func parsefeed13f(url string) ([]InformationTable, error) {
	fp := gofeed.NewParser()
	fmt.Println("SEC req", url)
	feed, err := fp.ParseURL(url)
	tables := []InformationTable{}

	if err != nil {
		return nil, err
	}
	for _, it := range feed.Items {
		link := strings.Replace(it.Link, "-index.htm", ".txt", 1)
		fmt.Println("SEC req", link)
		resp, err := http.Get(link)
		if err != nil {
			return nil, err
		}
		body, _ := ioutil.ReadAll(resp.Body)
		table, err := parse13f(string(body))
		if err != nil {
			// return nil, err
			fmt.Println("Error parsing 13F form:", err)
			continue
		}
		info, err := parse13fheader(string(body))
		if err != nil {
			// return nil, err
			fmt.Println("Error parsing 13F form header:", err)
			continue
		}
		table.ReportInfo = info
		tables = append(tables, *table)
		time.Sleep(time.Millisecond * time.Duration(200))
	}
	return tables, nil
}

// Company13Fs will pull all available 13F filings for the company for the given
// CIK number.
func Company13Fs(cik string) ([]InformationTable, error) {
	urltmp := "https://www.sec.gov/cgi-bin/browse-edgar?action=getcompany&" +
		"CIK=%s&type=13F%%25&output=atom"
	url := fmt.Sprintf(urltmp, cik)
	return parsefeed13f(url)
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
	CompanyID      string
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

	if len(xlines) < 1 {
		fmt.Println("I'm confused, so I will try the legacy parser...")
		return parseLegacy13F(data)
	}

	xdata := strings.Join(xlines, "\n")
	table := InformationTable{}
	err := xml.Unmarshal([]byte(xdata), &table)
	if err != nil {
		return nil, err
	}
	return &table, nil
}

// Filters down a slice of lines to those appearing between the XML-style
// delimiter provided.
func extractLines(delim string, lines []string) [][]string {
	xxlines := [][]string{}
	cap := false
	xlines := []string{}

	for _, ln := range lines {
		if strings.Index(ln, "<"+delim) != -1 {
			cap = true
			fmt.Println("cap start", ln)
		}

		if cap {
			xlines = append(xlines, ln)
		}

		if cap && (strings.Index(ln, "</"+delim) != -1) {
			cap = false
			xxlines = append(xxlines, xlines)
			xlines = []string{}
			fmt.Println("cap end", ln)
		}
	}

	return xxlines
}

// The legacy filings are in a whitespace-delimited format. By reading a header
// row, we can get a picture of where we should split subsequent lines to
// isolate individual fields.
func legacyGetFormat(lines []string) ([]int, error) {
	// Find the line with <S> and <C> entries to get the whitespace format of the
	// text report.
	for _, ln := range lines {
		if len(ln) > 3 && ln[0:3] == "<S>" {
			format := []int{}

			for i, c := range ln {
				if c == '<' {
					format = append(format, i)
				}
			}
			return format, nil
		}
	}
	// Error condition - no format found
	return nil, fmt.Errorf("No format header row detected")
}

func parseLegacy13F(data string) (*InformationTable, error) {
	lines := strings.Split(data, "\n")

	tables := extractLines("TABLE", lines)

	for _, t := range tables {
		format, err := legacyGetFormat(t)
		if err != nil {
			panic(err)
		}
		for _, ln := range t {
			fields := []string{}

			if len(ln) < format[len(format)-1] {
				fmt.Println("This line is too short:", ln)
				continue
			}

			for i := range format {
				if i < len(format)-1 {
					f := strings.TrimSpace(ln[format[i]:format[i+1]])
					fields = append(fields, f)
				} else {
					fields = append(fields, strings.TrimSpace(ln[format[i]:]))
				}
			}
			fmt.Println(len(fields))
		}
	}

	return nil, nil
}

type ReportInfo struct {
	AccessionNumber string
	DateFiled       time.Time
	CompanyIRS      int
	CompanyCIK      string
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

		// SEC CIK number of filer
		if strings.Contains(ln, "CENTRAL INDEX KEY:") {
			info.CompanyCIK = extractval(ln)
		}
	}

	return &info, nil
}

func ToPositionList(table *InformationTable) []Position {
	ps := []Position{}

	for _, t := range table.InfoTable {
		p := Position{
			DocumentID:     table.ReportInfo.AccessionNumber,
			CompanyID:      table.ReportInfo.CompanyCIK,
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
