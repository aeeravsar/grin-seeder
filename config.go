package main

import (
	"fmt"

	"github.com/BurntSushi/toml"
)

type Config struct {
	DNS  DNSConfig  `toml:"dns"`
	Node NodeConfig `toml:"node"`
}

type DNSConfig struct {
	Host   string `toml:"host"`
	Port   int    `toml:"port"`
	Origin string `toml:"origin"`
	NS     string `toml:"ns"`
	Email  string `toml:"email"`
}

type NodeConfig struct {
	Mode      string   `toml:"mode"`
	Peers     []string `toml:"peers"`
	AliveOnly bool     `toml:"alive_only"`
	URL       string   `toml:"url"`
	Secret    string   `toml:"secret"`
	Interval  int      `toml:"interval"`
}

func loadConfig(path string) (*Config, error) {
	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, err
	}
	if cfg.Node.Interval == 0 {
		cfg.Node.Interval = 60
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
	if cfg.Node.Mode != "static" && cfg.Node.Mode != "dynamic" {
		return fmt.Errorf("node.mode must be \"static\" or \"dynamic\", got %q", cfg.Node.Mode)
	}
	if cfg.Node.Mode == "dynamic" || cfg.Node.AliveOnly {
		if cfg.Node.URL == "" {
			return fmt.Errorf("node.url is required when mode is dynamic or alive_only is true")
		}
		if cfg.Node.Secret == "" {
			return fmt.Errorf("node.secret is required when mode is dynamic or alive_only is true")
		}
	}
	return nil
}
