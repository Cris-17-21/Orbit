package codexconfig

import (
	"strings"
	"testing"

	"github.com/Cris-17-21/Orbit/internal/atomicmemory"
	"github.com/pelletier/go-toml/v2"
)

func TestBuildConfigCreatesCodexMCPServerWithoutCredentials(t *testing.T) {
	config := atomicmemory.Config{
		SchemaVersion: atomicmemory.ConfigSchemaVersion,
		Mode:          atomicmemory.ModePersonal,
		Backend:       atomicmemory.BackendHostedCloud,
		APIURL:        "https://api.atomicstrata.ai",
	}

	got, err := BuildConfig(config)
	if err != nil {
		t.Fatalf("BuildConfig() error = %v", err)
	}
	var parsed codexConfig
	if err := toml.Unmarshal(got, &parsed); err != nil {
		t.Fatalf("BuildConfig() returned invalid TOML: %v\n%s", err, got)
	}
	server, ok := parsed.MCPServers[ServerName]
	if !ok {
		t.Fatalf("BuildConfig() has no %q MCP server: %#v", ServerName, parsed.MCPServers)
	}
	if server.Command != "npx" {
		t.Errorf("MCP command = %q, want npx", server.Command)
	}
	wantArgs := []string{"-y", "--package=@atomicmemory/mcp-server@0.1.6", "atomicmemory-mcp"}
	if !equalStrings(server.Args, wantArgs) {
		t.Errorf("MCP args = %#v, want %#v", server.Args, wantArgs)
	}
	if server.Env["ATOMICMEMORY_API_URL"] != "https://api.atomicstrata.ai" || server.Env["ATOMICMEMORY_SCOPE_LOCK"] != "true" {
		t.Errorf("Hosted Cloud environment = %#v", server.Env)
	}
	if !equalStrings(server.EnvVars, []string{"ATOMICMEMORY_API_KEY", "ATOMICMEMORY_SCOPE_USER"}) {
		t.Errorf("MCP env_vars = %#v, want credential names only", server.EnvVars)
	}
	if len(server.Env) != 4 {
		t.Errorf("MCP env includes unexpected values: %#v", server.Env)
	}
}

func TestBuildConfigUsesConfiguredSelfHostedEndpoint(t *testing.T) {
	config := atomicmemory.Config{
		SchemaVersion: atomicmemory.ConfigSchemaVersion,
		Mode:          atomicmemory.ModePersonal,
		Backend:       atomicmemory.BackendSelfHosted,
		APIURL:        "https://memory.example.test",
	}

	got, err := BuildConfig(config)
	if err != nil {
		t.Fatalf("BuildConfig() error = %v", err)
	}
	var parsed codexConfig
	if err := toml.Unmarshal(got, &parsed); err != nil {
		t.Fatalf("BuildConfig() returned invalid TOML: %v\n%s", err, got)
	}
	if parsed.MCPServers[ServerName].Env["ATOMICMEMORY_API_URL"] != "https://memory.example.test" {
		t.Fatalf("BuildConfig() does not use the configured endpoint: %#v", parsed.MCPServers[ServerName].Env)
	}
}

func TestBuildConfigUsesLocalCoreDefaultWithoutRemoteURL(t *testing.T) {
	config := atomicmemory.Config{
		SchemaVersion: atomicmemory.ConfigSchemaVersion,
		Mode:          atomicmemory.ModePersonal,
		Backend:       atomicmemory.BackendConnectedLocal,
	}

	got, err := BuildConfig(config)
	if err != nil {
		t.Fatalf("BuildConfig() error = %v", err)
	}
	var parsed codexConfig
	if err := toml.Unmarshal(got, &parsed); err != nil {
		t.Fatalf("BuildConfig() returned invalid TOML: %v\n%s", err, got)
	}
	if _, exists := parsed.MCPServers[ServerName].Env["ATOMICMEMORY_API_URL"]; exists {
		t.Fatalf("BuildConfig() should leave the documented Connected Local default unset: %#v", parsed.MCPServers[ServerName].Env)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestBuildConfigRejectsDisabledOrInvalidConfiguration(t *testing.T) {
	for _, config := range []atomicmemory.Config{
		atomicmemory.DefaultConfig(),
		{SchemaVersion: atomicmemory.ConfigSchemaVersion, Mode: atomicmemory.ModePersonal, Backend: atomicmemory.BackendSelfHosted},
	} {
		if _, err := BuildConfig(config); err == nil {
			t.Errorf("BuildConfig(%#v) succeeded, want validation error", config)
		}
	}
}

func TestParseServerNamesDiscardsCredentialAndIdentityFields(t *testing.T) {
	data := []byte(`[{"name":"atomicmemory","transport":{"env":{"ATOMICMEMORY_SCOPE_USER":"private-user","ATOMICMEMORY_API_KEY":"private-key"}}}]`)
	names, err := ParseServerNames(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 1 || names[0] != ServerName {
		t.Fatalf("ParseServerNames() = %#v", names)
	}
	if strings.Contains(strings.Join(names, " "), "private") {
		t.Fatal("sensitive fields appeared in parsed names")
	}
}

func TestParseServerNamesRejectsMalformedOrIncompleteOutput(t *testing.T) {
	for _, data := range [][]byte{[]byte("not json"), []byte(`[{"name":""}]`)} {
		if _, err := ParseServerNames(data); err == nil {
			t.Errorf("ParseServerNames(%q) succeeded", data)
		}
	}
}

func TestHasAtomicMemoryServerMatchesExactName(t *testing.T) {
	if !HasAtomicMemoryServer([]string{"other", ServerName}) {
		t.Fatal("did not detect the standard Atomic Memory server")
	}
	if HasAtomicMemoryServer([]string{"atomicmemory-dev"}) {
		t.Fatal("matched a nonstandard server name")
	}
}
