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

	for _, t := range tables {
		fmt.Println(t)
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
