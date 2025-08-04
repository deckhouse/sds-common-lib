package slogh

import "sync"

// This is reloadable config, shared across all [Handler] structs.
// Reloading can be started with [EnableConfigReload].
var defaultConfig = &Config{}

var defaultConfigMu = &sync.RWMutex{}

func DefaultConfig() *Config {
	// make a copy atomically
	defaultConfigMu.RLock()
	defer defaultConfigMu.RUnlock()
	res := *defaultConfig
	return &res
}

func DefaultConfigVersion() uint {
	return defaultConfig.version
}

func UpdateDefaultConfig(data map[string]string) error {
	defaultConfigMu.Lock()
	defer defaultConfigMu.Unlock()
	return defaultConfig.UpdateConfigData(data)
}
