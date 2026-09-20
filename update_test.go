package main

import (
	"strings"
	"testing"
)

func TestStripChecksumsSection(t *testing.T) {
	body := "New stuff here.\n\n## Контрольные суммы (SHA-256)\n\n```\nabc  f.tar.gz\n```\n\n## What's Changed\n- x"
	got := stripChecksumsSection(body)
	if strings.Contains(got, "abc") || strings.Contains(got, "Контрольные суммы") {
		t.Fatalf("checksums block not stripped:\n%s", got)
	}
	if !strings.Contains(got, "New stuff here.") || !strings.Contains(got, "What's Changed") {
		t.Fatalf("human text damaged:\n%s", got)
	}

	en := "Notes.\n\n## SHA-256 Checksums\n\n```\nabc  f.tar.gz\n```\n"
	if got := stripChecksumsSection(en); strings.Contains(got, "abc") {
		t.Fatalf("EN block not stripped:\n%s", got)
	}

	plain := "## Migration to SHA-256\nJust a header, no fence, keep it.\n"
	if got := stripChecksumsSection(plain); got != strings.TrimRight(plain, "\n") {
		t.Fatalf("non-fenced header damaged:\n%s", got)
	}

	untouched := "Hello.\n\nNo hashes here.\n"
	if got := stripChecksumsSection(untouched); got != strings.TrimRight(untouched, "\n") {
		t.Fatalf("plain body changed:\n%s", got)
	}
}
