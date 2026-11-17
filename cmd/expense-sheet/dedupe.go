package main

import (
	"math"
	"strings"
)

// duplicate records an expense dropped because it matches one already kept.
type duplicate struct {
	Dropped Expense
	Kept    Expense
}

type purchaseKey struct {
	date     string
	currency string
	minor    int64 // amount in minor units (pence, cents)
}

// dedupeExpenses removes expenses that describe the same purchase as an earlier
// one: same date, same currency (case-insensitive) and same amount to the
// nearest minor unit. Vendor and summary are ignored, as the model may name the
// same purchase differently from different documents (e.g. a ticket and its
// card voucher). The first occurrence is kept and input order is preserved.
func dedupeExpenses(in []Expense) ([]Expense, []duplicate) {
	kept := make([]Expense, 0, len(in))
	var dups []duplicate
	seen := make(map[purchaseKey]int, len(in)) // key -> index into kept

	for _, e := range in {
		k := purchaseKey{
			date:     e.Date,
			currency: strings.ToUpper(e.Currency),
			minor:    int64(math.Round(e.Amount * 100)),
		}
		if i, ok := seen[k]; ok {
			dups = append(dups, duplicate{Dropped: e, Kept: kept[i]})
			continue
		}
		seen[k] = len(kept)
		kept = append(kept, e)
	}
	return kept, dups
}
