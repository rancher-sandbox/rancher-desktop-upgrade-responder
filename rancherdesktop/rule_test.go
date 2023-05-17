package rancherdesktop

import (
	"strings"
	"testing"

	"github.com/Masterminds/semver/v3"
)

func newRule(t *testing.T, appVersion, platform, arch, platformVersion, constraintsVersion string) Rule {
	parsedAppVersion, err := semver.NewConstraint(appVersion)
	if err != nil {
		t.Fatalf("failed to parse appVersion %q: %s", appVersion, err)
	}
	parsedPlatformVersion, err := semver.NewConstraint(platformVersion)
	if err != nil {
		t.Fatalf("failed to parse platformVersion %q: %s", platformVersion, err)
	}
	parsedConstraintsVersion, err := semver.NewConstraint(constraintsVersion)
	if err != nil {
		t.Fatalf("failed to parse constraintsVersion %q: %s", constraintsVersion, err)
	}
	return Rule{
		Criteria: Criteria{
			AppVersion:      parsedAppVersion,
			Platform:        platform,
			Arch:            arch,
			PlatformVersion: parsedPlatformVersion,
		},
		Constraints: Constraints{
			Version: parsedConstraintsVersion,
		},
	}
}

func newInstanceInfo(t *testing.T, appVersion, platform, arch, platformVersion string) InstanceInfo {
	parsedAppVersion, err := semver.NewVersion(appVersion)
	if err != nil {
		t.Fatalf("failed to parse appVersion %q: %s", appVersion, err)
	}
	if platform == "" {
		t.Fatal("must specify platform")
	}
	if arch == "" {
		t.Fatal("must specify arch")
	}
	parsedPlatformVersion, err := semver.NewVersion(platformVersion)
	if err != nil {
		t.Fatalf("failed to parse platformVersion %q: %s", platformVersion, err)
	}
	return InstanceInfo{
		AppVersion:      parsedAppVersion,
		Platform:        platform,
		Arch:            arch,
		PlatformVersion: parsedPlatformVersion,
	}
}

