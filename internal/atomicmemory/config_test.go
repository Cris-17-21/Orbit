package atomicmemory

import (
	"strings"
	"testing"
)

func TestDefaultConfigIsDisabled(t *testing.T) {
	config := DefaultConfig()
	if config.Mode != ModeDisabled {
		t.Fatalf("default mode = %q, want %q", config.Mode, ModeDisabled)
	}
	if err := config.Validate(); err != nil {
		t.Fatalf("default config is invalid: %v", err)
	}
}

func TestPersonalHostedCloudConfigRoundTripsWithoutCredentialsOrIdentity(t *testing.T) {
	config := Config{
		SchemaVersion: ConfigSchemaVersion,
		Mode:          ModePersonal,
		Backend:       BackendHostedCloud,
	}

	encoded, err := config.Marshal()
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if strings.Contains(strings.ToLower(string(encoded)), "api_key") || strings.Contains(string(encoded), "user_id") {
		t.Fatalf("serialized config includes a credential or personal identity: %s", encoded)
	}

	decoded, err := ParseConfig(encoded)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if decoded != config {
		t.Fatalf("round trip = %#v, want %#v", decoded, config)
	}
}

func TestPersonalConnectedLocalConfigAllowsLoopbackHTTP(t *testing.T) {
	config := Config{
		SchemaVersion: ConfigSchemaVersion,
		Mode:          ModePersonal,
		Backend:       BackendConnectedLocal,
		APIURL:        "http://127.0.0.1:17350",
	}
	if err := config.Validate(); err != nil {
		t.Fatalf("connected-local config is invalid: %v", err)
	}
}

func TestConfigRejectsUnsupportedSharedMode(t *testing.T) {
	config := Config{SchemaVersion: ConfigSchemaVersion, Mode: Mode("shared"), Backend: BackendHostedCloud}
	if err := config.Validate(); err == nil {
		t.Fatal("shared mode validated before the two-collaborator pilot")
	}
}

func TestConfigRequiresSupportedSchemaVersion(t *testing.T) {
	config := Config{SchemaVersion: ConfigSchemaVersion + 1, Mode: ModeDisabled}
	if err := config.Validate(); err == nil {
		t.Fatal("accepted an unsupported config schema version")
	}
}

func TestDisabledConfigRejectsBackendSettings(t *testing.T) {
	config := Config{SchemaVersion: ConfigSchemaVersion, Mode: ModeDisabled, Backend: BackendHostedCloud}
	if err := config.Validate(); err == nil {
		t.Fatal("accepted backend settings while Atomic Memory is disabled")
	}
}

func TestSelfHostedConfigRequiresHTTPS(t *testing.T) {
	config := Config{SchemaVersion: ConfigSchemaVersion, Mode: ModePersonal, Backend: BackendSelfHosted, APIURL: "http://memory.example"}
	if err := config.Validate(); err == nil {
		t.Fatal("accepted a non-HTTPS self-hosted endpoint")
	}
}

func TestConfigRejectsTrailingJSON(t *testing.T) {
	if _, err := ParseConfig([]byte(`{"schema_version":1,"mode":"disabled"}{}`)); err == nil {
		t.Fatal("accepted trailing JSON data")
	}
}

func TestConfigRejectsCredentialsAndPersonalIdentityFields(t *testing.T) {
	for _, input := range []string{
		`{"schema_version":1,"mode":"personal","backend":"hosted-cloud","api_key":"secret"}`,
		`{"schema_version":1,"mode":"personal","backend":"hosted-cloud","user_id":"person-1"}`,
	} {
		if _, err := ParseConfig([]byte(input)); err == nil {
			t.Fatalf("parsed forbidden config field from %s", input)
		}
	}
}

func TestConfigRejectsUnsafeEndpoints(t *testing.T) {
	cases := []Config{
		{SchemaVersion: ConfigSchemaVersion, Mode: ModePersonal, Backend: BackendHostedCloud, APIURL: "https://attacker.example"},
		{SchemaVersion: ConfigSchemaVersion, Mode: ModePersonal, Backend: BackendConnectedLocal, APIURL: "http://memory.example"},
		{SchemaVersion: ConfigSchemaVersion, Mode: ModePersonal, Backend: BackendSelfHosted, APIURL: "https://user:password@memory.example"},
	}
	for _, config := range cases {
		if err := config.Validate(); err == nil {
			t.Errorf("accepted unsafe endpoint %#v", config)
		}
	}
}
