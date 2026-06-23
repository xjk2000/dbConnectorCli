package app

import (
	"bytes"
	"encoding/json"
	"strings"
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
		if _, ok := resp["commandTree"].(map[string]any); !ok {
			t.Fatalf("Run(%v) missing commandTree: %#v", args, resp)
		}
	}
}

func TestHelpUsageContainsAllCommands(t *testing.T) {
	var stdout bytes.Buffer
	code := Run([]string{"--help"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}

	var resp map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		t.Fatalf("output is not JSON: %v", err)
	}
	usage, ok := resp["usage"].(string)
	if !ok {
		t.Fatalf("missing usage: %#v", resp)
	}

	required := []string{
		"dbc config path",
		"dbc profile list",
		"dbc profile test",
		"dbc mysql databases",
		"dbc mysql tables",
		"dbc mysql table",
		"dbc mysql query",
		"dbc mysql count",
		"dbc mysql explain",
		"dbc mysql exec",
		"dbc redis ping",
		"dbc redis info",
		"dbc redis scan",
		"dbc redis count",
		"dbc redis get",
		"dbc redis hgetall",
		"dbc redis ttl",
		"dbc redis type",
		"dbc redis set",
		"dbc redis del",
		"dbc -version",
	}
	for _, command := range required {
		if !strings.Contains(usage, command) {
			t.Fatalf("usage missing %q:\n%s", command, usage)
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
