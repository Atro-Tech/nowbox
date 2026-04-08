package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/nowbox/nowbox/internal/appui"
	"github.com/nowbox/nowbox/internal/manifest"
	"github.com/nowbox/nowbox/internal/mcpserver"
	"github.com/nowbox/nowbox/internal/names"
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
	f, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return ""
	}
	defer f.Close()

	// Disable echo so the key isn't shown
	oldState, err := term.MakeRaw(int(f.Fd()))
	if err != nil {
		// Fall back to visible read
		line, _ := ttyReader.ReadString('\n')
		return strings.TrimSpace(line)
	}

	// Read until enter, stripping non-printable chars
	var buf []byte
	b := make([]byte, 1)
	for {
		n, err := f.Read(b)
		if err != nil || n == 0 {
			break
		}
		if b[0] == '\r' || b[0] == '\n' {
			break
		}
		if b[0] >= 32 && b[0] < 127 {
			buf = append(buf, b[0])
		}
	}

	// Restore terminal before printing newline
	term.Restore(int(f.Fd()), oldState)
	fmt.Fprintf(os.Stderr, "\r\n")
	return string(buf)
}

func main() {
	// CLI flags per spec: --host / -h, --agent / -a
	var hostName, agentName, clientMode string
	flag.StringVar(&hostName, "host", "", "host provider")
	flag.StringVar(&hostName, "h", "", "host provider (short)")
	flag.StringVar(&agentName, "agent", "", "agent")
	flag.StringVar(&agentName, "a", "", "agent (short)")
	flag.StringVar(&clientMode, "client", "cli", "client mode: cli, browser, app, mcp")
	flag.StringVar(&clientMode, "c", "cli", "client mode (short)")
	flag.Parse()

	args := flag.Args()
	createMode := false
	var existingInstanceID string
	var nowFilePath string
	var sessionName string
	fromNowFile := false

	// Check for "create" subcommand: nowbox create sprites claude
	// Re-parse flags after "create" since Go's flag stops at the first non-flag arg
	if len(args) > 0 && args[0] == "create" {
		createMode = true
		sub := flag.NewFlagSet("create", flag.ExitOnError)
		sub.StringVar(&hostName, "host", hostName, "host provider")
		sub.StringVar(&hostName, "h", hostName, "host provider (short)")
		sub.StringVar(&agentName, "agent", agentName, "agent")
		sub.StringVar(&agentName, "a", agentName, "agent (short)")
		sub.StringVar(&clientMode, "client", clientMode, "client mode")
		sub.StringVar(&clientMode, "c", clientMode, "client mode (short)")
		sub.StringVar(&sessionName, "name", "", "session name")
		sub.StringVar(&sessionName, "n", "", "session name (short)")
		sub.Parse(args[1:])
		args = sub.Args()
	}

	// Check if first arg is a .now file or NOWBOX_TOKEN env var
	if tok := os.Getenv("NOWBOX_TOKEN"); tok != "" {
		existingInstanceID = loadToken(tok, &hostName, &agentName)
		nowFilePath = os.Getenv("NOWBOX_FILE")
		fromNowFile = true
	} else if len(args) > 0 && strings.HasSuffix(args[0], ".now") {
		data, err := os.ReadFile(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "nowbox: %v\n", err)
			os.Exit(1)
		}
		existingInstanceID = loadToken(strings.TrimSpace(string(data)), &hostName, &agentName)
		nowFilePath = args[0]
		fromNowFile = true
		args = args[1:]
	}

	if hostName == "" && len(args) > 0 {
		hostName = args[0]
	}
	if agentName == "" && len(args) > 1 {
		agentName = args[1]
	}
	if len(args) > 2 {
		clientMode = args[2]
	}

	initTTY()

	// Check for orphan from previous crash — skip if we're reconnecting to it
	if orphan := session.CheckOrphan(); orphan != nil && orphan.InstanceID != existingInstanceID {
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

	if hostName == "" {
		hosts := manifest.ListHosts()
		if len(hosts) == 0 {
			fmt.Fprintf(os.Stderr, "nowbox: no hosts available\n")
			os.Exit(1)
		}
		hostName = pick("host", hosts)
	}

	if agentName == "" {
		agents := manifest.ListAgents()
		if len(agents) == 0 {
			fmt.Fprintf(os.Stderr, "nowbox: no agents available\n")
			os.Exit(1)
		}
		agentName = pick("agent", agents)
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

	// Create mode — provision sandbox, write .now file, don't connect
	if createMode {
		createNowFile(host, agent, vars, clientMode, sessionName)
		return
	}

	// Display name for UI (toolbar, mDNS) — from .now filename or random
	displayName := sessionName // from -name flag
	if displayName == "" && nowFilePath != "" {
		displayName = strings.TrimSuffix(filepath.Base(nowFilePath), ".now")
	}
	if displayName == "" {
		displayName = names.Generate()
	}

	// Connect to existing sandbox, or create a new one
	var sess *session.Session
	if existingInstanceID != "" {
		sess, err = session.Reconnect(host, agent, existingInstanceID, vars)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  reconnect failed, creating new sandbox...\n")
			sess, err = session.New(host, agent, vars)
		}
	} else {
		sess, err = session.New(host, agent, vars)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "nowbox: %v\n", err)
		os.Exit(1)
	}

	// Connect via selected client mode
	hostAgent := fmt.Sprintf("%s/%s", host.Name, agent.Name)

	allModes := []string{"cli", "browser", "app", "mcp"}
	saveFunc := func() error {
		// Skip save if the .now file already has this instance ID
		if existingInstanceID == sess.InstanceID && nowFilePath != "" {
			return nil
		}

		keyVars := make(map[string]string)
		for k, v := range sess.Vars {
			if k == "SESSION_NAME" || k == "INSTANCE_ID" {
				continue
			}
			keyVars[k] = v
		}
		sealed, err := token.Seal(&token.Payload{
			Host:       host.Name,
			Agent:      agent.Name,
			Vars:       keyVars,
			InstanceID: sess.InstanceID,
		})
		if err != nil {
			return err
		}
		filename := nowFilePath
		if filename == "" {
			filename = displayName + ".now"
		}
		cf := ""
		if clientMode != "" && clientMode != "cli" {
			cf = fmt.Sprintf(" -client %s", clientMode)
		}
		script := fmt.Sprintf(`#!/bin/sh
# nowbox session — %s + %s
export NOWBOX_TOKEN="%s"
export NOWBOX_FILE="$0"
curl -fsSL nowbox.lol | sh -s --%s "$@"
`, host.Name, agent.Name, sealed, cf)
		return os.WriteFile(filename, []byte(script), 0755)
	}
	savePath := nowFilePath
	if savePath == "" {
		savePath = displayName + ".now"
	}

	// Set up signal handler — smart default based on session origin
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		if fromNowFile {
			saveFunc()
			session.ClearRecovery()
			fmt.Fprintf(os.Stderr, "\nnowbox: disconnected — sandbox still running\n")
		} else {
			fmt.Fprintf(os.Stderr, "\nnowbox: interrupted, destroying %s...\n", displayName)
			sess.Destroy()
		}
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

	for {
		switch clientMode {
		case "cli":
			otherModes := modesExcept(allModes, "cli")
			var nextMode string
			var exitAction terminal.ExitAction
			nextMode, exitAction, err = terminal.Proxy(sess.Stream, displayName, hostAgent, &terminal.ProxyOptions{
				Modes:        otherModes,
				SaveFunc:     saveFunc,
				IsPersistent: fromNowFile,
			})
			if nextMode != "" {
				clientMode = nextMode
				continue
			}
			if exitAction == terminal.ExitKeep {
				if saveErr := saveFunc(); saveErr != nil {
					fmt.Fprintf(os.Stderr, "\n  warning: could not save: %v\n", saveErr)
				}
				session.ClearRecovery()
				fmt.Fprintf(os.Stderr, "\nnowbox: disconnected — sandbox still running\n")
				if existingInstanceID != sess.InstanceID {
					fmt.Fprintf(os.Stderr, "  saved: %s\n", savePath)
				}
				return
			}
		case "browser":
			err = webui.Serve(sess.Stream, displayName, hostAgent, &webui.SessionInfo{
				HostName:   host.Name,
				AgentName:  agent.Name,
				Vars:       sess.Vars,
				InstanceID: sess.InstanceID,
			})
		case "app":
			err = appui.Serve(sess.Stream, displayName, hostAgent, &appui.SessionInfo{
				HostName:   host.Name,
				AgentName:  agent.Name,
				Vars:       sess.Vars,
				InstanceID: sess.InstanceID,
			})
		case "mcp":
			err = mcpserver.Serve(host, sess.Stream, sess.InstanceID, sess.Name, agent.Name, sess.Vars)
		default:
			fmt.Fprintf(os.Stderr, "nowbox: unknown client mode: %s\n", clientMode)
			sess.Destroy()
			os.Exit(1)
		}
		break
	}

	// Teardown — browser/app/mcp use smart default
	if fromNowFile {
		if saveErr := saveFunc(); saveErr != nil {
			fmt.Fprintf(os.Stderr, "\n  warning: could not save: %v\n", saveErr)
		}
		session.ClearRecovery()
		fmt.Fprintf(os.Stderr, "\nnowbox: disconnected — sandbox still running\n")
		if existingInstanceID != sess.InstanceID {
			fmt.Fprintf(os.Stderr, "  saved: %s\n", savePath)
		}
	} else {
		fmt.Fprintf(os.Stderr, "\nnowbox: disconnected, destroying %s...\n", displayName)
		sess.Destroy()
	}

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

