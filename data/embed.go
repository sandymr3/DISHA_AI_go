// Package data embeds static JSON data files into the binary.
package data

import _ "embed"

//go:embed universities.json
var UniversitiesJSON []byte

//go:embed loan_offers.json
var LoanOffersJSON []byte
