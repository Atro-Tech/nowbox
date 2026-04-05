package manifest

// HostManifest is the fixed schema for host packages.
type HostManifest struct {
	Name        string            `toml:"name"`
	Description string            `toml:"description"`
	Adapter     string            `toml:"adapter"`
	Keys        HostKeys          `toml:"keys"`
	Create      HostCreate        `toml:"create"`
	Connect     HostConnect       `toml:"connect"`
	Destroy     HostDestroy       `toml:"destroy"`
	Lifecycle   HostLifecycle     `toml:"lifecycle"`
}

type HostKeys struct {
	Required []string `toml:"required"`
	Prompt   string   `toml:"prompt"`
}

type HostCreate struct {
	Method  string            `toml:"method"`
	URL     string            `toml:"url"`
	Headers map[string]string `toml:"headers"`
	Body    string            `toml:"body"`
	ParseID string            `toml:"parse_id"`
}

type HostConnect struct {
	URL     string            `toml:"url"`
	Headers map[string]string `toml:"headers"`
	Query   map[string]string `toml:"query"`
}

type HostDestroy struct {
	Method  string            `toml:"method"`
	URL     string            `toml:"url"`
	Headers map[string]string `toml:"headers"`
}

type HostLifecycle struct {
	IdleTimeout       string `toml:"idle_timeout"`
	HeartbeatInterval string `toml:"heartbeat_interval"`
}

// AgentManifest is the fixed schema for agent packages.
type AgentManifest struct {
	Name        string     `toml:"name"`
	Description string     `toml:"description"`
	Setup       AgentSetup `toml:"setup"`
	Keys        AgentKeys  `toml:"keys"`
}

type AgentSetup struct {
	Commands []string `toml:"commands"`
}

type AgentKeys struct {
	Optional []string `toml:"optional"`
	SelfAuth bool     `toml:"self_auth"`
}
