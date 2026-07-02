package upgraderesponder

import (
	"testing"
	"time"

	rd "github.com/longhorn/upgrade-responder/rancherdesktop"
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

func TestValidate(t *testing.T) {
	testCases := []struct {
		schema   Schema
		value    interface{}
		expected bool
	}{
		{
			schema:   Schema{DataType: "string", MaxLen: 5},
			value:    "1234",
			expected: true,
		},
		{
			schema:   Schema{DataType: "string", MaxLen: 5},
			value:    "123456",
			expected: false,
		},
		{
			schema:   Schema{DataType: "string"},
			value:    "123456",
			expected: true,
		},
		{
			schema:   Schema{DataType: "string"},
			value:    1234,
			expected: false,
		},
		{
			schema:   Schema{DataType: "string"},
			value:    1.0,
			expected: false,
		},
		{
			schema:   Schema{DataType: "float"},
			value:    1,
			expected: false,
		},
		{
			schema:   Schema{DataType: "float"},
			value:    "1",
			expected: false,
		},
		{
			schema:   Schema{DataType: "float"},
			value:    false,
			expected: false,
		},
		{
			schema:   Schema{DataType: "float"},
			value:    1.0,
			expected: true,
		},
		{
			schema:   Schema{DataType: "invalid"},
			value:    1.0,
			expected: false,
		},
		{
			schema:   Schema{DataType: "int"},
			value:    1.0,
			expected: false,
		},
		{
			schema:   Schema{DataType: "int"},
			value:    "1",
			expected: false,
		},
		{
			schema:   Schema{DataType: "int"},
			value:    1,
			expected: false,
		},
	}

	for i, testCase := range testCases {
		if output := testCase.schema.Validate(testCase.value); output != testCase.expected {
			t.Errorf("Test case %v: %+v Output %v not equal to expected %v", i, testCase, output, testCase.expected)
		}
	}
}

func TestValidateAndLoadRequestSchema(t *testing.T) {
	s := Server{}

	testCases := []struct {
		requestSchema RequestSchema
		expectedError bool
	}{
		{
			requestSchema: RequestSchema{AppVersionSchema: Schema{DataType: "float"}},
			expectedError: true,
		},
		{
			requestSchema: RequestSchema{AppVersionSchema: Schema{DataType: "string", MaxLen: -1}},
			expectedError: true,
		},
		{
			requestSchema: RequestSchema{AppVersionSchema: Schema{DataType: "string", MaxLen: 10}},
			expectedError: false,
		},
		{
			requestSchema: RequestSchema{
				AppVersionSchema: Schema{DataType: "string", MaxLen: 10},
				ExtraTagInfoSchema: map[string]Schema{
					"tag-1": {
						DataType: "boolean",
					},
				},
			},
			expectedError: true,
		},
		{
			requestSchema: RequestSchema{
				AppVersionSchema: Schema{DataType: "string", MaxLen: 10},
				ExtraTagInfoSchema: map[string]Schema{
					"tag-1": {DataType: "string"},
				},
				ExtraFieldInfoSchema: map[string]Schema{
					"field-1": {DataType: "string"},
					"field-2": {DataType: "float"},
					"field-3": {DataType: "int"},
				},
			},
			expectedError: true,
		},
		{
			requestSchema: RequestSchema{
				AppVersionSchema: Schema{DataType: "string", MaxLen: 10},
				ExtraTagInfoSchema: map[string]Schema{
					"tag-1": {DataType: "string"},
				},
				ExtraFieldInfoSchema: map[string]Schema{
					"field-1": {DataType: "string"},
					"field-2": {DataType: "float"},
					"field-3": {DataType: "boolean"},
				},
			},
			expectedError: false,
		},
	}

	boolToString := func(b bool) string {
		if b {
			return ""
		}
		return "no "
	}
	for i, testCase := range testCases {
		err := s.validateAndLoadRequestSchema(testCase.requestSchema)
		if testCase.expectedError != (err != nil) {
			t.Errorf("Test case %v : %v expected %verror but got %v", i, testCase, boolToString(testCase.expectedError), err)
		}
	}
}

func TestValidateExtraInfo(t *testing.T) {
	s := Server{}
	s.RequestSchema = RequestSchema{
		AppVersionSchema: Schema{DataType: "string", MaxLen: 10},
		ExtraTagInfoSchema: map[string]Schema{
			"tag-1": {DataType: "string", MaxLen: 5},
		},
		ExtraFieldInfoSchema: map[string]Schema{
			"field-1": {DataType: "string"},
			"field-2": {DataType: "float"},
			"field-3": {DataType: "boolean"},
		},
	}

	testCases := []struct {
		key           string
		value         interface{}
		extraInfoType string
		expected      bool
	}{
		{key: "tag-1", value: "1234", extraInfoType: extraInfoTypeTag, expected: true},
		{key: "tag-1", value: "123456", extraInfoType: extraInfoTypeTag, expected: false},
		{key: "tag-x", value: "1234", extraInfoType: extraInfoTypeTag, expected: false},
		{key: "field-1", value: "1234", extraInfoType: extraInfoTypeField, expected: true},
		{key: "field-1", value: 1234, extraInfoType: extraInfoTypeField, expected: false},
		{key: "field-x", value: 1234, extraInfoType: extraInfoTypeField, expected: false},
	}

	for i, testCase := range testCases {
		if output := s.ValidateExtraInfo(testCase.key, testCase.value, testCase.extraInfoType); output != testCase.expected {
			t.Errorf("Test case %v: %+v Output %v not equal to expected %v", i, testCase, output, testCase.expected)
		}
	}
}

func TestNewScarfService(t *testing.T) {
	tests := []struct {
		name                     string
		endpointTemplates        []string
		timeoutSeconds           int
		expectedEnabled          bool
		expectedEndpointTemplate map[string]struct{}
		expectedTimeout          time.Duration
		expectHTTPClient         bool
	}{
		{
			name: "trim empty entries and deduplicate templates",
			endpointTemplates: []string{
				"",
				"   ",
				"https://scarf.sh/a",
				" https://scarf.sh/a ",
				"https://scarf.sh/b",
				"\thttps://scarf.sh/b\t",
			},
			timeoutSeconds:  5,
			expectedEnabled: true,
			expectedEndpointTemplate: map[string]struct{}{
				"https://scarf.sh/a": {},
				"https://scarf.sh/b": {},
			},
			expectedTimeout:  5 * time.Second,
			expectHTTPClient: true,
		},
		{
			name: "disable service when no usable endpoint templates",
			endpointTemplates: []string{
				"",
				" ",
				"\n\t",
			},
			timeoutSeconds:           3,
			expectedEnabled:          false,
			expectedEndpointTemplate: nil,
			expectedTimeout:          0,
			expectHTTPClient:         false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewScarfService(tc.endpointTemplates, tc.timeoutSeconds)

			if svc == nil {
				t.Fatalf("expected service to be created, got nil")
			}

			if svc.enabled != tc.expectedEnabled {
				t.Fatalf("expected enabled=%v, got %v", tc.expectedEnabled, svc.enabled)
			}

			if len(svc.endpointTemplates) != len(tc.expectedEndpointTemplate) {
				t.Fatalf("expected %d endpoint templates, got %d",
					len(tc.expectedEndpointTemplate), len(svc.endpointTemplates))
			}

			for k := range tc.expectedEndpointTemplate {
				if _, ok := svc.endpointTemplates[k]; !ok {
					t.Fatalf("expected endpoint template %q not found", k)
				}
			}

			if svc.timeout != tc.expectedTimeout {
				t.Fatalf("expected timeout %v, got %v", tc.expectedTimeout, svc.timeout)
			}

			if tc.expectHTTPClient {
				if svc.httpClient == nil {
					t.Fatalf("expected httpClient to be initialized")
				}
				if svc.httpClient.Timeout != tc.expectedTimeout {
					t.Fatalf("expected httpClient timeout %v, got %v",
						tc.expectedTimeout, svc.httpClient.Timeout)
				}
			} else {
				if svc.httpClient != nil {
					t.Fatalf("expected httpClient to be nil")
				}
			}
		})
	}
}

