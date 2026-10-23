package claudecodeconfig

import (
	"encoding/json"
	"fmt"

	"github.com/Cris-17-21/Orbit/internal/atomicmemory"
)

const (
	ServerName     = "atomicmemory"
	mcpPackage     = "@atomicmemory/mcp-server@0.1.6"
	mcpEntrypoint  = "atomicmemory-mcp"
	hostedCloudURL = "https://api.atomicstrata.ai"
)

type serverConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env"`
}

// BuildServerConfig returns one Claude Code project MCP entry. Credentials and
// collaborator identity are read from each person's local environment.
func BuildServerConfig(config atomicmemory.Config) ([]byte, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid Atomic Memory project config: %w", err)
	}
	if config.Mode != atomicmemory.ModePersonal {
		return nil, fmt.Errorf("Atomic Memory must be enabled in personal mode before configuring Claude Code")
	}

	env := map[string]string{
		"ATOMICMEMORY_API_KEY":     "${ATOMICMEMORY_API_KEY}",
		"ATOMICMEMORY_PROVIDER":    "atomicmemory",
		"ATOMICMEMORY_SCOPE_AGENT": "claude-code",
		"ATOMICMEMORY_SCOPE_LOCK":  "true",
		"ATOMICMEMORY_SCOPE_USER":  "${ATOMICMEMORY_SCOPE_USER}",
	}
	if config.APIURL != "" {
		env["ATOMICMEMORY_API_URL"] = config.APIURL
	} else if config.Backend == atomicmemory.BackendHostedCloud {
		env["ATOMICMEMORY_API_URL"] = hostedCloudURL
	}

	data, err := json.Marshal(serverConfig{
		Command: "npx",
		Args:    []string{"-y", "--package=" + mcpPackage, mcpEntrypoint},
		Env:     env,
	})
	if err != nil {
		return nil, fmt.Errorf("encode Claude Code MCP config: %w", err)
	}
	return data, nil
}
