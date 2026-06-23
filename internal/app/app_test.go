package app

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestRunHelpAliases(t *testing.T) {
	for _, args := range [][]string{{"-help"}, {"--help"}, {"-h"}, {"help"}} {
		var stdout bytes.Buffer
		code := Run(args, &stdout, &bytes.Buffer{})
		if code != 0 {
			t.Fatalf("Run(%v) exit code = %d", args, code)
		}

		var resp map[string]any
		if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
			t.Fatalf("Run(%v) output is not JSON: %v", args, err)
		}
		if resp["ok"] != true || resp["type"] != "help" {
			t.Fatalf("Run(%v) response = %#v", args, resp)
		}
	}
}

func TestRunVersionAliases(t *testing.T) {
	for _, args := range [][]string{{"-version"}, {"--version"}, {"version"}} {
		var stdout bytes.Buffer
		code := Run(args, &stdout, &bytes.Buffer{})
		if code != 0 {
			t.Fatalf("Run(%v) exit code = %d", args, code)
		}

		var resp map[string]any
		if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
			t.Fatalf("Run(%v) output is not JSON: %v", args, err)
		}
		if resp["ok"] != true || resp["type"] != "version" {
			t.Fatalf("Run(%v) response = %#v", args, resp)
		}
		if resp["version"] == "" {
			t.Fatalf("Run(%v) missing version: %#v", args, resp)
		}
	}
}