func TestSubstituteTemplateVars(t *testing.T) {
	tests := []struct {
		name     string
		template string
		vars     map[string]string
		expected string
	}{
		{
			name:     "single variable",
			template: "https://example.com/{version}",
			vars: map[string]string{
				"version": "v1.0.0",
			},
			expected: "https://example.com/v1.0.0",
		},
		{
			name:     "multiple variables",
			template: "https://example.com/{version}/{longhornDistro}",
			vars: map[string]string{
				"version":        "v1.0.0",
				"longhornDistro": "oss",
			},
			expected: "https://example.com/v1.0.0/oss",
		},
		{
			name:     "unused variable",
			template: "https://example.com/{version}",
			vars: map[string]string{
				"version":        "v1.0.0",
				"longhornDistro": "oss",
			},
			expected: "https://example.com/v1.0.0",
		},
		{
			name:     "no variables",
			template: "https://example.com/static",
			vars:     map[string]string{},
			expected: "https://example.com/static",
		},
		{
			name:     "unknown variable replaced with empty string",
			template: "https://example.com/{version}/{unknown}",
			vars: map[string]string{
				"version": "v1.0.0",
			},
			expected: "https://example.com/v1.0.0/",
		},
		{
			name:     "empty variable value",
			template: "https://example.com/{version}/{longhornDistro}",
			vars: map[string]string{
				"version":        "v1.0.0",
				"longhornDistro": "",
			},
			expected: "https://example.com/v1.0.0/",
		},
		{
			name:     "same variable multiple times",
			template: "https://example.com/{version}/download/{version}",
			vars: map[string]string{
				"version": "v1.0.0",
			},
			expected: "https://example.com/v1.0.0/download/v1.0.0",
		},
		{
			name:     "adjacent variables",
			template: "https://example.com/{version}{longhornDistro}",
			vars: map[string]string{
				"version":        "v1.0.0",
				"longhornDistro": "oss",
			},
			expected: "https://example.com/v1.0.0oss",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := substituteTemplateVars(tt.template, tt.vars)
			if result != tt.expected {
				t.Fatalf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}
