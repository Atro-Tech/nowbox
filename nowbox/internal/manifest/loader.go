package manifest

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

const repoBase = "https://raw.githubusercontent.com/Atro-Tech/nowbox/main/nowbox/internal/manifest/packages"

var cacheDir = filepath.Join(os.Getenv("HOME"), ".nowbox")

// Bundled index — just names and descriptions, compiled into the binary.
// The actual manifests are fetched from GitHub at runtime.
type IndexEntry struct {
	Name        string
	Description string
}

var HostIndex = []IndexEntry{
	{Name: "sprites", Description: "Fly.io sandbox (sprites.dev)"},
	{Name: "modal", Description: "Modal container"},
	{Name: "e2b", Description: "E2B microVM"},
	{Name: "daytona", Description: "Daytona sandbox"},
	{Name: "fly", Description: "Fly.io machine"},
	{Name: "cloudflare", Description: "Cloudflare container"},
	{Name: "docker", Description: "Local Docker container"},
	{Name: "aws", Description: "AWS ECS Fargate"},
	{Name: "blaxel", Description: "Blaxel sandbox"},
	{Name: "runloop", Description: "Runloop devbox"},
	{Name: "vercel", Description: "Vercel sandbox"},
	{Name: "codesandbox", Description: "CodeSandbox devbox"},
	{Name: "podman", Description: "Local Podman container"},
	{Name: "apple", Description: "Apple Virtualization (macOS)"},
}

var AgentIndex = []IndexEntry{
	{Name: "claude", Description: "Claude Code (Anthropic)"},
	{Name: "codex", Description: "Codex (OpenAI)"},
	{Name: "aider", Description: "Aider (AI pair programmer)"},
	{Name: "openclaw", Description: "OpenClaw (AI assistant)"},
	{Name: "hermes", Description: "Hermes Agent (Nous Research)"},
	{Name: "opencode", Description: "OpenCode (terminal coding agent)"},
	{Name: "goose", Description: "Goose (Block)"},
	{Name: "cline", Description: "Cline (terminal AI agent)"},
}

// LoadHost resolves a host by name or path.
func LoadHost(nameOrPath string) (*HostManifest, error) {
	data, err := resolve(nameOrPath, "hosts", "host.toml")
	if err != nil {
		return nil, err
	}

	var m HostManifest
	if err := toml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing host manifest: %w", err)
	}

	return &m, nil
}

// LoadAgent resolves an agent by name or path.
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

// resolve follows the manifest fetch chain:
// 1. local file path → use it (unsigned, developer mode)
// 2. cached locally → use ~/.nowbox/<kind>/<name>.toml
// 3. known name in index → fetch from GitHub, cache it
// 4. URL → fetch it, cache it
// 5. unknown → error
func resolve(nameOrPath string, kind string, filename string) ([]byte, error) {
	// 1. Local file path
	if strings.Contains(nameOrPath, "/") || strings.Contains(nameOrPath, ".toml") {
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

	// 3. Known name in index → fetch from GitHub
	var index []IndexEntry
	if kind == "hosts" {
		index = HostIndex
	} else {
		index = AgentIndex
	}

	known := false
	for _, entry := range index {
		if entry.Name == nameOrPath {
			known = true
			break
		}
	}

	if known {
		url := fmt.Sprintf("%s/%s/%s/%s", repoBase, kind, nameOrPath, filename)
		data, err := fetchAndCache(url, cachePath)
		if err != nil {
			return nil, fmt.Errorf("fetching %s manifest for %q: %w", kind[:len(kind)-1], nameOrPath, err)
		}
		return data, nil
	}

	// 4. URL
	if strings.HasPrefix(nameOrPath, "http://") || strings.HasPrefix(nameOrPath, "https://") {
		data, err := fetchAndCache(nameOrPath, cachePath)
		if err != nil {
			return nil, fmt.Errorf("fetching manifest from %s: %w", nameOrPath, err)
		}
		return data, nil
	}

	// 5. Unknown
	return nil, fmt.Errorf("unknown %s: %q", kind[:len(kind)-1], nameOrPath)
}

func fetchAndCache(url string, cachePath string) ([]byte, error) {
	fmt.Fprintf(os.Stderr, "  fetching manifest...\n")

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Cache it
	dir := filepath.Dir(cachePath)
	os.MkdirAll(dir, 0700)
	os.WriteFile(cachePath, data, 0600)

	return data, nil
}

// ListHosts returns the bundled host index.
func ListHosts() []IndexEntry {
	return HostIndex
}

// ListAgents returns the bundled agent index.
func ListAgents() []IndexEntry {
	return AgentIndex
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
