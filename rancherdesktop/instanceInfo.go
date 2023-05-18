package rancherdesktop

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
)

var validPlatform map[string]bool = map[string]bool{
	"win32":  true,
	"darwin": true,
	"linux":  true,
}

var validArch map[string]bool = map[string]bool{
	"x64":   true,
	"arm64": true,
}

type CheckUpgradeRequest struct {
	AppVersion string            `json:"appVersion"`
	ExtraInfo  map[string]string `json:"extraInfo"`
}

// InstanceInfo contains all the info we need about an instance of Rancher Desktop.
// Note that older versions of Rancher Desktop cannot be represented by
// an InstanceInfo struct, since they do not send certain required fields.
type InstanceInfo struct {
	AppVersion      *semver.Version
	Platform        string
	Arch            string
	PlatformVersion *semver.Version
}

// NewInstanceInfo converts the general CheckUpgradeRequest type into an InstanceInfo.
// If the CheckUpgradeRequest does not contain the needed info (which is optional in
// a CheckUpgradeRequest), an error is returned.
func NewInstanceInfo(checkUpgradeRequest CheckUpgradeRequest) (InstanceInfo, error) {
	appVersion, err := semver.NewVersion(checkUpgradeRequest.AppVersion)
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("failed to parse AppVersion as semver: %w", err)
	}

	platformAndArch, ok := checkUpgradeRequest.ExtraInfo["platform"]
	if !ok {
		return InstanceInfo{}, errors.New("extraInfo.platform not present")
	}
	components := strings.Split(platformAndArch, "-")
	if len(components) != 2 {
		return InstanceInfo{}, fmt.Errorf("invalid extraInfo.platform %q", platformAndArch)
	}

	platform := components[0]
	if !validPlatform[platform] {
		return InstanceInfo{}, fmt.Errorf("invalid platform %q", platform)
	}

	arch := components[1]
	if !validArch[arch] {
		return InstanceInfo{}, fmt.Errorf("invalid arch %q", arch)
	}

	rawPlatformVersion, ok := checkUpgradeRequest.ExtraInfo["platformVersion"]
	if !ok {
		return InstanceInfo{}, errors.New("extraInfo.platformVersion not present")
	}
	platformVersion, err := semver.NewVersion(rawPlatformVersion)
	if err != nil {
		err := fmt.Errorf("failed to parse platformVersion %q as semver: %w", platformVersion, err)
		return InstanceInfo{}, err
	}

	return InstanceInfo{
		AppVersion:      appVersion,
		Platform:        platform,
		Arch:            arch,
		PlatformVersion: platformVersion,
	}, nil
}
