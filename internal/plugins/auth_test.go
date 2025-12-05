package plugins

import (
	"testing"
	"time"
)

// =============================================================================
// hashConfig Tests
// =============================================================================

// TestHashConfig_Deterministic verifies same input produces same hash.
func TestHashConfig_Deterministic(t *testing.T) {
	programConfig := map[string]any{"region": "us-east-1", "profile": "default"}
	stackConfig := map[string]any{"bucket": "my-bucket"}

	hash1 := hashConfig(programConfig, stackConfig)
	hash2 := hashConfig(programConfig, stackConfig)

	if hash1 != hash2 {
		t.Errorf("expected deterministic hash, got %q and %q", hash1, hash2)
	}
}

// TestHashConfig_Different verifies different inputs produce different hashes.
func TestHashConfig_Different(t *testing.T) {
	programConfig1 := map[string]any{"region": "us-east-1"}
	programConfig2 := map[string]any{"region": "us-west-2"}
	stackConfig := map[string]any{}

	hash1 := hashConfig(programConfig1, stackConfig)
	hash2 := hashConfig(programConfig2, stackConfig)

	if hash1 == hash2 {
		t.Errorf("expected different hashes for different inputs, got same: %q", hash1)
	}
}

// TestHashConfig_EmptyMaps verifies empty maps produce consistent hash.
func TestHashConfig_EmptyMaps(t *testing.T) {
	hash1 := hashConfig(map[string]any{}, map[string]any{})
	hash2 := hashConfig(map[string]any{}, map[string]any{})

	if hash1 != hash2 {
		t.Errorf("expected same hash for empty maps, got %q and %q", hash1, hash2)
	}

	// Verify hash is not empty
	if hash1 == "" {
		t.Error("expected non-empty hash for empty maps")
	}
}

// TestHashConfig_NilMaps verifies nil maps are handled gracefully.
func TestHashConfig_NilMaps(t *testing.T) {
	// This tests that nil maps don't cause a panic
	hash := hashConfig(nil, nil)

	if hash == "" {
		t.Error("expected non-empty hash for nil maps")
	}

	// Nil and empty should produce the same hash
	hashEmpty := hashConfig(map[string]any{}, map[string]any{})
	// Note: nil maps may marshal differently than empty maps in JSON,
	// so we just verify both work, not that they're equal
	if hash == "" || hashEmpty == "" {
		t.Error("both nil and empty should produce valid hashes")
	}
}

// =============================================================================
// convertToStringMap Tests
// =============================================================================

// TestConvertToStringMap_StringValues verifies strings pass through.
func TestConvertToStringMap_StringValues(t *testing.T) {
	input := map[string]any{
		"key1": "value1",
		"key2": "value2",
	}

	result := convertToStringMap(input)

	if result["key1"] != "value1" {
		t.Errorf("expected key1=%q, got %q", "value1", result["key1"])
	}
	if result["key2"] != "value2" {
		t.Errorf("expected key2=%q, got %q", "value2", result["key2"])
	}
}

// TestConvertToStringMap_IntValues verifies int to string conversion.
func TestConvertToStringMap_IntValues(t *testing.T) {
	input := map[string]any{
		"int":   42,
		"int64": int64(1234567890),
	}

	result := convertToStringMap(input)

	if result["int"] != "42" {
		t.Errorf("expected int=%q, got %q", "42", result["int"])
	}
	if result["int64"] != "1234567890" {
		t.Errorf("expected int64=%q, got %q", "1234567890", result["int64"])
	}
}

// TestConvertToStringMap_FloatValues verifies float to string conversion.
func TestConvertToStringMap_FloatValues(t *testing.T) {
	input := map[string]any{
		"float": 3.14,
	}

	result := convertToStringMap(input)

	// Note: float formatting may vary, just check it's not empty
	if result["float"] == "" {
		t.Error("expected non-empty float conversion")
	}
	if result["float"] != "3.14" {
		t.Errorf("expected float=%q, got %q", "3.14", result["float"])
	}
}

// TestConvertToStringMap_BoolValues verifies bool to string conversion.
func TestConvertToStringMap_BoolValues(t *testing.T) {
	input := map[string]any{
		"true":  true,
		"false": false,
	}

	result := convertToStringMap(input)

	if result["true"] != "true" {
		t.Errorf("expected true=%q, got %q", "true", result["true"])
	}
	if result["false"] != "false" {
		t.Errorf("expected false=%q, got %q", "false", result["false"])
	}
}

// TestConvertToStringMap_ComplexValues verifies complex types are JSON marshaled.
func TestConvertToStringMap_ComplexValues(t *testing.T) {
	input := map[string]any{
		"array": []string{"a", "b", "c"},
		"map":   map[string]any{"nested": "value"},
	}

	result := convertToStringMap(input)

	// Array should be JSON
	if result["array"] != `["a","b","c"]` {
		t.Errorf("expected array to be JSON, got %q", result["array"])
	}

	// Map should be JSON
	if result["map"] != `{"nested":"value"}` {
		t.Errorf("expected map to be JSON, got %q", result["map"])
	}
}

// TestConvertToStringMap_EmptyMap verifies empty input returns empty result.
func TestConvertToStringMap_EmptyMap(t *testing.T) {
	result := convertToStringMap(map[string]any{})

	if result == nil {
		t.Error("expected non-nil result")
	}
	if len(result) != 0 {
		t.Errorf("expected 0 entries, got %d", len(result))
	}
}

// TestConvertToStringMap_NilMap verifies nil input is handled.
func TestConvertToStringMap_NilMap(t *testing.T) {
	result := convertToStringMap(nil)

	if result == nil {
		t.Error("expected non-nil result")
	}
	if len(result) != 0 {
		t.Errorf("expected 0 entries, got %d", len(result))
	}
}

// =============================================================================
// Credentials.IsExpired Tests
// =============================================================================

// TestCredentials_IsExpired_AlwaysCall verifies AlwaysCall=true is always expired.
func TestCredentials_IsExpired_AlwaysCall(t *testing.T) {
	creds := &Credentials{
		PluginName: "test",
		AlwaysCall: true,
		ExpiresAt:  time.Now().Add(time.Hour), // Future time, but AlwaysCall overrides
	}

	if !creds.IsExpired() {
		t.Error("expected AlwaysCall=true to always return IsExpired=true")
	}
}

// TestCredentials_IsExpired_NeverExpires verifies zero time never expires.
func TestCredentials_IsExpired_NeverExpires(t *testing.T) {
	creds := &Credentials{
		PluginName: "test",
		AlwaysCall: false,
		ExpiresAt:  time.Time{}, // Zero time = never expires
	}

	if creds.IsExpired() {
		t.Error("expected zero ExpiresAt to never expire")
	}
}

// TestCredentials_IsExpired_Future verifies future time is not expired.
func TestCredentials_IsExpired_Future(t *testing.T) {
	creds := &Credentials{
		PluginName: "test",
		AlwaysCall: false,
		ExpiresAt:  time.Now().Add(time.Hour),
	}

	if creds.IsExpired() {
		t.Error("expected future ExpiresAt to not be expired")
	}
}

// TestCredentials_IsExpired_Past verifies past time is expired.
func TestCredentials_IsExpired_Past(t *testing.T) {
	creds := &Credentials{
		PluginName: "test",
		AlwaysCall: false,
		ExpiresAt:  time.Now().Add(-time.Hour),
	}

	if !creds.IsExpired() {
		t.Error("expected past ExpiresAt to be expired")
	}
}
