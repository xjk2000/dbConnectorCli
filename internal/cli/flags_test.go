package cli

import "testing"

func TestParseFlags(t *testing.T) {
	flags, remaining, errResp := ParseFlags([]string{
		"--profile", "local",
		"--timeout-ms=3000",
		"--limit", "50",
		"--write",
		"extra",
	})
	if errResp != nil {
		t.Fatalf("unexpected error: %v", errResp)
	}
	if flags.Profile != "local" {
		t.Fatalf("profile = %q", flags.Profile)
	}
	if flags.TimeoutMs != 3000 {
		t.Fatalf("timeout = %d", flags.TimeoutMs)
	}
	if flags.Limit != 50 {
		t.Fatalf("limit = %d", flags.Limit)
	}
	if !flags.Write {
		t.Fatal("write flag was not set")
	}
	if len(remaining) != 1 || remaining[0] != "extra" {
		t.Fatalf("remaining = %#v", remaining)
	}
}

func TestParseFlagsRejectsInvalidLimit(t *testing.T) {
	_, _, errResp := ParseFlags([]string{"--limit", "0"})
	if errResp == nil {
		t.Fatal("expected error")
	}
	if errResp.Code != "USAGE_ERROR" {
		t.Fatalf("code = %s", errResp.Code)
	}
}