func TestRule(t *testing.T) {

	t.Run(".Validate", func(t *testing.T) {
		t.Run("should return nil if Rule is valid", func(t *testing.T) {
			rules := []Rule{
				newRule(t, "*", "linux", "x64", "*", "*"),
				newRule(t, "*", "darwin", "x64", "*", "*"),
				newRule(t, "*", "win32", "x64", "*", "*"),
				newRule(t, "*", "darwin", "arm64", "*", "*"),
			}
			for _, rule := range rules {
				err := rule.Validate()
				if err != nil {
					t.Errorf("unexpected error %q for Rule %#v", err, rule)
				}
			}
		})

		// Test cases that should return errors
		wildcardConstraint, _ := semver.NewConstraint("*")
		testCases := []struct {
			Description   string
			Rule          Rule
			ExpectedError string
		}{
			{
				Description: "should return error if Criteria.AppVersion is nil",
				Rule: Rule{
					Criteria: Criteria{
						AppVersion:      nil,
						Platform:        "darwin",
						Arch:            "x64",
						PlatformVersion: wildcardConstraint,
					},
				},
				ExpectedError: "invalid Criteria.AppVersion",
			},
			{
				Description:   "should return error if Criteria.Platform is invalid",
				Rule:          newRule(t, "*", "weirdPlatform", "x64", "*", "*"),
				ExpectedError: "invalid Criteria.Platform",
			},
			{
				Description:   "should return error if Criteria.Platform is empty",
				Rule:          newRule(t, "*", "", "x64", "*", "*"),
				ExpectedError: "invalid Criteria.Platform",
			},
			{
				Description:   "should return error if Criteria.Platform is *",
				Rule:          newRule(t, "*", "darwin", "", "*", "*"),
				ExpectedError: "invalid Criteria.Arch",
			},
			{
				Description:   "should return error if Criteria.Arch is invalid",
				Rule:          newRule(t, "*", "darwin", "weirdArch", "*", "*"),
				ExpectedError: "invalid Criteria.Arch",
			},
			{
				Description:   "should return error if Criteria.Arch is empty",
				Rule:          newRule(t, "*", "darwin", "", "*", "*"),
				ExpectedError: "invalid Criteria.Arch",
			},
			{
				Description: "should return error if Criteria.PlatformVersion is nil",
				Rule: Rule{
					Criteria: Criteria{
						AppVersion:      wildcardConstraint,
						Platform:        "darwin",
						Arch:            "x64",
						PlatformVersion: nil,
					},
				},
				ExpectedError: "invalid Criteria.PlatformVersion",
			},
			{
				Description: "should return error if Constraints.Version is nil",
				Rule: Rule{
					Criteria: Criteria{
						AppVersion:      wildcardConstraint,
						Platform:        "darwin",
						Arch:            "x64",
						PlatformVersion: wildcardConstraint,
					},
					Constraints: Constraints{
						Version: nil,
					},
				},
				ExpectedError: "invalid Constraints.Version",
			},
		}
		for _, testCase := range testCases {
			t.Run(testCase.Description, func(t *testing.T) {
				err := testCase.Rule.Validate()
				if err == nil {
					t.Errorf("no error produced while validating invalid Rule %#v", testCase.Rule)
				} else if !strings.HasPrefix(err.Error(), testCase.ExpectedError) {
					t.Errorf("error %q does not contain %q", err, testCase.ExpectedError)
				}
			})
		}
	})

	t.Run(".Test", func(t *testing.T) {
		testCases := []struct {
			Description    string
			Rule           Rule
			InstanceInfo   InstanceInfo
			ExpectedReturn bool
		}{
			{
				Description:    "should always return true when all fields other than Platform are *",
				Rule:           newRule(t, "*", "darwin", "*", "*", "*"),
				InstanceInfo:   newInstanceInfo(t, "1.2.3", "darwin", "x64", "2.3.45"),
				ExpectedReturn: true,
			},
			{
				Description:    "should always return true when all fields other than Platform are *",
				Rule:           newRule(t, "*", "darwin", "*", "*", "*"),
				InstanceInfo:   newInstanceInfo(t, "1.2.3", "darwin", "arm64", "2.3.45"),
				ExpectedReturn: true,
			},
			{
				Description:    "should always return true when all fields other than Platform are *",
				Rule:           newRule(t, "*", "darwin", "*", "*", "*"),
				InstanceInfo:   newInstanceInfo(t, "1.2.3", "darwin", "testArch", "0.0.0"),
				ExpectedReturn: true,
			},
			{
				Description:    "should always return true when all fields other than Platform are *",
				Rule:           newRule(t, "*", "darwin", "*", "*", "*"),
				InstanceInfo:   newInstanceInfo(t, "11.22.33", "darwin", "testArch", "2.3.45"),
				ExpectedReturn: true,
			},
			{
				Description:    "should always return true when all fields other than Platform are *",
				Rule:           newRule(t, "*", "darwin", "*", "*", "*"),
				InstanceInfo:   newInstanceInfo(t, "1.2.3", "darwin", "testArch", "11.22.333"),
				ExpectedReturn: true,
			},
			{
				Description:    "should return true if AppVersion criterion is satisfied",
				Rule:           newRule(t, "<=1.8.0", "darwin", "*", "*", "*"),
				InstanceInfo:   newInstanceInfo(t, "1.2.3", "darwin", "x64", "2.3.45"),
				ExpectedReturn: true,
			},
			{
				Description:    "should return false if AppVersion criterion is not satisfied",
				Rule:           newRule(t, ">=1.8.0", "darwin", "*", "*", "*"),
				InstanceInfo:   newInstanceInfo(t, "1.2.3", "darwin", "x64", "2.3.45"),
				ExpectedReturn: false,
			},
			{
				Description:    "should return true if Platform is equal",
				Rule:           newRule(t, "*", "darwin", "*", "*", "*"),
				InstanceInfo:   newInstanceInfo(t, "1.2.3", "darwin", "x64", "2.3.45"),
				ExpectedReturn: true,
			},
			{
				Description:    "should return false if Platform is not equal",
				Rule:           newRule(t, "*", "darwin", "*", "*", "*"),
				InstanceInfo:   newInstanceInfo(t, "1.2.3", "win32", "x64", "2.3.45"),
				ExpectedReturn: false,
			},
			{
				Description:    "should return true if Arch is equal",
				Rule:           newRule(t, "*", "darwin", "x64", "*", "*"),
				InstanceInfo:   newInstanceInfo(t, "1.2.3", "darwin", "x64", "2.3.45"),
				ExpectedReturn: true,
			},
			{
				Description:    "should return false if Arch is not equal",
				Rule:           newRule(t, "*", "darwin", "x64", "*", "*"),
				InstanceInfo:   newInstanceInfo(t, "1.2.3", "darwin", "arm64", "2.3.45"),
				ExpectedReturn: false,
			},
			{
				Description:    "should return true for any value of PlatformVersion when Platform is Linux",
				Rule:           newRule(t, "*", "linux", "*", ">0.0.0", "*"),
				InstanceInfo:   newInstanceInfo(t, "1.2.3", "linux", "x64", "1.2.3"),
				ExpectedReturn: true,
			},
			{
				Description:    "should return true for any value of PlatformVersion when Platform is Linux",
				Rule:           newRule(t, "*", "linux", "*", ">0.0.0", "*"),
				InstanceInfo:   newInstanceInfo(t, "1.2.3", "linux", "x64", "12.13.23"),
				ExpectedReturn: true,
			},
			{
				Description:    "should return true for any value of PlatformVersion when Platform is Linux",
				Rule:           newRule(t, "*", "linux", "*", ">0.0.0", "*"),
				InstanceInfo:   newInstanceInfo(t, "1.2.3", "linux", "x64", "0.0.0"),
				ExpectedReturn: true,
			},
			{
				Description:    "should return true if PlatformVersion criterion is satisfied",
				Rule:           newRule(t, "*", "darwin", "*", ">1.2.3", "*"),
				InstanceInfo:   newInstanceInfo(t, "1.2.3", "darwin", "x64", "2.3.45"),
				ExpectedReturn: true,
			},
			{
				Description:    "should return false if PlatformVersion criterion is not satisfied",
				Rule:           newRule(t, "*", "darwin", "*", "<1.2.3", "*"),
				InstanceInfo:   newInstanceInfo(t, "1.2.3", "darwin", "x64", "2.3.45"),
				ExpectedReturn: false,
			},
		}
		for _, testCase := range testCases {
			t.Run(testCase.Description, func(t *testing.T) {
				result := testCase.Rule.AppliesTo(testCase.InstanceInfo)
				if result != testCase.ExpectedReturn {
					t.Errorf("got result %t but expected %t\nRule: %#v\nInstanceInfo: %#v",
						result, testCase.ExpectedReturn, testCase.Rule, testCase.InstanceInfo)
				}
			})
		}
	})

	t.Run(".Supported", func(t *testing.T) {
		testCases := []struct {
			Description    string
			Rule           Rule
			Version        Version
			ExpectedReturn bool
		}{
			{
				Description: "should return false for a version that does not satisfy the Version constraint",
				Rule:        newRule(t, "*", "darwin", "*", "*", ">2.0.0"),
				Version: Version{
					Name:        "v1.2.3",
					ReleaseDate: "2022-07-28T11:00:00Z",
				},
				ExpectedReturn: false,
			},
			{
				Description: "should return true for a version that satisfies the Version constraint",
				Rule:        newRule(t, "*", "darwin", "*", "*", "<2.0.0"),
				Version: Version{
					Name:        "v1.2.3",
					ReleaseDate: "2022-07-28T11:00:00Z",
				},
				ExpectedReturn: true,
			},
		}
		for _, testCase := range testCases {
			t.Run(testCase.Description, func(t *testing.T) {
				supported, err := testCase.Rule.Supported(testCase.Version)
				if err != nil {
					t.Errorf("unexpected error %q\nRule: %#v\nVersion: %#v",
						err, testCase.Rule, testCase.Version)
				}
				if supported != testCase.ExpectedReturn {
					t.Errorf("result %t did not match expected %t\nRule: %#v\nVersion%#v",
						supported, testCase.ExpectedReturn, testCase.Rule, testCase.Version)
				}
			})
		}

		t.Run("should return an error if Version.Name is not valid semver", func(t *testing.T) {
			expectedError := "failed to parse version"
			version := Version{
				Name:        "invalidSemver",
				ReleaseDate: "2022-07-28T11:00:00Z",
			}
			rule := newRule(t, "*", "darwin", "*", "*", "<2.0.0")
			_, err := rule.Supported(version)
			if err == nil {
				t.Errorf("did not return error\nRule: %#v\nVersion: %#v", rule, version)
			}
			if !strings.Contains(err.Error(), expectedError) {
				t.Errorf("error %q does not contain %q", err, expectedError)
			}
		})
	})
}
