package upgraderesponder

import (
	rd "github.com/longhorn/upgrade-responder/rancherdesktop"
	"testing"
)

var testConfig rd.ResponseConfig

func init() {
	config, err := rd.ReadConfig("../rancherdesktop/testdata/test-config.json")
	if err != nil {
		panic(err)
	}
	testConfig = config
}

func getTestServer(t *testing.T, config rd.ResponseConfig) *Server {
	server := &Server{
		DefaultVersions: config.Versions,
	}

	if err := server.generatePrecomputedVersions(config); err != nil {
		t.Fatalf("failed to generate precomputed versions: %s", err)
	}
	return server
}

func countSupported(versions []rd.Version) (supportedCount, unsupportedCount int) {
	for _, version := range versions {
		if version.Supported {
			supportedCount++
		} else {
			unsupportedCount++
		}
	}
	return
}

func TestServer(t *testing.T) {

	t.Run("GenerateCheckUpgradeResponse", func(t *testing.T) {

		t.Run("all Version.Supported should be true when request cannot be parsed to InstanceInfo", func(t *testing.T) {
			server := getTestServer(t, testConfig)
			checkUpgradeRequest := rd.CheckUpgradeRequest{
				AppVersion: "1.2.3",
				ExtraInfo: map[string]string{
					"platform": "darwin-x64",
				},
			}
			checkUpgradeResponse, err := server.GenerateCheckUpgradeResponse(checkUpgradeRequest)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			supportedCount, unsupportedCount := countSupported(checkUpgradeResponse.Versions)
			if supportedCount != 3 || unsupportedCount != 0 {
				t.Errorf("unexpected supportedCount %d or unsupportedCount %d", supportedCount, unsupportedCount)
			}
		})

		t.Run("all Version.Supported should be true when request does not match any Rules", func(t *testing.T) {
			server := getTestServer(t, testConfig)
			checkUpgradeRequest := rd.CheckUpgradeRequest{
				AppVersion: "2.0.0",
				ExtraInfo: map[string]string{
					"platform":        "darwin-x64",
					"platformVersion": "12.0.3",
				},
			}
			checkUpgradeResponse, err := server.GenerateCheckUpgradeResponse(checkUpgradeRequest)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			supportedCount, unsupportedCount := countSupported(checkUpgradeResponse.Versions)
			if supportedCount != 3 || unsupportedCount != 0 {
				t.Errorf("unexpected supportedCount %d or unsupportedCount %d", supportedCount, unsupportedCount)
			}
		})

		t.Run("values of Version.Supported should be correct when request matches a Rule", func(t *testing.T) {
			type TestCase struct {
				AppVersion               string
				ExpectedSupportedCount   int
				ExpectedUnsupportedCount int
			}
			testCases := []TestCase{
				{
					AppVersion:               "0.9.0",
					ExpectedSupportedCount:   1,
					ExpectedUnsupportedCount: 2,
				},
				{
					AppVersion:               "3.5.0",
					ExpectedSupportedCount:   2,
					ExpectedUnsupportedCount: 1,
				},
			}
			for _, testCase := range testCases {
				server := getTestServer(t, testConfig)
				checkUpgradeRequest := rd.CheckUpgradeRequest{
					AppVersion: testCase.AppVersion,
					ExtraInfo: map[string]string{
						"platform":        "darwin-x64",
						"platformVersion": "12.0.3",
					},
				}
				checkUpgradeResponse, err := server.GenerateCheckUpgradeResponse(checkUpgradeRequest)
				if err != nil {
					t.Fatalf("unexpected error: %s", err)
				}
				supportedCount, unsupportedCount := countSupported(checkUpgradeResponse.Versions)
				if supportedCount != testCase.ExpectedSupportedCount ||
					unsupportedCount != testCase.ExpectedUnsupportedCount {
					t.Errorf("unexpected supportedCount %d or unsupportedCount %d", supportedCount, unsupportedCount)
				}
			}
		})

		t.Run("the first Rule that matches should be used to set the values of Version.Supported", func(t *testing.T) {
			config, err := rd.ReadConfig("testdata/same-constraint-config.json")
			if err != nil {
				t.Fatalf("unexpected error parsing config: %s", err)
			}
			server := getTestServer(t, config)
			checkUpgradeRequest := rd.CheckUpgradeRequest{
				AppVersion: "1.0.0",
				ExtraInfo: map[string]string{
					"platform":        "darwin-x64",
					"platformVersion": "12.0.3",
				},
			}
			checkUpgradeResponse, err := server.GenerateCheckUpgradeResponse(checkUpgradeRequest)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			supportedCount, unsupportedCount := countSupported(checkUpgradeResponse.Versions)
			if supportedCount != 1 || unsupportedCount != 2 {
				t.Fatalf("unexpected supportedCount %d or unsupportedCount %d", supportedCount, unsupportedCount)
			}
		})
	})
}
