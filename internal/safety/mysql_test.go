package safety

import "testing"

func TestCheckMySQLReadAllowsSelectWithLeadingComment(t *testing.T) {
	if errResp := CheckMySQLRead("/* check */\nselect * from users"); errResp != nil {
		t.Fatalf("unexpected error: %v", errResp)
	}
}

func TestCheckMySQLReadRejectsWriteKeyword(t *testing.T) {
	errResp := CheckMySQLRead("update users set name = 'a'")
	if errResp == nil {
		t.Fatal("expected error")
	}
	if errResp.Code != "SQL_NOT_ALLOWED" {
		t.Fatalf("code = %s", errResp.Code)
	}
}

func TestCheckMySQLRejectsMultiStatement(t *testing.T) {
	errResp := CheckMySQLRead("select 1; drop table users")
	if errResp == nil {
		t.Fatal("expected error")
	}
	if errResp.Code != "MULTI_STATEMENT_NOT_ALLOWED" {
		t.Fatalf("code = %s", errResp.Code)
	}
}

func TestCheckMySQLAllowsSemicolonInString(t *testing.T) {
	if errResp := CheckMySQLRead("select ';' as value;"); errResp != nil {
		t.Fatalf("unexpected error: %v", errResp)
	}
}
