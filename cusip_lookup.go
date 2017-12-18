package edgar

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/yhat/scrape"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Automates requests to the public Fidelity page that allows looking up of
// securities by their CUSIP.

// Example query:
// https://quotes.fidelity.com/mmnet/SymLookup.phtml?reqforlookup=
// REQUESTFORLOOKUP&productid=mmnet&isLoggedIn=mmnet&rows=50&for=stock
// &by=cusip&criteria=88579Y101&submit=Search

const cusipBaseURL = "https://quotes.fidelity.com/mmnet/SymLookup.phtml"

type FidelitySecurityType string

const (
	FidelityStock      FidelitySecurityType = "stock"
	FidelityMutualFund FidelitySecurityType = "fund"
	FidelityIndex      FidelitySecurityType = "index"
	FidelityAnnuity    FidelitySecurityType = "annuity"
)

type FidelityLookupResult struct {
	Type        FidelitySecurityType
	CompanyName string
	Symbol      string
}

type Security struct {
	Type        string
	CompanyName string
	Symbol      string
	CUSIP       string `gorm:"primary_key"`
}

func cusipParams(t FidelitySecurityType, cusip string) map[string]string {
	return map[string]string{
		"reqforlookup": "REQUESTFORLOOKUP",
		"productid":    "mmnet",
		"isLoggedIn":   "mmnet",
		"rows":         "50",
		"for":          string(t),
		"by":           "cusip",
		"criteria":     cusip,
		"submit":       "Search",
	}
}

func qstring(params map[string]string) string {
	kvs := []string{}
	for k, v := range params {
		s := fmt.Sprintf("%s=%s", url.QueryEscape(k), url.QueryEscape(v))
		kvs = append(kvs, s)
	}
	return fmt.Sprintf("?%s", strings.Join(kvs, "&"))
}

func LookupCUSIP(t FidelitySecurityType, cusip string) (*Security, error) {
	u := cusipBaseURL + qstring(cusipParams(t, cusip))
	resp, err := http.Get(u)
	if err != nil {
		return nil, err
	}

	root, err := html.Parse(resp.Body)
	if err != nil {
		return nil, err
	}

	// Find the company name
	companyNameMatcher := func(n *html.Node) bool {
		if n.DataAtom == atom.Font && scrape.Attr(n, "class") == "smallfont" {
			return true
		}
		return false
	}

	companyName, ok := scrape.Find(root, companyNameMatcher)
	if !ok {
		return nil, fmt.Errorf("No result for CUSIP %s", cusip)
	}

	// Find the security symbol
	symbolMatcher := func(n *html.Node) bool {
		if n.DataAtom == atom.A && n.Parent.Parent.DataAtom == atom.Td {
			return strings.HasPrefix(scrape.Attr(n, "href"), "/webxpress")
		}
		return false
	}

	symbol, ok := scrape.Find(root, symbolMatcher)
	if !ok {
		return nil, fmt.Errorf("Could not extract symbol for CUSIP %s", cusip)
	}

	result := Security{
		Type:        string(t),
		CompanyName: scrape.Text(companyName),
		Symbol:      scrape.Text(symbol),
		CUSIP:       cusip,
	}

	return &result, nil
}
