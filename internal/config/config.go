package config

import (
	"errors"
	"os"

	"github.com/compatgate/compatgate/internal/findings"
	"gopkg.in/yaml.v3"
)

type Cloud struct {
	BaseURL      string `yaml:"base_url" json:"base_url"`
	ProjectToken string `yaml:"project_token" json:"project_token"`
}

type Config struct {
	SeverityThreshold string   `yaml:"severity_threshold" json:"severity_threshold"`
	IgnoreRules       []string `yaml:"ignore_rules" json:"ignore_rules"`
	IncludePaths      []string `yaml:"include_paths" json:"include_paths"`
	ExcludePaths      []string `yaml:"exclude_paths" json:"exclude_paths"`
	Cloud             Cloud    `yaml:"cloud" json:"cloud"`
}

func Default() Config {
	return Config{
		SeverityThreshold: string(findings.SeverityWarn),
		IgnoreRules:       []string{},
		IncludePaths:      []string{},
		ExcludePaths:      []string{},
	}
}

func Load(path string) (Config, error) {
	if path == "" {
		return Default(), nil
	}
	bytes, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Default(), nil
		}
		return Config{}, err
	}
	cfg := Default()
	if err := yaml.Unmarshal(bytes, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Threshold() (findings.Severity, error) {
	return findings.ParseSeverity(c.SeverityThreshold)
}

func (c Config) ShouldIgnore(ruleID string) bool {
	for _, item := range c.IgnoreRules {
		if item == ruleID {
			return true
		}
	}
	return false
}