func createNowFile(host *manifest.HostManifest, agent *manifest.AgentManifest, vars map[string]string, clientMode string, displayName string) {
	// Filter vars to only include API keys
	keyVars := make(map[string]string)
	for k, v := range vars {
		if k == "SESSION_NAME" || k == "INSTANCE_ID" {
			continue
		}
		keyVars[k] = v
	}

	sealed, err := token.Seal(&token.Payload{
		Host:  host.Name,
		Agent: agent.Name,
		Vars:  keyVars,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "nowbox: failed to create token: %v\n", err)
		os.Exit(1)
	}

	if displayName == "" {
		displayName = names.Generate()
	}
	filename := displayName + ".now"

	clientFlag := ""
	if clientMode != "" && clientMode != "cli" {
		clientFlag = fmt.Sprintf(" -client %s", clientMode)
	}

	script := fmt.Sprintf(`#!/bin/sh
# nowbox session — %s + %s
export NOWBOX_TOKEN="%s"
export NOWBOX_FILE="$0"
curl -fsSL nowbox.lol | sh -s --%s "$@"
`, host.Name, agent.Name, sealed, clientFlag)

	if err := os.WriteFile(filename, []byte(script), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "nowbox: failed to write %s: %v\n", filename, err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "  created: %s\n", filename)
	fmt.Fprintf(os.Stderr, "  run:     sh %s\n", filename)
}

