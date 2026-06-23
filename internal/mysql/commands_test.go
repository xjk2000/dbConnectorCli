package mysql

import "testing"

func TestQuoteIdentifier(t *testing.T) {
	got := quoteIdentifier("db`name")
	want := "`db``name`"
	if got != want {
		t.Fatalf("quoteIdentifier() = %q, want %q", got, want)
	}
}

func TestTrimTrailingSemicolon(t *testing.T) {
	got := trimTrailingSemicolon(" select 1;; ")
	want := "select 1"
	if got != want {
		t.Fatalf("trimTrailingSemicolon() = %q, want %q", got, want)
	}
}
