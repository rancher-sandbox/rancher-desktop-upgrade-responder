package rancherdesktop

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const VersionTagLatest = "latest"

// ResponseConfig is the Upgrade Responder configuration.
type ResponseConfig struct {
	Rules    []Rule
	Versions []Version
}

func (responseConfig *ResponseConfig) Validate() error {
	// validate Rules
	for _, rule := range responseConfig.Rules {
		if err := rule.Validate(); err != nil {
			return fmt.Errorf("invalid rule %v: %w", rule, err)
		}
	}

	// validate Versions
	versionMap := map[string]Version{}
	tagVersionsMap := map[string][]Version{}
	for _, version := range responseConfig.Versions {
		if err := version.Validate(); err != nil {
			return fmt.Errorf("invalid version %q: %w", version.Name, err)
		}
		if _, ok := versionMap[version.Name]; ok {
			return fmt.Errorf("duplicate version name %q", version.Name)
		}
		for _, tag := range version.Tags {
			tagVersionsMap[tag] = append(tagVersionsMap[tag], version)
		}
		versionMap[version.Name] = version
	}
	if len(tagVersionsMap[VersionTagLatest]) == 0 {
		return errors.New("no latest label specified")
	}

	return nil
}

// ReadConfig reads a JSON file, processes the data therein into a ResponseConfig,
// and validates that ResponseConfig.
func ReadConfig(configPath string) (ResponseConfig, error) {
	path := filepath.Clean(configPath)
	f, err := os.Open(path)
	if err != nil {
		return ResponseConfig{}, fmt.Errorf("failed to open config file: %w", err)
	}
	defer f.Close()
	var config ResponseConfig
	if err := json.NewDecoder(f).Decode(&config); err != nil {
		return ResponseConfig{}, fmt.Errorf("failed to parse config as JSON: %w", err)
	}

	// Set every Supported key to true by default
	for i := range config.Versions {
		config.Versions[i].Supported = true
	}

	if err := config.Validate(); err != nil {
		return ResponseConfig{}, fmt.Errorf("invalid config: %w", err)
	}

	return config, nil
}
