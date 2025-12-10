package config

import (
	"flag"
	"os"
	"testing"
)

func TestNewConfig_DefaultValues(t *testing.T) {
	originalArgs := os.Args
	originalEnv := os.Environ()
	defer func() {
		os.Args = originalArgs
		os.Clearenv()
		for _, env := range originalEnv {
			if key, val, ok := cut(env, "="); ok {
				os.Setenv(key, val)
			}
		}
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	}()

	os.Clearenv()
	os.Args = []string{"test"}

	config := NewConfig()

	if config.AppPort != "localhost:8080" {
		t.Errorf("Expected AppPort to be 'localhost:8080', got '%s'", config.AppPort)
	}

	if config.DatabaseDSN != "" {
		t.Errorf("Expected DatabaseDSN to be empty, got '%s'", config.DatabaseDSN)
	}
}

func TestNewConfig_FlagValues(t *testing.T) {
	originalArgs := os.Args
	originalEnv := os.Environ()
	defer func() {
		os.Args = originalArgs
		os.Clearenv()
		for _, env := range originalEnv {
			if key, val, ok := cut(env, "="); ok {
				os.Setenv(key, val)
			}
		}
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	}()

	os.Clearenv()
	os.Args = []string{"test", "-a", "localhost:9090", "-d", "postgres://user:pass@localhost/db"}

	config := NewConfig()

	if config.AppPort != "localhost:9090" {
		t.Errorf("Expected AppPort to be 'localhost:9090', got '%s'", config.AppPort)
	}

	if config.DatabaseDSN != "postgres://user:pass@localhost/db" {
		t.Errorf("Expected DatabaseDSN to be 'postgres://user:pass@localhost/db', got '%s'", config.DatabaseDSN)
	}
}

func TestNewConfig_EnvVariables(t *testing.T) {
	originalArgs := os.Args
	originalEnv := os.Environ()
	defer func() {
		os.Args = originalArgs
		os.Clearenv()
		for _, env := range originalEnv {
			if key, val, ok := cut(env, "="); ok {
				os.Setenv(key, val)
			}
		}
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	}()

	os.Clearenv()
	os.Setenv("RUN_ADDRESS", "0.0.0.0:8080")
	os.Setenv("DATABASE_DSN", "postgres://test:test@localhost/testdb")

	// Устанавливаем минимальные аргументы
	os.Args = []string{"test"}

	config := NewConfig()

	if config.AppPort != "0.0.0.0:8080" {
		t.Errorf("Expected AppPort to be '0.0.0.0:8080', got '%s'", config.AppPort)
	}

	if config.DatabaseDSN != "postgres://test:test@localhost/testdb" {
		t.Errorf("Expected DatabaseDSN to be 'postgres://test:test@localhost/testdb', got '%s'", config.DatabaseDSN)
	}
}

func TestNewConfig_EnvOverridesFlags(t *testing.T) {
	originalArgs := os.Args
	originalEnv := os.Environ()
	defer func() {
		os.Args = originalArgs
		os.Clearenv()
		for _, env := range originalEnv {
			if key, val, ok := cut(env, "="); ok {
				os.Setenv(key, val)
			}
		}
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	}()

	os.Clearenv()
	os.Setenv("RUN_ADDRESS", "0.0.0.0:3000")
	os.Setenv("DATABASE_DSN", "env_dsn")

	os.Args = []string{"test", "-a", "localhost:9090", "-d", "flag_dsn"}

	config := NewConfig()

	if config.AppPort != "0.0.0.0:3000" {
		t.Errorf("Expected AppPort to be '0.0.0.0:3000' from env, got '%s'", config.AppPort)
	}

	if config.DatabaseDSN != "env_dsn" {
		t.Errorf("Expected DatabaseDSN to be 'env_dsn' from env, got '%s'", config.DatabaseDSN)
	}
}

func TestNewConfig_PartialEnv(t *testing.T) {
	originalArgs := os.Args
	originalEnv := os.Environ()
	defer func() {
		os.Args = originalArgs
		os.Clearenv()
		for _, env := range originalEnv {
			if key, val, ok := cut(env, "="); ok {
				os.Setenv(key, val)
			}
		}
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	}()

	os.Clearenv()
	os.Setenv("RUN_ADDRESS", "127.0.0.1:8080")

	os.Args = []string{"test", "-d", "flag_dsn"}

	config := NewConfig()

	if config.AppPort != "127.0.0.1:8080" {
		t.Errorf("Expected AppPort to be '127.0.0.1:8080', got '%s'", config.AppPort)
	}

	if config.DatabaseDSN != "flag_dsn" {
		t.Errorf("Expected DatabaseDSN to be 'flag_dsn', got '%s'", config.DatabaseDSN)
	}
}

func cut(s, sep string) (string, string, bool) {
	if i := index(s, sep); i >= 0 {
		return s[:i], s[i+len(sep):], true
	}
	return "", "", false
}

func index(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
