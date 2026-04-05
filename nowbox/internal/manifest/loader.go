package manifest

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

//go:embed packages/hosts/*/host.toml
var embeddedHosts embed.FS

//go:embed packages/agents/*/agent.toml
var embeddedAgents embed.FS

var cacheDir = filepath.Join(os.Getenv("HOME"), ".nowbox")

// LoadHost resolves a host by name or path following the manifest fetch chain.
func LoadHost(nameOrPath string) (*HostManifest, error) {
	data, err := resolve(nameOrPath, "hosts", "host.toml")
	if err != nil {
		return nil, err
	}

	var m HostManifest
	if err := toml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing host manifest: %w", err)
	}

	// Validate adapter is a known type
	switch m.Adapter {
	case "websocket_exec":
		// ok
	default:
		return nil, fmt.Errorf("unsupported adapter: %s", m.Adapter)
	}

	return &m, nil
}

// LoadAgent resolves an agent by name or path following the manifest fetch chain.
func LoadAgent(nameOrPath string) (*AgentManifest, error) {
	data, err := resolve(nameOrPath, "agents", "agent.toml")
	if err != nil {
		return nil, err
	}

	var m AgentManifest
	if err := toml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing agent manifest: %w", err)
	}

	return &m, nil
}

// resolve follows the manifest fetch chain from the spec:
// 1. local file path → use it
// 2. cached locally → use ~/.nowbox/<kind>/<name>.toml
// 3. bundled in binary → use embedded
// 4. URL → fetch, verify, cache (not implemented in v0)
// 5. unknown → error
func resolve(nameOrPath string, kind string, filename string) ([]byte, error) {
	// 1. Local file path
	if strings.Contains(nameOrPath, "/") || strings.Contains(nameOrPath, ".") {
		data, err := os.ReadFile(nameOrPath)
		if err != nil {
			return nil, fmt.Errorf("reading local manifest %s: %w", nameOrPath, err)
		}
		return data, nil
	}

	// 2. Cached locally
	cachePath := filepath.Join(cacheDir, kind, nameOrPath+".toml")
	if data, err := os.ReadFile(cachePath); err == nil {
		return data, nil
	}

	// 3. Bundled in binary
	var fs embed.FS
	if kind == "hosts" {
		fs = embeddedHosts
	} else {
		fs = embeddedAgents
	}

	embeddedPath := fmt.Sprintf("packages/%s/%s/%s", kind, nameOrPath, filename)
	if data, err := fs.ReadFile(embeddedPath); err == nil {
		return data, nil
	}

	// 4. URL fetch — not implemented in v0

	// 5. Unknown
	return nil, fmt.Errorf("unknown %s: %q", kind[:len(kind)-1], nameOrPath)
}

// ListHosts returns all known host names from the bundled index.
func ListHosts() []HostManifest {
	return listManifests[HostManifest](embeddedHosts, "hosts", "host.toml")
}

// ListAgents returns all known agent names from the bundled index.
func ListAgents() []AgentManifest {
	return listManifests[AgentManifest](embeddedAgents, "agents", "agent.toml")
}

func listManifests[T any](fs embed.FS, kind string, filename string) []T {
	var results []T

	dirPath := fmt.Sprintf("packages/%s", kind)
	entries, err := fs.ReadDir(dirPath)
	if err != nil {
		return results
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		path := fmt.Sprintf("packages/%s/%s/%s", kind, entry.Name(), filename)
		data, err := fs.ReadFile(path)
		if err != nil {
			continue
		}

		var m T
		if err := toml.Unmarshal(data, &m); err != nil {
			continue
		}
		results = append(results, m)
	}

	return results
}

// Expand replaces ${VAR} references in a string with values from the vars map.
func Expand(s string, vars map[string]string) string {
	for k, v := range vars {
		s = strings.ReplaceAll(s, "${"+k+"}", v)
	}
	return s
}

// ExpandMap replaces ${VAR} references in all values of a map.
func ExpandMap(m map[string]string, vars map[string]string) map[string]string {
	result := make(map[string]string, len(m))
	for k, v := range m {
		result[k] = Expand(v, vars)
	}
	return result
}
