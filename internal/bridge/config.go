package bridge

import "os"

const (
	EnvDebug         = "WSLGIT_DEBUG"
	EnvDistro        = "WSLGIT_DISTRO"
	EnvWSLPath       = "WSLGIT_WSL_PATH"
	EnvGitBinary     = "WSLGIT_GIT_BINARY"
	EnvForceShell    = "WSLGIT_FORCE_SHELL"
	defaultWSLPath   = "wsl.exe"
	defaultGitBinary = "git"
)

type Config struct {
	Debug      bool
	Distro     string
	WSLPath    string
	GitBinary  string
	ForceShell bool
}

func LoadConfigFromEnv() Config {
	config := Config{
		Debug:      isTruthy(os.Getenv(EnvDebug)),
		Distro:     os.Getenv(EnvDistro),
		WSLPath:    defaultIfEmpty(os.Getenv(EnvWSLPath), defaultWSLPath),
		GitBinary:  defaultIfEmpty(os.Getenv(EnvGitBinary), defaultGitBinary),
		ForceShell: isTruthy(os.Getenv(EnvForceShell)),
	}

	return config
}

func defaultIfEmpty(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func isTruthy(value string) bool {
	switch value {
	case "1", "true", "TRUE", "True", "yes", "YES", "on", "ON":
		return true
	default:
		return false
	}
}
