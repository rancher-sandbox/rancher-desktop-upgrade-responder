package rancherdesktop

import (
	"fmt"
	"github.com/Masterminds/semver/v3"
	"time"
)

// Version represents a version of the application. Present in both the config
// passed to Upgrade Responder, and in the responses sent to clients.
type Version struct {
	// Must be a valid semver.
	Name        string
	ReleaseDate string
	// Indicates whether this specific Version is supported for
	// a given client that talks to Upgrade Responder.
	// Determined from information that the client sends in its request.
	Supported bool
	Tags      []string
	ExtraInfo map[string]string
}

// Validate is used to check whether a Version is valid.
func (version *Version) Validate() error {
	if _, err := semver.NewVersion(version.Name); err != nil {
		return fmt.Errorf("failed to parse Name: %w", err)
	}
	if _, err := time.Parse(time.RFC3339, version.ReleaseDate); err != nil {
		return fmt.Errorf("failed to parse ReleaseDate: %w", err)
	}
	return nil
}
