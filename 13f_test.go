package edgar

import (
	"fmt"
	"io/ioutil"
	"testing"
)

func TestLatest13F(t *testing.T) {
	tables, err := Latest13F()
	if err != nil {
		t.Error(err)
	}

	if len(tables) < 1 {
		t.Errorf("No info tables")
	}

	for _, t := range tables {
		fmt.Println(ToPositionList(&t))
	}
}

func TestExtractID(t *testing.T) {
	guid := "urn:tag:sec.gov,2008:accession-number=0001633207-17-000006"
	id := "0001633207-17-000006"

	if extractID(guid) != id {
		t.Errorf("Extract ID from GUID failed: got %s, expected %s",
			extractID(guid), id)
	}
}

func TestParse13F(t *testing.T) {
	data, _ := ioutil.ReadFile("test/sample_13f.txt")
	parse13f(string(data))
}

func TestParse13FHeader(t *testing.T) {
	data, _ := ioutil.ReadFile("test/sample_13f.txt")
	info, err := parse13fheader(string(data))
	if err != nil {
		t.Error(err)
	}

	acc := "0000950123-17-011329"
	if info.AccessionNumber != acc {
		t.Errorf("Accession number wrong: expected %s, got %s",
			acc, info.AccessionNumber)
	}

	if info.DateFiled.Day() != 13 {
		t.Errorf("Date is wrong: %s", info.DateFiled)
	}

	if info.CompanyIRS != 464937137 {
		t.Errorf("IRS number should be %d, got %d", 464937137, info.CompanyIRS)
	}
}

func TestParseLegacy13F(t *testing.T) {
	data, _ := ioutil.ReadFile("test/legacy_sample_13f.txt")
	info, err := parse13f(string(data))
	if err != nil {
		t.Error(err)
	}

	// Check header parsing
	expectStrEq(t, "0001193125-10-027043", info.ReportInfo.AccessionNumber)
	expectStrEq(t, "0000932859", info.ReportInfo.CompanyCIK)

	// Find the entry for Exxon Mobil
	var exxon *InfoTable
	for _, t := range info.InfoTable {
		if t.CUSIP == "30231G102" {
			exxon = &t
			break
		}
	}
	if exxon == nil {
		t.Errorf("Did not find the expected entry")
	}
	if exxon.Value != 457916.59326 {
		t.Errorf("Position value wrong: expected %f, got %f",
			457916.59326, exxon.Value)
	}
}

func expectStrEq(t *testing.T, expected, actual string) {
	if expected != actual {
		t.Errorf("Expected %s to equal %s\n", actual, expected)
	}
}