func loadToken(tok string, hostName *string, agentName *string) string {
	p, err := token.Open(tok)
	if err != nil {
		fmt.Fprintf(os.Stderr, "nowbox: invalid token: %v\n", err)
		os.Exit(1)
	}
	if *hostName == "" {
		*hostName = p.Host
	}
	if *agentName == "" {
		*agentName = p.Agent
	}
	// Inject vars as env vars so collectHostVars picks them up
	for k, v := range p.Vars {
		os.Setenv(k, v)
	}
	return p.InstanceID
}

// pick renders an interactive arrow-key picker on /dev/tty.
// Returns the selected item name.
func pick(label string, items []manifest.IndexEntry) string {
	ttyFile, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		// Fall back to numbered list
		fmt.Fprintf(os.Stderr, "%s:\n", label)
		for i, item := range items {
			fmt.Fprintf(os.Stderr, "  [%d] %s — %s\n", i+1, item.Name, item.Description)
		}
		choice := prompt(label + ": ")
		return resolveChoice(choice, indexNames(items))
	}

	oldState, err := term.MakeRaw(int(ttyFile.Fd()))
	if err != nil {
		ttyFile.Close()
		choice := prompt(label + ": ")
		return resolveChoice(choice, indexNames(items))
	}

	selected := 0
	buf := make([]byte, 3)

	render := func() {
		// Move to start and clear
		fmt.Fprintf(os.Stderr, "\r\033[K  %s\r\n", label)
		for i, item := range items {
			if i == selected {
				fmt.Fprintf(os.Stderr, "\033[K  \033[1m❯ %s\033[0m  %s\r\n", item.Name, item.Description)
			} else {
				fmt.Fprintf(os.Stderr, "\033[K    %s  \033[90m%s\033[0m\r\n", item.Name, item.Description)
			}
		}
		// Move cursor back up
		for range items {
			fmt.Fprintf(os.Stderr, "\033[A")
		}
		fmt.Fprintf(os.Stderr, "\033[A")
	}

	render()

	for {
		n, err := ttyFile.Read(buf)
		if err != nil || n == 0 {
			break
		}

		if n == 1 {
			switch buf[0] {
			case '\r', '\n': // Enter
				term.Restore(int(ttyFile.Fd()), oldState)
				ttyFile.Close()
				// Clear the picker
				fmt.Fprintf(os.Stderr, "\r\033[K")
				for range items {
					fmt.Fprintf(os.Stderr, "\033[B\033[K")
				}
				// Move back up and show selection
				for range items {
					fmt.Fprintf(os.Stderr, "\033[A")
				}
				fmt.Fprintf(os.Stderr, "\r\033[K  %s: %s\r\n", label, items[selected].Name)
				return items[selected].Name
			case 'j', 'J': // vim down
				if selected < len(items)-1 {
					selected++
					render()
				}
			case 'k', 'K': // vim up
				if selected > 0 {
					selected--
					render()
				}
			case 'q', 3: // q or ctrl-c
				term.Restore(int(ttyFile.Fd()), oldState)
				ttyFile.Close()
				fmt.Fprintf(os.Stderr, "\r\n")
				os.Exit(0)
			}
		} else if n == 3 && buf[0] == '\033' && buf[1] == '[' {
			switch buf[2] {
			case 'A': // Up arrow
				if selected > 0 {
					selected--
					render()
				}
			case 'B': // Down arrow
				if selected < len(items)-1 {
					selected++
					render()
				}
			}
		}
	}

	term.Restore(int(ttyFile.Fd()), oldState)
	ttyFile.Close()
	return items[selected].Name
}

func indexNames(entries []manifest.IndexEntry) []string {
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name)
	}
	return names
}

func modesExcept(all []string, current string) []string {
	var out []string
	for _, m := range all {
		if m != current {
			out = append(out, m)
		}
	}
	return out
}

