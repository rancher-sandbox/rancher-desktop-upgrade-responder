package rancherdesktop

import (
	"errors"
	"fmt"
	"github.com/Masterminds/semver/v3"
)

// Rule represents a constraint on which Versions are supported that
// applies to instances of Rancher Desktop that satisfy specific criteria.
type Rule struct {
	Criteria    Criteria
	Constraints Constraints
}

// Criteria is the conditions that are used to determine whether a Rule
// applies for a given client. All parts of Criteria must be satisfied for
// the Rule to apply to the client.
type Criteria struct {
	AppVersion      *semver.Constraints
	Platform        string
	Arch            string
	PlatformVersion *semver.Constraints
}

// Constraints contains logic that is applied to a Version to determine
// the value of its Supported key. All parts of Constraints must be satisfied
// for a Version to be supported.
type Constraints struct {
	Version *semver.Constraints
}

// Validate a Rule. Special attention is paid to fields of type
// *semver.Constraints, because when parsing a Rule from JSON, a field of
// this type that is not present is set to nil.
func (rule Rule) Validate() error {
	// validate Criteria.AppVersion
	if rule.Criteria.AppVersion == nil {
		return fmt.Errorf("invalid Criteria.AppVersion %q", rule.Criteria.AppVersion)
	}

	// validate Criteria.Platform
	if rule.Criteria.Platform != "*" && !validPlatform[rule.Criteria.Platform] {
		return fmt.Errorf("invalid Criteria.Platform %q", rule.Criteria.Platform)
	}

	// validate Criteria.Arch
	if rule.Criteria.Arch != "*" && !validArch[rule.Criteria.Arch] {
		return fmt.Errorf("invalid Criteria.Arch %q", rule.Criteria.Arch)
	}

	// validate Criteria.PlatformVersion
	if rule.Criteria.PlatformVersion == nil {
		return fmt.Errorf("invalid Criteria.PlatformVersion %q", rule.Criteria.PlatformVersion)
	}
	if rule.Criteria.Platform == "*" && rule.Criteria.PlatformVersion.String() != "*" {
		return errors.New("Criteria.Platform must be specified if Criteria.PlatformVersion is specified")
	}

	// validate Constraints.Version
	if rule.Constraints.Version == nil {
		return fmt.Errorf("invalid Constraints.Version %q", rule.Constraints.Version)
	}

	return nil
}

// AppliesTo returns true if a Rule applies to a client, which is represented by
// an InstanceInfo, and false otherwise.
func (rule Rule) AppliesTo(instanceInfo InstanceInfo) bool {
	if !rule.Criteria.AppVersion.Check(instanceInfo.AppVersion) {
		return false
	}

	if rule.Criteria.Platform != "*" && rule.Criteria.Platform != instanceInfo.Platform {
		return false
	}

	if rule.Criteria.Arch != "*" && rule.Criteria.Arch != instanceInfo.Arch {
		return false
	}

	if !rule.Criteria.PlatformVersion.Check(instanceInfo.PlatformVersion) {
		return false
	}

	return true
}

// Supported applies Rule.Constraints to a Version in order to determine
// whether that Version is supported.
func (rule Rule) Supported(version Version) (bool, error) {
	parsedVersion, err := semver.NewVersion(version.Name)
	if err != nil {
		return false, fmt.Errorf("failed to parse version %q: %w", version.Name, err)
	}
	return rule.Constraints.Version.Check(parsedVersion), nil
}
