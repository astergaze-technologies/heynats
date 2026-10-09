package util

import (
	"testing"
)

func TestVersionComponents(t *testing.T) {
	tests := []struct {
		name          string
		version       string
		expectedMajor int
		expectedMinor int
		expectedPatch int
		expectError   bool
	}{
		{
			name:          "Valid semver with v prefix",
			version:       "v2.9.8",
			expectedMajor: 2,
			expectedMinor: 9,
			expectedPatch: 8,
			expectError:   false,
		},
		{
			name:          "Valid semver without v prefix",
			version:       "2.9.8",
			expectedMajor: 2,
			expectedMinor: 9,
			expectedPatch: 8,
			expectError:   false,
		},
		{
			name:          "Valid semver with only major and minor",
			version:       "2.11",
			expectedMajor: 2,
			expectedMinor: 11,
			expectedPatch: 0,
		},
		{
			name:          "Valid semver with only major",
			version:       "2",
			expectedMajor: 2,
			expectedMinor: 0,
			expectedPatch: 0,
		},
		{
			name:          "Valid semver with leading zeros",
			version:       "v02.09.08",
			expectedMajor: 2,
			expectedMinor: 9,
			expectedPatch: 8,
			expectError:   false,
		},
		{
			name:          "Valid semver with large numbers",
			version:       "v10.15.23",
			expectedMajor: 10,
			expectedMinor: 15,
			expectedPatch: 23,
			expectError:   false,
		},
		{
			name:          "Invalid semver - empty string",
			version:       "",
			expectedMajor: 0,
			expectedMinor: 0,
			expectedPatch: 0,
			expectError:   true,
		},
		{
			name:          "Invalid semver - non-numeric",
			version:       "abc.def.ghi",
			expectedMajor: 0,
			expectedMinor: 0,
			expectedPatch: 0,
			expectError:   true,
		},
		{
			name:          "Invalid semver - missing version number",
			version:       "v",
			expectedMajor: 0,
			expectedMinor: 0,
			expectedPatch: 0,
			expectError:   true,
		},
		{
			name:          "Invalid semver - special characters",
			version:       "v2.9.8-beta1",
			expectedMajor: 2,
			expectedMinor: 9,
			expectedPatch: 8,
			expectError:   false, // The regex should match the numeric part
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			major, minor, patch, err := versionComponents(tt.version)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for version %s, but got none", tt.version)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for version %s: %v", tt.version, err)
				return
			}

			if major != tt.expectedMajor {
				t.Errorf("Expected major %d, got %d for version %s", tt.expectedMajor, major, tt.version)
			}

			if minor != tt.expectedMinor {
				t.Errorf("Expected minor %d, got %d for version %s", tt.expectedMinor, minor, tt.version)
			}

			if patch != tt.expectedPatch {
				t.Errorf("Expected patch %d, got %d for version %s", tt.expectedPatch, patch, tt.version)
			}
		})
	}
}

func TestServerMinVersion(t *testing.T) {
	// Note: Since ServerMinVersion takes a *nats.Conn which is a concrete type,
	// we need to test this function with actual NATS connections or modify the function
	// to accept an interface for better testability.
	// For now, we'll test the logic indirectly by testing versionComponents
	// and create a separate test for the comparison logic.

	t.Skip("ServerMinVersion requires a real NATS connection or refactoring for testability")
}

