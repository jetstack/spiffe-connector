package config

import (
	"fmt"
	"io/fs"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/jetstack/spiffe-connector/types"
)

func LoadConfigFromFs(fsys fs.FS, path string) (*types.Config, error) {
	rawConfig, err := fs.ReadFile(fsys, path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %s", err)
	}

	var cfg types.Config

	err = yaml.Unmarshal(rawConfig, &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %s", err)
	}

	errs := cfg.Validate()
	if len(errs) > 0 {
		var messages []string
		for _, e := range errs {
			messages = append(messages, e.Error())
		}
		return nil, fmt.Errorf("config validation failed: %s", strings.Join(messages, ", "))
	}

	return &cfg, nil
}
