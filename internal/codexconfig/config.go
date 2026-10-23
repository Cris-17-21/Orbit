package codexconfig

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Cris-17-21/Orbit/internal/atomicmemory"
	"github.com/pelletier/go-toml/v2"
)

const (
	ServerName     = "atomicmemory"
	mcpPackage     = "@atomicmemory/mcp-server@0.1.6"
	mcpEntrypoint  = "atomicmemory-mcp"
	hostedCloudURL = "https://api.atomicstrata.ai"
)

type codexConfig struct {
	MCPServers map[string]mcpServer `toml:"mcp_servers"`
}

type mcpServer struct {
	Command string            `toml:"command"`
	Args    []string          `toml:"args"`
	Env     map[string]string `toml:"env"`
	EnvVars []string          `toml:"env_vars"`
}

// BuildConfig returns a Codex project-scoped MCP config. Secret values and
// collaborator identity are forwarded from the local environment, never copied
// into the project file.
func BuildConfig(config atomicmemory.Config) ([]byte, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid Atomic Memory project config: %w", err)
	}
	if config.Mode != atomicmemory.ModePersonal {
		return nil, fmt.Errorf("Atomic Memory must be enabled in personal mode before configuring Codex")
	}

	env := map[string]string{
		"ATOMICMEMORY_PROVIDER":    "atomicmemory",
		"ATOMICMEMORY_SCOPE_AGENT": "codex",
		"ATOMICMEMORY_SCOPE_LOCK":  "true",
	}
	if config.APIURL != "" {
		env["ATOMICMEMORY_API_URL"] = config.APIURL
	} else if config.Backend == atomicmemory.BackendHostedCloud {
		env["ATOMICMEMORY_API_URL"] = hostedCloudURL
	}

	data, err := toml.Marshal(codexConfig{MCPServers: map[string]mcpServer{
		ServerName: {
			Command: "npx",
			Args:    []string{"-y", "--package=" + mcpPackage, mcpEntrypoint},
			Env:     env,
			EnvVars: []string{"ATOMICMEMORY_API_KEY", "ATOMICMEMORY_SCOPE_USER"},
		},
	}})
	if err != nil {
		return nil, fmt.Errorf("encode Codex MCP config: %w", err)
	}
	return data, nil
}

type listedServer struct {
	Name string `json:"name"`
}

// ParseServerNames reads only names from Codex's list output. The output can
// contain environment values and personal scope identifiers, which are
// intentionally discarded here.
func ParseServerNames(data []byte) ([]string, error) {
	var servers []listedServer
	if err := json.Unmarshal(data, &servers); err != nil {
		return nil, errors.New("Codex devolvió una lista de servidores no válida")
	}
	for _, server := range servers {
		if server.Name == "" {
			return nil, errors.New("Codex devolvió un servidor sin nombre")
		}
	}
	names := make([]string, 0, len(servers))
	for _, server := range servers {
		names = append(names, server.Name)
	}
	return names, nil
}

func HasAtomicMemoryServer(names []string) bool {
	for _, name := range names {
		if name == ServerName {
			return true
		}
	}
	return false
}
