package main

import (
	"reflect"
	"testing"
)

func sources(es []Expense) []string {
	out := make([]string, len(es))
	for i, e := range es {
		out[i] = e.Source
	}
	return out
}

func TestDedupeExpenses(t *testing.T) {
	type pair struct{ dropped, kept string }

	tests := []struct {
		name     string
		in       []Expense
		wantKept []string // Source fields of kept expenses, in order
		wantDups []pair
	}{
		{
			name: "ticket and two card-voucher photos of one purchase collapse to the first",
			in: []Expense{
				{Source: "IMG_7858.HEIC", Date: "2026-11-15", Vendor: "Heathrow Express", Currency: "GBP", Amount: 26.00},
				{Source: "IMG_7859.HEIC", Date: "2026-11-15", Vendor: "Heathrow Express Central Rail Station", Currency: "GBP", Amount: 26.00},
				{Source: "IMG_7860.HEIC", Date: "2026-11-15", Vendor: "National Rail", Currency: "GBP", Amount: 26.00},
			},
			wantKept: []string{"IMG_7858.HEIC"},
			wantDups: []pair{
				{"IMG_7859.HEIC", "IMG_7858.HEIC"},
				{"IMG_7860.HEIC", "IMG_7858.HEIC"},
			},
		},
		{
			name: "same amount on different dates is kept",
			in: []Expense{
				{Source: "a", Date: "2026-11-04", Currency: "GBP", Amount: 4.29},
				{Source: "b", Date: "2026-11-05", Currency: "GBP", Amount: 4.29},
			},
			wantKept: []string{"a", "b"},
		},
		{
			name: "same date and amount in different currencies is kept",
			in: []Expense{
				{Source: "a", Date: "2026-11-04", Currency: "GBP", Amount: 10.00},
				{Source: "b", Date: "2026-11-04", Currency: "EUR", Amount: 10.00},
			},
			wantKept: []string{"a", "b"},
		},
		{
			name: "amounts differing by a penny are kept",
			in: []Expense{
				{Source: "a", Date: "2026-11-04", Currency: "GBP", Amount: 26.00},
				{Source: "b", Date: "2026-11-04", Currency: "GBP", Amount: 26.01},
			},
			wantKept: []string{"a", "b"},
		},
		{
			name: "floating-point noise below a penny is treated as equal",
			in: []Expense{
				{Source: "a", Date: "2026-11-04", Currency: "EUR", Amount: 14.40},
				{Source: "b", Date: "2026-11-04", Currency: "EUR", Amount: 14.399999999},
			},
			wantKept: []string{"a"},
			wantDups: []pair{{"b", "a"}},
		},
		{
			name: "currency comparison is case-insensitive",
			in: []Expense{
				{Source: "a", Date: "2026-11-04", Currency: "GBP", Amount: 7.00},
				{Source: "b", Date: "2026-11-04", Currency: "gbp", Amount: 7.00},
			},
			wantKept: []string{"a"},
			wantDups: []pair{{"b", "a"}},
		},
		{
			name: "non-adjacent duplicate is removed and order of the rest preserved",
			in: []Expense{
				{Source: "a", Date: "2026-11-15", Currency: "GBP", Amount: 26.00},
				{Source: "b", Date: "2026-11-15", Currency: "GBP", Amount: 48.50},
				{Source: "c", Date: "2026-11-15", Currency: "GBP", Amount: 26.00},
				{Source: "d", Date: "2026-11-04", Currency: "GBP", Amount: 7.00},
			},
			wantKept: []string{"a", "b", "d"},
			wantDups: []pair{{"c", "a"}},
		},
		{
			name:     "empty input",
			in:       []Expense{},
			wantKept: []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			kept, dups := dedupeExpenses(tc.in)

			if got := sources(kept); !reflect.DeepEqual(got, tc.wantKept) {
				t.Errorf("kept: got %v, want %v", got, tc.wantKept)
			}

			gotDups := make([]pair, len(dups))
			for i, d := range dups {
				gotDups[i] = pair{d.Dropped.Source, d.Kept.Source}
			}
			if len(gotDups) != len(tc.wantDups) || (len(gotDups) > 0 && !reflect.DeepEqual(gotDups, tc.wantDups)) {
				t.Errorf("duplicates: got %v, want %v", gotDups, tc.wantDups)
			}
		})
	}
}
