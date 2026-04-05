package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/nowbox/nowbox/internal/manifest"
	"github.com/nowbox/nowbox/internal/session"
	"github.com/nowbox/nowbox/internal/terminal"
	"github.com/nowbox/nowbox/internal/token"
	"github.com/nowbox/nowbox/internal/webui"
	"golang.org/x/term"
)

// ttyReader reads from /dev/tty so prompts work even when stdin is piped (curl | sh).
var ttyReader *bufio.Reader

func initTTY() {
	f, err := os.Open("/dev/tty")
	if err != nil {
		fmt.Fprintf(os.Stderr, "nowbox: error: cannot open terminal for input\n")
		os.Exit(1)
	}
	ttyReader = bufio.NewReader(f)
}

func prompt(label string) string {
	fmt.Fprintf(os.Stderr, "%s", label)
	line, _ := ttyReader.ReadString('\n')
	return strings.TrimSpace(line)
}

func promptHidden(label string) string {
	fmt.Fprintf(os.Stderr, "%s", label)
	// Read from /dev/tty as a line — strip any non-printable chars
	// that terminal emulators inject on focus/click events
	line, _ := ttyReader.ReadString('\n')
	clean := strings.Map(func(r rune) rune {
		if r >= 32 && r < 127 {
			return r
		}
		return -1
	}, line)
	fmt.Fprintf(os.Stderr, "\n")
	return strings.TrimSpace(clean)
}

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

	// If a .now token is present, decrypt it and inject vars + args
	if tok := os.Getenv("NOWBOX_TOKEN"); tok != "" {
		payload, err := token.Open(tok)
		if err != nil {
			fmt.Fprintf(os.Stderr, "nowbox: %v\n", err)
			os.Exit(1)
		}
		if payload.Host != "" && hostName == "" {
			hostName = payload.Host
		}
		if payload.Agent != "" && agentName == "" {
			agentName = payload.Agent
		}
		for k, v := range payload.Vars {
			os.Setenv(k, v)
		}
	}

	initTTY()

	// Check for orphan from previous crash
	if orphan := session.CheckOrphan(); orphan != nil {
		fmt.Fprintf(os.Stderr, "nowbox: orphan sandbox found: %s (%s on %s)\n",
			orphan.SessionName, orphan.InstanceID, orphan.Provider)
		answer := prompt("nowbox: destroy it? [y/N] ")
		if strings.ToLower(answer) == "y" {
			host, err := manifest.LoadHost(orphan.Provider)
			if err != nil {
				fmt.Fprintf(os.Stderr, "nowbox: could not load host manifest for cleanup: %v\n", err)
			} else {
				vars, err := collectHostVars(host)
				if err != nil {
					fmt.Fprintf(os.Stderr, "nowbox: could not collect host credentials for cleanup: %v\n", err)
				} else if err := session.DestroyOrphan(orphan.Provider, orphan.InstanceID, vars); err != nil {
					fmt.Fprintf(os.Stderr, "nowbox: orphan cleanup failed: %v\n", err)
				} else {
					fmt.Fprintf(os.Stderr, "nowbox: destroyed orphan sandbox\n")
				}
			}
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
		hostName = prompt("host: ")
		hostName = resolveChoice(hostName, indexNames(hosts))
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
		agentName = prompt("agent: ")
		agentName = resolveChoice(agentName, indexNames(agents))
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
	vars, err := collectHostVars(host)
	if err != nil {
		fmt.Fprintf(os.Stderr, "nowbox: %v\n", err)
		os.Exit(1)
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
			if _, err := sess.Stream.Write([]byte(cmd + "\n")); err != nil {
				fmt.Fprintf(os.Stderr, "nowbox: setup failed: %v\n", err)
				sess.Destroy()
				os.Exit(1)
			}
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

func collectHostVars(host *manifest.HostManifest) (map[string]string, error) {
	vars := make(map[string]string, len(host.Keys.Required))

	for _, keyName := range host.Keys.Required {
		if value := os.Getenv(keyName); value != "" {
			vars[keyName] = value
			continue
		}

		p := host.Keys.Prompt
		if p == "" {
			p = keyName
		}

		value := promptHidden(fmt.Sprintf("  %s: ", p))
		if value == "" {
			return nil, fmt.Errorf("%s required", keyName)
		}
		vars[keyName] = value
	}

	return vars, nil
}

func resolveChoice(input string, options []string) string {
	choice := strings.TrimSpace(input)
	if idx, err := strconv.Atoi(choice); err == nil && idx >= 1 && idx <= len(options) {
		return options[idx-1]
	}
	return choice
}

func indexNames(entries []manifest.IndexEntry) []string {
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name)
	}
	return names
}

