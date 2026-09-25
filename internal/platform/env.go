package platform

import (
	"errors"
	"io/fs"
	"os"
	"strings"
)

// LoadDotEnv reads a KEY=VALUE file into the process environment.
//
// Behaviour:
//   - existing environment variables take precedence (never overwritten)
//   - lines starting with '#' and empty lines are skipped
//   - values wrapped in single or double quotes have the quotes stripped
//   - a missing file is NOT an error (running without .env is fine)
//
// This intentionally covers the simple .env format this project uses.
// It exists so the app behaves identically whether launched with `make run`,
// `go run`, or from an IDE like GoLand (which does not read .env files).
func LoadDotEnv(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil // no .env file: keep going with defaults / shell env
		}
		return err
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		eq := strings.Index(line, "=")
		if eq < 0 {
			continue // not KEY=VALUE; ignore
		}

		name := strings.TrimSpace(line[:eq])
		value := strings.TrimSpace(line[eq+1:])

		// Strip optional surrounding quotes: FOO="bar" or FOO='bar'.
		if len(value) >= 2 &&
			((value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'')) {
			value = value[1 : len(value)-1]
		}

		if name == "" {
			continue
		}
		if _, ok := os.LookupEnv(name); !ok {
			if err := os.Setenv(name, value); err != nil {
				return err
			}
		}
	}

	return nil
}
