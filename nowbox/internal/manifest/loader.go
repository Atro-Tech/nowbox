package manifest

import (
	"crypto/ed25519"
	"crypto/x509"
	_ "embed"
	"encoding/pem"
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

//go:embed signing.pub
var signingPubKeyPEM []byte

var signingPubKey ed25519.PublicKey

func init() {
	block, _ := pem.Decode(signingPubKeyPEM)
	if block == nil {
		panic("invalid embedded signing public key")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		panic("invalid embedded signing public key: " + err.Error())
	}
	signingPubKey = pub.(ed25519.PublicKey)
}

// Bundled index — names and descriptions only.
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
// 2. cached locally → use it (was verified on first fetch)
// 3. known name in index → fetch from GitHub + signature, verify, cache
// 4. URL → fetch + signature, verify, cache
// 5. unknown → error
func resolve(nameOrPath string, kind string, filename string) ([]byte, error) {
	// 1. Local file path — unsigned developer mode
	if strings.Contains(nameOrPath, "/") || strings.Contains(nameOrPath, ".toml") {
		data, err := os.ReadFile(nameOrPath)
		if err != nil {
			return nil, fmt.Errorf("reading local manifest %s: %w", nameOrPath, err)
		}
		return data, nil
	}

	// 2. Cached locally (verified on first fetch)
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
		manifestURL := fmt.Sprintf("%s/%s/%s/%s", repoBase, kind, nameOrPath, filename)
		sigURL := manifestURL + ".sig"
		data, err := fetchAndVerify(nameOrPath, manifestURL, sigURL, cachePath)
		if err != nil {
			return nil, fmt.Errorf("fetching %s manifest for %q: %w", kind[:len(kind)-1], nameOrPath, err)
		}
		return data, nil
	}

	// 4. URL → fetch + verify
	if strings.HasPrefix(nameOrPath, "http://") || strings.HasPrefix(nameOrPath, "https://") {
		sigURL := nameOrPath + ".sig"
		data, err := fetchAndVerify(nameOrPath, nameOrPath, sigURL, cachePath)
		if err != nil {
			return nil, fmt.Errorf("fetching manifest from %s: %w", nameOrPath, err)
		}
		return data, nil
	}

	// 5. Unknown
	return nil, fmt.Errorf("unknown %s: %q", kind[:len(kind)-1], nameOrPath)
}

func fetchAndVerify(name string, manifestURL string, sigURL string, cachePath string) ([]byte, error) {
	fmt.Fprintf(os.Stderr, "  loading %s...\n", name)

	// Fetch manifest
	resp, err := http.Get(manifestURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, manifestURL)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Fetch signature
	sigResp, err := http.Get(sigURL)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch signature: %w", err)
	}
	defer sigResp.Body.Close()
	if sigResp.StatusCode != 200 {
		return nil, fmt.Errorf("manifest signature not found at %s (HTTP %d) — refusing unsigned manifest", sigURL, sigResp.StatusCode)
	}
	sig, err := io.ReadAll(sigResp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading signature: %w", err)
	}

	// Verify signature
	if !ed25519.Verify(signingPubKey, data, sig) {
		return nil, fmt.Errorf("manifest signature verification FAILED — manifest may be tampered")
	}
	// Cache (only after verification)
	dir := filepath.Dir(cachePath)
	os.MkdirAll(dir, 0700)
	os.WriteFile(cachePath, data, 0600)

	return data, nil
}

func ListHosts() []IndexEntry  { return HostIndex }
func ListAgents() []IndexEntry { return AgentIndex }

func Expand(s string, vars map[string]string) string {
	for k, v := range vars {
		s = strings.ReplaceAll(s, "${"+k+"}", v)
	}
	return s
}

func ExpandMap(m map[string]string, vars map[string]string) map[string]string {
	result := make(map[string]string, len(m))
	for k, v := range m {
		result[k] = Expand(v, vars)
	}
	return result
}
