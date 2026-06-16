package main

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/BurntSushi/toml"
)

type Config struct {
	DNS  DNSConfig  `toml:"dns"`
	Node NodeConfig `toml:"node"`
}

type DNSConfig struct {
	Host       string `toml:"host"`
	Port       int    `toml:"port"`
	Origin     string `toml:"origin"`
	NS         string `toml:"ns"`
	Email      string `toml:"email"`
	MaxRecords int    `toml:"max_records"`
}

type NodeConfig struct {
	Mode         string   `toml:"mode"`
	Peers        []string `toml:"peers"`
	AliveOnly    bool     `toml:"alive_only"`
	URL          string   `toml:"url"`
	Secret       string   `toml:"secret"`
	Interval     int      `toml:"interval"`
	P2PPort      int      `toml:"p2p_port"`
	CheckTimeout int      `toml:"check_timeout"`
	MinUserAgent string   `toml:"min_user_agent"`
}

func loadConfig(path string) (*Config, error) {
	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, err
	}
	if cfg.Node.Interval == 0 {
		cfg.Node.Interval = 60
	}
	if cfg.Node.CheckTimeout == 0 {
		cfg.Node.CheckTimeout = 3
	}
	if cfg.Node.P2PPort == 0 {
		cfg.Node.P2PPort = inferP2PPort(cfg.Node.URL)
	}
	if cfg.DNS.MaxRecords == 0 {
		cfg.DNS.MaxRecords = 24
	}
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	return &cfg, nil
}

func (cfg *Config) validate() error {
	if cfg.DNS.Host == "" {
		return fmt.Errorf("dns.host is required")
	}
	if cfg.DNS.Port == 0 {
		return fmt.Errorf("dns.port is required")
	}
	if cfg.DNS.Origin == "" {
		return fmt.Errorf("dns.origin is required")
	}
	if cfg.DNS.NS == "" {
		return fmt.Errorf("dns.ns is required")
	}
	if cfg.DNS.Email == "" {
		return fmt.Errorf("dns.email is required")
	}
	if cfg.DNS.MaxRecords < 0 {
		return fmt.Errorf("dns.max_records must not be negative")
	}
	if cfg.Node.Mode != "static" && cfg.Node.Mode != "dynamic" {
		return fmt.Errorf("node.mode must be \"static\" or \"dynamic\", got %q", cfg.Node.Mode)
	}
	if cfg.Node.P2PPort < 0 {
		return fmt.Errorf("node.p2p_port must not be negative")
	}
	if cfg.Node.CheckTimeout < 0 {
		return fmt.Errorf("node.check_timeout must not be negative")
	}
	if cfg.Node.MinUserAgent != "" {
		if _, _, _, ok := parseMWGrinUserAgent(cfg.Node.MinUserAgent); !ok {
			return fmt.Errorf("node.min_user_agent must look like \"MW/Grin 5.4.0\"")
		}
	}
	if cfg.Node.Mode == "dynamic" {
		if cfg.Node.URL == "" {
			return fmt.Errorf("node.url is required when mode is dynamic")
		}
		if cfg.Node.Secret == "" {
			return fmt.Errorf("node.secret is required when mode is dynamic")
		}
	}
	return nil
}

func inferP2PPort(rawURL string) int {
	u, err := url.Parse(rawURL)
	if err != nil {
		return 3414
	}
	port := u.Port()
	if port == "" {
		return 3414
	}
	apiPort, err := strconv.Atoi(port)
	if err != nil {
		return 3414
	}
	return apiPort + 1
}
