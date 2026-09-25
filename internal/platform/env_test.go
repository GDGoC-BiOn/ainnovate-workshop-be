package platform

import (
	"os"
	"path/filepath"
	"testing"
)

// cleanEnv removes the given keys so each test starts from a clean slate,
// then restores whatever the machine/IDE had them set to.
func cleanEnv(t *testing.T, keys ...string) {
	t.Helper()
	for _, key := range keys {
		old, had := os.LookupEnv(key)
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unset %s: %v", key, err)
		}
		t.Cleanup(func() {
			if had {
				_ = os.Setenv(key, old)
			} else {
				_ = os.Unsetenv(key)
			}
		})
	}
}

func writeTestEnv(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}
	return path
}

func TestLoadDotEnv_SetsValues(t *testing.T) {
	cleanEnv(t, "HTTP_ADDR", "DATABASE_URL")
	path := writeTestEnv(t, "HTTP_ADDR=:8087\nDATABASE_URL=postgres://x/y?sslmode=disable\n")

	if err := LoadDotEnv(path); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := os.Getenv("HTTP_ADDR"); got != ":8087" {
		t.Errorf("HTTP_ADDR = %q, want :8087", got)
	}
	if got := os.Getenv("DATABASE_URL"); got != "postgres://x/y?sslmode=disable" {
		t.Errorf("DATABASE_URL = %q, want postgres://x/y?sslmode=disable", got)
	}
}

func TestLoadDotEnv_StripsQuotes(t *testing.T) {
	cleanEnv(t, "HTTP_ADDR", "AI_STUB_FAIL_RATE")
	path := writeTestEnv(t, "HTTP_ADDR=\":8089\"\nAI_STUB_FAIL_RATE='0.5'\n")

	if err := LoadDotEnv(path); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := os.Getenv("HTTP_ADDR"); got != ":8089" {
		t.Errorf("HTTP_ADDR = %q, want :8089 (double-quoted)", got)
	}
	if got := os.Getenv("AI_STUB_FAIL_RATE"); got != "0.5" {
		t.Errorf("AI_STUB_FAIL_RATE = %q, want 0.5 (single-quoted)", got)
	}
}

func TestLoadDotEnv_DoesNotOverrideExistingEnv(t *testing.T) {
	t.Setenv("HTTP_ADDR", ":1234") // already set by the shell / IDE

	path := writeTestEnv(t, "HTTP_ADDR=:8087\n")
	if err := LoadDotEnv(path); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := os.Getenv("HTTP_ADDR"); got != ":1234" {
		t.Errorf("existing env was overwritten: HTTP_ADDR = %q, want :1234", got)
	}
}

func TestLoadDotEnv_IgnoresCommentsAndSkipsInvalidLines(t *testing.T) {
	cleanEnv(t, "HTTP_ADDR")
	path := writeTestEnv(t, `
# a comment
HTTP_ADDR=:8087

NOT_A_KEY_VALUE_LINE
ALSO_BAD:8080
`)

	if err := LoadDotEnv(path); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := os.Getenv("HTTP_ADDR"); got != ":8087" {
		t.Errorf("HTTP_ADDR = %q, want :8087", got)
	}
}

func TestLoadDotEnv_MissingFileIsNotAnError(t *testing.T) {
	if err := LoadDotEnv(filepath.Join(t.TempDir(), "does-not-exist.env")); err != nil {
		t.Fatalf("missing .env should not error, got: %v", err)
	}
}
