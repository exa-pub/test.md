package parser

import (
	"github.com/testmd/testmd/internal/models"
	"gopkg.in/yaml.v3"
)

// ParseConfig parses .testmd.yaml content into Config.
// Empty or nil input returns defaults.
func ParseConfig(data []byte) (models.Config, error) {
	cfg := models.Config{
		Ignorefile: ".gitignore",
	}
	if len(data) == 0 {
		return cfg, nil
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	if cfg.Ignorefile == "" {
		cfg.Ignorefile = ".gitignore"
	}
	return cfg, nil
}
