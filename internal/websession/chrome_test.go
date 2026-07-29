package websession

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestDeriveChromeKey(t *testing.T) {
	key, err := deriveChromeKey("test-password")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := hex.EncodeToString(key), "c0ffe4c25f07f62bfc6ab011d9efa54e"; got != want {
		t.Fatalf("deriveChromeKey() = %s, want %s", got, want)
	}
}

func TestDecryptChromeValue_V24(t *testing.T) {
	key, err := deriveChromeKey("test-password")
	if err != nil {
		t.Fatal(err)
	}
	encrypted, err := hex.DecodeString("7631301712cfebbd2b19888cec2be4dc805c434274e1725c967526742857b46344e4da85fe0d706ab8c64b866711deeec9b594")
	if err != nil {
		t.Fatal(err)
	}

	got, err := decryptChromeValue(key, ".google.com", encrypted, 24)
	if err != nil {
		t.Fatal(err)
	}
	if got != "sapisid-secret" {
		t.Fatalf("decryptChromeValue() = %q, want %q", got, "sapisid-secret")
	}

	if _, err := decryptChromeValue(key, ".example.com", encrypted, 24); err == nil || !strings.Contains(err.Error(), "domain hash") {
		t.Fatalf("wrong-domain error = %v, want domain hash error", err)
	}
	if _, err := decryptChromeValue(key, ".google.com", encrypted, 25); err == nil || !strings.Contains(err.Error(), "schema") {
		t.Fatalf("newer-schema error = %v, want schema error", err)
	}
}

func TestChromeTime(t *testing.T) {
	if got := chromeTime(0); !got.IsZero() {
		t.Fatalf("chromeTime(0) = %s, want zero", got)
	}
	want := time.Unix(1700000000, 0).UTC()
	if got := chromeTime(13344473600000000); !got.Equal(want) {
		t.Fatalf("chromeTime() = %s, want %s", got, want)
	}
}

func TestDiscoverChromeCookieDBs(t *testing.T) {
	root := t.TempDir()
	want := []string{
		filepath.Join(root, "Default", "Network", "Cookies"),
		filepath.Join(root, "Profile 2", "Network", "Cookies"),
	}
	paths := append(slices.Clone(want), filepath.Join(root, "Default", "Cookies"))
	for _, path := range paths {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(root, "System Profile"), 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := discoverChromeCookieDBs(root)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, want) {
		t.Fatalf("discoverChromeCookieDBs() = %q, want %q", got, want)
	}
}

func TestParseChromeRows(t *testing.T) {
	rows, err := parseChromeRows([]byte(`[{
		"host_key": ".google.com",
		"name": "SAPISID",
		"value": "",
		"encrypted_value": "763130",
		"expires_utc": 13344473600000000,
		"db_version": 24
	}]`))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].HostKey != ".google.com" || rows[0].Name != "SAPISID" ||
		rows[0].EncryptedValue != "763130" || rows[0].ExpiresUTC != 13344473600000000 ||
		rows[0].DBVersion != 24 {
		t.Fatalf("parseChromeRows() = %#v", rows)
	}
}

func TestChromeCookies_Plaintext(t *testing.T) {
	cookies, err := chromeCookies([]chromeCookieRow{{
		HostKey: ".google.com",
		Name:    "SAPISID",
		Value:   "plaintext",
	}}, make([]byte, 16), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(cookies) != 1 || cookies[0].Value != "plaintext" {
		t.Fatalf("chromeCookies() = %#v", cookies)
	}
}

func TestChromeCookies_RejectsPlaintextAndEncrypted(t *testing.T) {
	_, err := chromeCookies([]chromeCookieRow{{
		HostKey:        ".google.com",
		Name:           "SAPISID",
		Value:          "plaintext",
		EncryptedValue: "763130",
		DBVersion:      24,
	}}, make([]byte, 16), time.Now())
	if err == nil || !strings.Contains(err.Error(), "both plaintext and encrypted") {
		t.Fatalf("chromeCookies() error = %v", err)
	}
}
