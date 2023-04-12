package rancherdesktop

import (
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {

	t.Run(".Validate", func(t *testing.T) {

		t.Run("should return nil for a valid version", func(t *testing.T) {
			version := Version{
				Name:                 "v1.2.3",
				ReleaseDate:          "2022-07-28T11:00:00Z",
				MinUpgradableVersion: "",
				Tags:                 []string{"testTag"},
			}
			err := version.Validate()
			if err != nil {
				t.FailNow()
			}
		})

		t.Run("should return nil for a valid version with MinUpgradableVersion nonempty", func(t *testing.T) {
			version := Version{
				Name:                 "v1.2.3",
				ReleaseDate:          "2022-07-28T11:00:00Z",
				MinUpgradableVersion: "v2.3.4",
				Tags:                 []string{"testTag"},
			}
			err := version.Validate()
			if err != nil {
				t.FailNow()
			}
		})

		// Test error conditions
		testCases := []struct {
			Description   string
			Version       Version
			ExpectedError string
		}{
			{
				Description: "should return error if no tags are present",
				Version: Version{
					Name:                 "v1.2.3",
					ReleaseDate:          "2022-07-28T11:00:00Z",
					MinUpgradableVersion: "",
					Tags:                 []string{},
				},
				ExpectedError: "invalid empty label",
			},
			{
				Description: "should return error if Version.Name is not valid semver",
				Version: Version{
					Name:                 "invalidSemver",
					ReleaseDate:          "2022-07-28T11:00:00Z",
					MinUpgradableVersion: "",
					Tags:                 []string{"testTag"},
				},
				ExpectedError: "failed to parse Name",
			},
			{
				Description: "should return error if Version.MinUpgradableVersion is nonempty and not valid semver",
				Version: Version{
					Name:                 "v1.2.3",
					ReleaseDate:          "2022-07-28T11:00:00Z",
					MinUpgradableVersion: "invalidSemver",
					Tags:                 []string{"testTag"},
				},
				ExpectedError: "failed to parse MinUpgradableVersion",
			},
			{
				Description: "should return error if Version.ReleaseDate is not in RFC3339 format",
				Version: Version{
					Name:                 "v1.2.3",
					ReleaseDate:          "notValidRFC3339",
					MinUpgradableVersion: "",
					Tags:                 []string{"testTag"},
				},
				ExpectedError: "failed to parse ReleaseDate",
			},
		}
		for _, testCase := range testCases {
			t.Run(testCase.Description, func(t *testing.T) {
				err := testCase.Version.Validate()
				if err == nil {
					t.Errorf("did not return error for %#v", testCase.Version)
				} else if !strings.Contains(err.Error(), testCase.ExpectedError) {
					t.Errorf("error %q does not contain %q", err, testCase.ExpectedError)
				}
			})
		}
	})
}