package atomicmemory

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"
)

const ConfigSchemaVersion = 1

const hostedCloudAPIURL = "https://api.atomicstrata.ai"

type Mode string

const (
	ModeDisabled Mode = "disabled"
	ModePersonal Mode = "personal"
)

type Backend string

const (
	BackendHostedCloud    Backend = "hosted-cloud"
	BackendConnectedLocal Backend = "connected-local"
	BackendSelfHosted     Backend = "self-hosted"
)

// Config contains project-wide Atomic Memory settings safe to commit to Git.
// Credentials and collaborator identity belong in each person's local host config.
type Config struct {
	SchemaVersion int     `json:"schema_version"`
	Mode          Mode    `json:"mode"`
	Backend       Backend `json:"backend,omitempty"`
	APIURL        string  `json:"api_url,omitempty"`
}

func DefaultConfig() Config {
	return Config{SchemaVersion: ConfigSchemaVersion, Mode: ModeDisabled}
}

func (c Config) Validate() error {
	if c.SchemaVersion != ConfigSchemaVersion {
		return fmt.Errorf("unsupported Atomic Memory config schema version %d; expected %d", c.SchemaVersion, ConfigSchemaVersion)
	}
	if c.Mode == ModeDisabled {
		if c.Backend != "" || c.APIURL != "" {
			return errors.New("disabled Atomic Memory config must not define a backend or API URL")
		}
		return nil
	}
	if c.Mode != ModePersonal {
		return fmt.Errorf("unsupported Atomic Memory mode %q", c.Mode)
	}
	switch c.Backend {
	case BackendHostedCloud:
		if c.APIURL != "" && c.APIURL != hostedCloudAPIURL {
			return errors.New("Hosted Cloud API URL is fixed to https://api.atomicstrata.ai")
		}
	case BackendConnectedLocal:
		if c.APIURL != "" && !isLoopbackHTTP(c.APIURL) {
			return errors.New("Connected Local API URL must use HTTP on localhost or a loopback IP")
		}
	case BackendSelfHosted:
		if c.APIURL == "" {
			return errors.New("self-hosted Atomic Memory config requires an API URL")
		}
		if err := validateURL(c.APIURL, "https"); err != nil {
			return fmt.Errorf("invalid self-hosted API URL: %w", err)
		}
	default:
		return fmt.Errorf("unsupported Atomic Memory backend %q", c.Backend)
	}
	return nil
}

func (c Config) Marshal() ([]byte, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal Atomic Memory config: %w", err)
	}
	return append(data, '\n'), nil
}

func ParseConfig(data []byte) (Config, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var config Config
	if err := decoder.Decode(&config); err != nil {
		return Config{}, fmt.Errorf("parse Atomic Memory config: %w", err)
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return Config{}, errors.New("parse Atomic Memory config: trailing data")
	}
	if err := config.Validate(); err != nil {
		return Config{}, err
	}
	return config, nil
}

func isLoopbackHTTP(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme != "http" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return false
	}
	host := parsed.Hostname()
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func validateURL(rawURL, requiredScheme string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	if parsed.Scheme != requiredScheme || parsed.Hostname() == "" {
		return fmt.Errorf("URL must use %s and include a host", requiredScheme)
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return errors.New("URL must not include credentials, a query, or a fragment")
	}
	if strings.TrimSpace(rawURL) != rawURL {
		return errors.New("URL must not contain surrounding whitespace")
	}
	return nil
}
