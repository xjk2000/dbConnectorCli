package safety

import (
	"testing"

	"dbconnector/internal/config"
)

func TestCheckWriteAllowed(t *testing.T) {
	cfg := &config.Config{Defaults: config.Defaults{AllowWrite: true}}
	profile := &config.Profile{Name: "local", Readonly: false}

	if errResp := CheckWriteAllowed(cfg, profile, true); errResp != nil {
		t.Fatalf("unexpected error: %v", errResp)
	}
}

func TestCheckWriteRequiresExplicitFlag(t *testing.T) {
	cfg := &config.Config{Defaults: config.Defaults{AllowWrite: true}}
	profile := &config.Profile{Name: "local", Readonly: false}

	errResp := CheckWriteAllowed(cfg, profile, false)
	if errResp == nil {
		t.Fatal("expected error")
	}
	if errResp.Code != "WRITE_NOT_ALLOWED" {
		t.Fatalf("code = %s", errResp.Code)
	}
}

func TestCheckWriteRequiresGlobalPermission(t *testing.T) {
	cfg := &config.Config{Defaults: config.Defaults{AllowWrite: false}}
	profile := &config.Profile{Name: "local", Readonly: false}

	errResp := CheckWriteAllowed(cfg, profile, true)
	if errResp == nil {
		t.Fatal("expected error")
	}
	if errResp.Code != "WRITE_NOT_ALLOWED" {
		t.Fatalf("code = %s", errResp.Code)
	}
}

func TestCheckWriteRespectsReadonlyProfile(t *testing.T) {
	cfg := &config.Config{Defaults: config.Defaults{AllowWrite: true}}
	profile := &config.Profile{Name: "local", Readonly: true}

	errResp := CheckWriteAllowed(cfg, profile, true)
	if errResp == nil {
		t.Fatal("expected error")
	}
	if errResp.Code != "WRITE_NOT_ALLOWED" {
		t.Fatalf("code = %s", errResp.Code)
	}
}
