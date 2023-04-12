package rancherdesktop

import (
	"fmt"
	"strings"
	"testing"
)

func newCheckUpgradeRequest(appVersion, platform, platformVersion string) *CheckUpgradeRequest {
	return &CheckUpgradeRequest{
		AppVersion: appVersion,
		ExtraInfo: map[string]string{
			"platform":        platform,
			"platformVersion": platformVersion,
		},
	}
}

func TestNewInstanceInfo(t *testing.T) {
	t.Run("should produce a valid InstanceInfo struct with valid input", func(t *testing.T) {
		appVersion := "1.2.3"
		platform := "darwin"
		arch := "x64"
		inputPlatform := fmt.Sprintf("%s-%s", platform, arch)
		platformVersion := "12.0.3"
		checkUpgradeRequest := newCheckUpgradeRequest(appVersion, inputPlatform, platformVersion)
		instanceInfo, err := NewInstanceInfo(checkUpgradeRequest)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
		}
		if instanceInfo.AppVersion.String() != appVersion {
			t.Errorf("expected instanceInfo.AppVersion %q but got %q",
				instanceInfo.AppVersion, appVersion)
		}
		if instanceInfo.Platform != platform {
			t.Errorf("expected instanceInfo.Platform %q but got %q",
				instanceInfo.Platform, platform)
		}
		if instanceInfo.Arch != arch {
			t.Errorf("expected instanceInfo.Arch %q but got %q",
				instanceInfo.Arch, arch)
		}
		if instanceInfo.PlatformVersion.String() != platformVersion {
			t.Errorf("expected instanceInfo.PlatformVersion %q but got %q",
				instanceInfo.PlatformVersion, platformVersion)
		}
	})

	testCases := []struct {
		Description         string
		CheckUpgradeRequest *CheckUpgradeRequest
		ExpectedError       string
	}{
		{
			Description:         "should fail if CheckUpgradeRequest.AppVersion is not valid semver",
			CheckUpgradeRequest: newCheckUpgradeRequest("asdf", "darwin-x64", "12.0.3"),
			ExpectedError:       "failed to parse AppVersion as semver",
		},
		{
			Description:         "should fail if CheckUpgradeRequest.ExtraInfo.platform is not valid",
			CheckUpgradeRequest: newCheckUpgradeRequest("1.2.3", "darwin-x64-somethingElse", "12.0.3"),
			ExpectedError:       "invalid extraInfo.platform",
		},
		{
			Description:         "should fail if CheckUpgradeRequest.ExtraInfo.platform is not valid",
			CheckUpgradeRequest: newCheckUpgradeRequest("1.2.3", "darwinx64", "12.0.3"),
			ExpectedError:       "invalid extraInfo.platform",
		},
		{
			Description:         "should fail if CheckUpgradeRequest.ExtraInfo.platform is not valid",
			CheckUpgradeRequest: newCheckUpgradeRequest("1.2.3", "", "12.0.3"),
			ExpectedError:       "invalid extraInfo.platform",
		},
		{
			Description: "should fail if CheckUpgradeRequest.ExtraInfo.platform is not present",
			CheckUpgradeRequest: &CheckUpgradeRequest{
				AppVersion: "1.2.3",
				ExtraInfo: map[string]string{
					"platformVersion": "12.0.3",
				},
			},
			ExpectedError: "extraInfo.platform not present",
		},
		{
			Description:         "should fail if parsed platform is not valid",
			CheckUpgradeRequest: newCheckUpgradeRequest("1.2.3", "darn-x64", "12.0.3"),
			ExpectedError:       "invalid platform",
		},
		{
			Description:         "should fail if parsed arch is not valid",
			CheckUpgradeRequest: newCheckUpgradeRequest("1.2.3", "darwin-mips", "12.0.3"),
			ExpectedError:       "invalid arch",
		},
		{
			Description: "should fail if CheckUpgradeRequest.ExtraInfo.platformVersion is not present",
			CheckUpgradeRequest: &CheckUpgradeRequest{
				AppVersion: "1.2.3",
				ExtraInfo: map[string]string{
					"platform": "darwin-x64",
				},
			},
			ExpectedError: "extraInfo.platformVersion not present",
		},
		{
			Description:         "should fail if CheckUpgradeRequest.ExtraInfo.platformVersion is empty",
			CheckUpgradeRequest: newCheckUpgradeRequest("1.2.3", "darwin-x64", ""),
			ExpectedError:       "failed to parse platformVersion",
		},
		{
			Description:         "should fail if CheckUpgradeRequest.ExtraInfo.platformVersion is not valid semver",
			CheckUpgradeRequest: newCheckUpgradeRequest("1.2.3", "darwin-x64", "notValidSemver"),
			ExpectedError:       "failed to parse platformVersion",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Description, func(t *testing.T) {
			_, err := NewInstanceInfo(testCase.CheckUpgradeRequest)
			if err == nil {
				t.Errorf("no error produced while parsing %#v", testCase.CheckUpgradeRequest)
			} else if !strings.HasPrefix(err.Error(), testCase.ExpectedError) {
				t.Errorf("error %q does not contain %q", err, testCase.ExpectedError)
			}
		})
	}
}
