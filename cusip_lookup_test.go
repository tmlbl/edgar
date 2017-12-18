package edgar

import "testing"

func TestFidelityCUSIP(t *testing.T) {
	result, err := LookupCUSIP(FidelityStock, "88579Y101")
	if err != nil {
		t.Error(err)
	}
	if result.CompanyName != "3M COMPANY" {
		t.Errorf("Failed to extract company name")
	}
	if result.Symbol != "MMM" {
		t.Errorf("Failed to extract the symbol")
	}
}
