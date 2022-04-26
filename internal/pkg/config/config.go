package config

import (
	"fmt"
	"io/fs"
	"strings"
	"sync/atomic"

	"gopkg.in/yaml.v3"

	"github.com/jetstack/spiffe-connector/types"
)

var (
	currentConfig atomic.Value // *types.ConfigFile
	currentSource atomic.Value // *SpiffeConnectorSource

	CurrentSource DynamicSource
)

func init() {
	currentConfig.Store(new(types.ConfigFile))
	currentSource.Store(new(SpiffeConnectorSource))
}

func ReadConfigFromFS(fsys fs.FS, path string) (*types.ConfigFile, error) {
	rawConfig, err := fs.ReadFile(fsys, path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %s", err)
	}

	var cfg types.ConfigFile

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

func ReadAndStoreConfig(fsys fs.FS, path string) error {
	config, err := ReadConfigFromFS(fsys, path)
	if err != nil {
		return err
	}
	StoreConfig(config)
	return nil
}

func StoreConfig(cfg *types.ConfigFile) {
	currentConfig.Store(cfg)
}

func GetCurrentConfig() *types.ConfigFile {
	return currentConfig.Load().(*types.ConfigFile)
}

func StoreSource(source *SpiffeConnectorSource) {
	currentSource.Store(source)
}

func GetCurrentSource() *SpiffeConnectorSource {
	return currentSource.Load().(*SpiffeConnectorSource)
}
