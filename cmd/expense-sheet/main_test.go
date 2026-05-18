package main

import "testing"

func TestSortExpensesByDate(t *testing.T) {
	tests := []struct {
		name string
		in   []Expense
		want []string // expected order of Vendor fields
	}{
		{
			name: "already in order is unchanged",
			in: []Expense{
				{Date: "2026-05-11", Vendor: "a"},
				{Date: "2026-05-12", Vendor: "b"},
				{Date: "2026-05-13", Vendor: "c"},
			},
			want: []string{"a", "b", "c"},
		},
		{
			name: "reversed input is sorted ascending",
			in: []Expense{
				{Date: "2026-05-13", Vendor: "c"},
				{Date: "2026-05-12", Vendor: "b"},
				{Date: "2026-05-11", Vendor: "a"},
			},
			want: []string{"a", "b", "c"},
		},
		{
			name: "same-date entries keep their original order",
			in: []Expense{
				{Date: "2026-05-11", Vendor: "a-first"},
				{Date: "2026-05-11", Vendor: "a-second"},
				{Date: "2026-05-10", Vendor: "earlier"},
			},
			want: []string{"earlier", "a-first", "a-second"},
		},
		{
			name: "unparseable dates sort to the end, in original order",
			in: []Expense{
				{Date: "unknown", Vendor: "bad-1"},
				{Date: "2026-05-12", Vendor: "b"},
				{Date: "", Vendor: "bad-2"},
				{Date: "2026-05-11", Vendor: "a"},
			},
			want: []string{"a", "b", "bad-1", "bad-2"},
		},
		{
			name: "cross-year ordering uses date semantics, not string compare",
			in: []Expense{
				{Date: "2025-12-31", Vendor: "ny-eve-2025"},
				{Date: "2026-01-01", Vendor: "ny-day-2026"},
				{Date: "2025-01-01", Vendor: "ny-day-2025"},
			},
			want: []string{"ny-day-2025", "ny-eve-2025", "ny-day-2026"},
		},
		{
			name: "empty slice is a no-op",
			in:   []Expense{},
			want: []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sortExpensesByDate(tc.in)
			if len(tc.in) != len(tc.want) {
				t.Fatalf("length: got %d, want %d", len(tc.in), len(tc.want))
			}
			for i, w := range tc.want {
				if tc.in[i].Vendor != w {
					got := make([]string, len(tc.in))
					for j, e := range tc.in {
						got[j] = e.Vendor
					}
					t.Fatalf("order: got %v, want %v", got, tc.want)
				}
			}
		})
	}
}