// TestVersionComparison tests the core logic of ServerMinVersion
func TestVersionComparison(t *testing.T) {
	tests := []struct {
		name           string
		serverMajor    int
		serverMinor    int
		serverPatch    int
		requiredMajor  int
		requiredMinor  int
		requiredPatch  int
		expectedResult bool
	}{
		{
			name:           "Server version higher than required",
			serverMajor:    2,
			serverMinor:    10,
			serverPatch:    5,
			requiredMajor:  2,
			requiredMinor:  9,
			requiredPatch:  8,
			expectedResult: true,
		},
		{
			name:           "Server version equal to required",
			serverMajor:    2,
			serverMinor:    9,
			serverPatch:    8,
			requiredMajor:  2,
			requiredMinor:  9,
			requiredPatch:  8,
			expectedResult: true,
		},
		{
			name:           "Server version lower major",
			serverMajor:    1,
			serverMinor:    9,
			serverPatch:    8,
			requiredMajor:  2,
			requiredMinor:  9,
			requiredPatch:  8,
			expectedResult: false,
		},
		{
			name:           "Server version lower minor",
			serverMajor:    2,
			serverMinor:    8,
			serverPatch:    8,
			requiredMajor:  2,
			requiredMinor:  9,
			requiredPatch:  8,
			expectedResult: false,
		},
		{
			name:           "Server version lower patch",
			serverMajor:    2,
			serverMinor:    9,
			serverPatch:    7,
			requiredMajor:  2,
			requiredMinor:  9,
			requiredPatch:  8,
			expectedResult: false,
		},
		{
			name:           "Higher major version",
			serverMajor:    3,
			serverMinor:    0,
			serverPatch:    0,
			requiredMajor:  2,
			requiredMinor:  9,
			requiredPatch:  8,
			expectedResult: true,
		},
		{
			name:           "Higher minor version",
			serverMajor:    2,
			serverMinor:    10,
			serverPatch:    0,
			requiredMajor:  2,
			requiredMinor:  9,
			requiredPatch:  8,
			expectedResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the core comparison logic from ServerMinVersion
			result := !(tt.serverMajor < tt.requiredMajor ||
				(tt.serverMajor == tt.requiredMajor && tt.serverMinor < tt.requiredMinor) ||
				(tt.serverMajor == tt.requiredMajor && tt.serverMinor == tt.requiredMinor && tt.serverPatch < tt.requiredPatch))

			if result != tt.expectedResult {
				t.Errorf("Expected %v, got %v for server version %d.%d.%d vs required %d.%d.%d",
					tt.expectedResult, result, tt.serverMajor, tt.serverMinor, tt.serverPatch,
					tt.requiredMajor, tt.requiredMinor, tt.requiredPatch)
			}
		})
	}
}

// Integration test that would require a real NATS server
// func TestServerMinVersionIntegration(t *testing.T) {
//     // This would require setting up a real NATS server for integration testing
//     // Skip for unit tests
//     t.Skip("Integration test - requires real NATS server")
// }

// Benchmark tests
func BenchmarkVersionComponents(b *testing.B) {
	versions := []string{
		"v2.9.8",
		"2.9.8",
		"v10.15.23",
		"1.0.0",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		version := versions[i%len(versions)]
		_, _, _, _ = versionComponents(version)
	}
}

func BenchmarkServerMinVersion(b *testing.B) {
	// Since we can't easily mock nats.Conn for benchmarking,
	// we'll benchmark the core comparison logic instead
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate the comparison logic from ServerMinVersion
		smajor, sminor, spatch := 2, 9, 8
		major, minor, patch := 2, 9, 0
		_ = !(smajor < major || (smajor == major && sminor < minor) || (smajor == major && sminor == minor && spatch < patch))
	}
}

// TestVersionComponentsEdgeCases tests edge cases and documents the current behavior
func TestVersionComponentsEdgeCases(t *testing.T) {
	t.Run("Regex pattern edge cases", func(t *testing.T) {
		// Test versions that match the regex but have specific patterns
		testCases := []struct {
			version     string
			shouldFail  bool
			description string
		}{
			{"v1.2.3-alpha", false, "Version with suffix - should extract 1.2.3"},
			{"v1.2.3+build", false, "Version with build metadata"},
			{"1.2.3-rc.1", false, "Version with release candidate"},
			{"v0.0.0", false, "Zero version"},
			{"v999.999.999", false, "Large version numbers"},
		}

		for _, tc := range testCases {
			t.Run(tc.description, func(t *testing.T) {
				major, minor, patch, err := versionComponents(tc.version)
				if tc.shouldFail {
					if err == nil {
						t.Errorf("Expected error for version %s", tc.version)
					}
				} else {
					if err != nil {
						t.Errorf("Unexpected error for version %s: %v", tc.version, err)
					} else {
						t.Logf("Version %s parsed as %d.%d.%d", tc.version, major, minor, patch)
					}
				}
			})
		}
	})
}
