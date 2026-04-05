package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/nowbox/nowbox/internal/manifest"
	"github.com/nowbox/nowbox/internal/session"
	"github.com/nowbox/nowbox/internal/terminal"
	"github.com/nowbox/nowbox/internal/webui"
	"golang.org/x/term"
)

func main() {
	// CLI flags per spec: --host / -h, --agent / -a
	var hostName, agentName, clientMode string
	flag.StringVar(&hostName, "host", "", "host provider")
	flag.StringVar(&hostName, "h", "", "host provider (short)")
	flag.StringVar(&agentName, "agent", "", "agent")
	flag.StringVar(&agentName, "a", "", "agent (short)")
	flag.StringVar(&clientMode, "client", "cli", "client mode: cli, web, mcp")
	flag.StringVar(&clientMode, "c", "cli", "client mode (short)")
	flag.Parse()

	// Positional args: nowbox sprites claude web
	args := flag.Args()
	if hostName == "" && len(args) > 0 {
		hostName = args[0]
	}
	if agentName == "" && len(args) > 1 {
		agentName = args[1]
	}
	if len(args) > 2 {
		clientMode = args[2]
	}

	// Check for orphan from previous crash
	if orphan := session.CheckOrphan(); orphan != nil {
		fmt.Fprintf(os.Stderr, "nowbox: orphan sandbox found: %s (%s on %s)\n",
			orphan.SessionName, orphan.InstanceID, orphan.Provider)
		fmt.Fprintf(os.Stderr, "nowbox: destroy it? [y/N] ")
		var answer string
		fmt.Scanln(&answer)
		if strings.ToLower(answer) == "y" {
			// Load the host manifest to get the adapter for destroy
			host, err := manifest.LoadHost(orphan.Provider)
			if err == nil {
				s := &session.Session{
					Name:       orphan.SessionName,
					InstanceID: orphan.InstanceID,
					Host:       host,
				}
				// TODO: needs adapter + vars to destroy properly
				_ = s
			}
			session.ClearRecovery()
			fmt.Fprintf(os.Stderr, "nowbox: cleared orphan record\n")
		}
	}

	// If no host specified, prompt (interactive picker is later — for now just ask)
	if hostName == "" {
		hosts := manifest.ListHosts()
		if len(hosts) == 0 {
			fmt.Fprintf(os.Stderr, "nowbox: no hosts available\n")
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "nowbox: available hosts:\n")
		for i, h := range hosts {
			fmt.Fprintf(os.Stderr, "  [%d] %s — %s\n", i+1, h.Name, h.Description)
		}
		fmt.Fprintf(os.Stderr, "host: ")
		fmt.Scanln(&hostName)
	}

	// If no agent specified, prompt
	if agentName == "" {
		agents := manifest.ListAgents()
		if len(agents) == 0 {
			fmt.Fprintf(os.Stderr, "nowbox: no agents available\n")
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "nowbox: available agents:\n")
		for i, a := range agents {
			fmt.Fprintf(os.Stderr, "  [%d] %s — %s\n", i+1, a.Name, a.Description)
		}
		fmt.Fprintf(os.Stderr, "agent: ")
		fmt.Scanln(&agentName)
	}

	// Load manifests
	host, err := manifest.LoadHost(hostName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "nowbox: %v\n", err)
		os.Exit(1)
	}

	agent, err := manifest.LoadAgent(agentName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "nowbox: %v\n", err)
		os.Exit(1)
	}

	// Collect host keys — interactive prompts with hidden input
	vars := make(map[string]string)
	for _, keyName := range host.Keys.Required {
		prompt := host.Keys.Prompt
		if prompt == "" {
			prompt = keyName
		}
		fmt.Fprintf(os.Stderr, "  %s: ", prompt)

		fd := int(os.Stdin.Fd())
		if !term.IsTerminal(fd) {
			fmt.Fprintf(os.Stderr, "\nnowbox: error: %s required (run interactively)\n", keyName)
			os.Exit(1)
		}

		keyBytes, err := term.ReadPassword(fd)
		fmt.Fprintf(os.Stderr, "\n")
		if err != nil {
			fmt.Fprintf(os.Stderr, "nowbox: error reading key: %v\n", err)
			os.Exit(1)
		}
		vars[keyName] = string(keyBytes)
	}

	// v0: agent keys are not collected — agent handles its own auth

	// Create session (provisions sandbox + connects)
	sess, err := session.New(host, agent, vars)
	if err != nil {
		fmt.Fprintf(os.Stderr, "nowbox: %v\n", err)
		os.Exit(1)
	}

	// Set up signal handler for cleanup
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Fprintf(os.Stderr, "\nnowbox: interrupted, destroying %s...\n", sess.Name)
		sess.Destroy()
		os.Exit(130)
	}()

	// Send agent setup commands
	if len(agent.Setup.Commands) > 0 {
		fmt.Fprintf(os.Stderr, "  setting up %s...\n", agent.Name)
		for _, cmd := range agent.Setup.Commands {
			sess.Stream.Write([]byte(cmd + "\n"))
		}
	}

	fmt.Fprintf(os.Stderr, "  ready\n")

	// Connect via selected client mode
	hostAgent := fmt.Sprintf("%s/%s", host.Name, agent.Name)

	switch clientMode {
	case "cli":
		err = terminal.Proxy(sess.Stream, sess.Name, hostAgent)
	case "web":
		err = webui.Serve(sess.Stream, sess.Name, hostAgent)
	default:
		fmt.Fprintf(os.Stderr, "nowbox: unknown client mode: %s\n", clientMode)
		sess.Destroy()
		os.Exit(1)
	}

	// Teardown
	fmt.Fprintf(os.Stderr, "\nnowbox: disconnected, destroying %s...\n", sess.Name)
	sess.Destroy()

	if err != nil {
		fmt.Fprintf(os.Stderr, "nowbox: %v\n", err)
		os.Exit(1)
	}
}
