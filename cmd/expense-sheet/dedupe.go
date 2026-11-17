package main

// duplicate records an expense dropped because it matches one already kept.
type duplicate struct {
	Dropped Expense
	Kept    Expense
}

// dedupeExpenses removes expenses that describe the same purchase as an earlier
// one: same date, same currency (case-insensitive) and same amount to the
// nearest minor unit. Vendor and summary are ignored, as the model may name the
// same purchase differently from different documents (e.g. a ticket and its
// card voucher). The first occurrence is kept and input order is preserved.
func dedupeExpenses(in []Expense) ([]Expense, []duplicate) {
	// Stub: implementation pending review of tests.
	return in, nil
}
