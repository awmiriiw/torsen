package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	TorUser             string
	TorControlPort      int
	TorControlPassword  string
	TorBootstrapTimeout int
	SystemTorrcPath     string

	TorDNSPort     int
	TorDNSIP       string
	TorTransPort   int
	NonTorNetworks []string

	ResolvConfPath string
	DNSServiceName string

	StateDir string
	LogDir   string
	LogLevel string
}

func Load() (*Config, error) {
	envPaths := []string{
		".env",
		"/etc/torsen/.env",
	}

	for _, p := range envPaths {
		if _, err := os.Stat(p); err == nil {
			_ = godotenv.Load(p)
			break
		}
	}

	cfg := &Config{}
	var err error

	cfg.TorUser = getEnv("TOR_USER", "tor")

	cfg.TorControlPort, err = getEnvInt("TOR_CONTROL_PORT", 9051)
	if err != nil {
		return nil, fmt.Errorf("invalid TOR_CONTROL_PORT: %w", err)
	}

	cfg.TorControlPassword = getEnv("TOR_CONTROL_PASSWORD", "")

	cfg.TorBootstrapTimeout, err = getEnvInt("TOR_BOOTSTRAP_TIMEOUT", 120)
	if err != nil {
		return nil, fmt.Errorf("invalid TOR_BOOTSTRAP_TIMEOUT: %w", err)
	}

	cfg.SystemTorrcPath = getEnv("SYSTEM_TORRC_PATH", "/etc/tor/torrc")

	cfg.TorDNSPort, err = getEnvInt("TOR_DNS_PORT", 5353)
	if err != nil {
		return nil, fmt.Errorf("invalid TOR_DNS_PORT: %w", err)
	}

	cfg.TorDNSIP = getEnv("TOR_DNS_IP", "127.0.0.1")

	cfg.TorTransPort, err = getEnvInt("TOR_TRANSPARENT_PORT", 9040)
	if err != nil {
		return nil, fmt.Errorf("invalid TOR_TRANSPARENT_PORT: %w", err)
	}

	networks := getEnv("NON_TOR_NETWORKS", "192.168.0.0/16 10.0.0.0/8 172.16.0.0/12")
	cfg.NonTorNetworks = strings.Fields(networks)

	cfg.ResolvConfPath = getEnv("RESOLV_CONF_PATH", "/etc/resolv.conf")
	cfg.DNSServiceName = getEnv("DNS_SERVICE_NAME", "")

	cfg.StateDir = getEnv("STATE_DIR", "/var/lib/torsen")
	cfg.LogDir = getEnv("LOG_DIR", "/var/log/torsen")
	cfg.LogLevel = getEnv("LOG_LEVEL", "info")

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return fallback
}

func getEnvInt(key string, fallback int) (int, error) {
	val, ok := os.LookupEnv(key)
	if !ok || val == "" {
		return fallback, nil
	}

	parsed, err := strconv.Atoi(val)
	if err != nil {
		return 0, fmt.Errorf("cannot parse '%s' as int for key %s", val, key)
	}

	return parsed, nil
}
