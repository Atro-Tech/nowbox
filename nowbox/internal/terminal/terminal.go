package terminal

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/nowbox/nowbox/internal/adapter"
	"golang.org/x/term"
)

type ProxyOptions struct {
	Modes     []string     // modes available to switch to
	SaveFunc  func() error // generates a .now file
	AltScreen bool         // agent is already in alt-screen (skip toolbar on start)
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// Proxy runs the CLI terminal with a bottom toolbar.
// Returns the next mode to switch to, or "" on quit.
func Proxy(stream adapter.Stream, sessionName, hostAgent string, opts *ProxyOptions) (string, error) {
	fd := int(os.Stdin.Fd())

	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return "", fmt.Errorf("setting raw mode: %w", err)
	}
	defer func() {
		fmt.Fprintf(os.Stdout, "\033[r")    // reset scroll region
		fmt.Fprintf(os.Stdout, "\033[?25h") // show cursor
		fmt.Fprintf(os.Stdout, "\033[0m")   // reset attributes
		term.Restore(fd, oldState)
		fmt.Fprintf(os.Stdout, "\r\n")
	}()

	var mu sync.Mutex
	w, h := getSize(fd)
	inAltScreen := false
	pickingMode := false
	pickIndex := 0

	title := fmt.Sprintf("%s — %s", sessionName, hostAgent)
	setTitle := func() {
		fmt.Fprintf(os.Stdout, "\033]0;%s\007", title)
	}
	setTitle()

	// Show toolbar: scroll region rows 1..(h-1), bar on row h
	prevH := h
	showToolbar := func() {
		mu.Lock()
		defer mu.Unlock()
		oldH := prevH
		w, h = getSize(fd)
		// Clear old bar row if height changed
		if oldH != h && oldH > 0 {
			fmt.Fprintf(os.Stdout, "\0337")
			fmt.Fprintf(os.Stdout, "\033[r")                // temporarily reset scroll region
			fmt.Fprintf(os.Stdout, "\033[%d;1H\033[2K", oldH) // clear old bar row
			fmt.Fprintf(os.Stdout, "\0338")
		}
		prevH = h
		fmt.Fprintf(os.Stdout, "\033[1;%dr", h-1)
		renderNormalBar(w, h, sessionName, hostAgent, opts)
	}

	// Hide toolbar: full screen for alt-screen apps
	hideToolbar := func() {
		mu.Lock()
		defer mu.Unlock()
		w, h = getSize(fd)
		prevH = h
		fmt.Fprintf(os.Stdout, "\033[r")
	}

	// Set up initial layout
	if opts != nil && opts.AltScreen {
		// Agent already in alt-screen — full passthrough, no toolbar
		// Bounce resize to force the agent to redraw (its initial render was drained)
		inAltScreen = true
		stream.Resize(w, h-1)
		stream.Resize(w, h)
	} else {
		fmt.Fprintf(os.Stdout, "\033[2J")   // clear entire screen
		fmt.Fprintf(os.Stdout, "\033[1;1H") // cursor home
		showToolbar()
		fmt.Fprintf(os.Stdout, "\033[1;1H") // cursor to content area
		stream.Resize(w, h-1)
	}

	watchResize(fd, func() {
		setTitle()
		mu.Lock()
		alt := inAltScreen
		mu.Unlock()
		if alt {
			mu.Lock()
			w, h = getSize(fd)
			mu.Unlock()
			stream.Resize(w, h)
		} else {
			showToolbar()
			mu.Lock()
			cw, ch := w, h
			mu.Unlock()
			stream.Resize(cw, ch-1)
		}
	})

	done := make(chan struct{})
	var once sync.Once
	nextMode := ""
	finish := func() { once.Do(func() { close(done) }) }

	// Remote → stdout
	go func() {
		buf := make([]byte, 32*1024)
		count := 0
		for {
			n, readErr := stream.Read(buf)
			if n > 0 {
				os.Stdout.Write(buf[:n])
				count++

				// Periodic toolbar refresh (skip during alt-screen or picker)
				if count%50 == 0 {
					setTitle()
					mu.Lock()
					alt := inAltScreen
					picking := pickingMode
					cw, ch := w, h
					mu.Unlock()
					if !alt && !picking {
						renderNormalBar(cw, ch, sessionName, hostAgent, opts)
					}
				}

				chunk := string(buf[:n])

				// Detect alt-screen enter
				if contains(chunk, "\033[?1049h") {
					mu.Lock()
					inAltScreen = true
					mu.Unlock()
					hideToolbar()
					mu.Lock()
					cw, ch := w, h
					mu.Unlock()
					stream.Resize(cw, ch)
				}

				// Detect alt-screen exit
				mu.Lock()
				alt := inAltScreen
				mu.Unlock()
				if alt && contains(chunk, "\033[?1049l") {
					mu.Lock()
					inAltScreen = false
					mu.Unlock()
					finish()
					return
				}
			}
			if readErr != nil {
				finish()
				return
			}
		}
	}()

	// Stdin → remote
	go func() {
		buf := make([]byte, 1024)
		for {
			n, readErr := os.Stdin.Read(buf)
			if readErr != nil {
				finish()
				return
			}

			mu.Lock()
			picking := pickingMode
			mu.Unlock()

			// --- Picker mode: input goes to picker, not remote ---
			if picking {
				handlePickerInput(buf[:n], &mu, &pickingMode, &pickIndex, &nextMode, opts, w, h, sessionName, hostAgent, finish)
				continue
			}

			// --- Normal mode ---
			out := make([]byte, 0, n)
			for i := 0; i < n; i++ {
				// Filter terminal focus events: ESC [ I (focus in) and ESC [ O (focus out)
				if buf[i] == 0x1B && i+2 < n && buf[i+1] == '[' && (buf[i+2] == 'I' || buf[i+2] == 'O') {
					i += 2
					continue
				}

				switch buf[i] {
				case 0x11: // Ctrl-Q: quit
					finish()
					return
				case 0x13: // Ctrl-S: save
					if opts != nil && opts.SaveFunc != nil {
						go opts.SaveFunc()
					}
				case 0x1C: // Ctrl-\: open mode picker
					if opts != nil && len(opts.Modes) > 0 {
						mu.Lock()
						pickingMode = true
						pickIndex = 0
						renderPickerBar(w, h, opts.Modes, 0)
						mu.Unlock()
					}
				default:
					out = append(out, buf[i])
				}
			}
			if len(out) > 0 {
				if _, writeErr := stream.Write(out); writeErr != nil {
					finish()
					return
				}
			}
		}
	}()

	<-done
	return nextMode, nil
}

func handlePickerInput(input []byte, mu *sync.Mutex, pickingMode *bool, pickIndex *int, nextMode *string, opts *ProxyOptions, w, h int, session, hostAgent string, finish func()) {
	n := len(input)
	modes := opts.Modes

	for i := 0; i < n; i++ {
		b := input[i]

		// Arrow keys: ESC [ A/B/C/D
		if b == 0x1B && i+2 < n && input[i+1] == '[' {
			switch input[i+2] {
			case 'A', 'D': // Up or Left: previous
				mu.Lock()
				if *pickIndex > 0 {
					*pickIndex--
				}
				renderPickerBar(w, h, modes, *pickIndex)
				mu.Unlock()
			case 'B', 'C': // Down or Right: next
				mu.Lock()
				if *pickIndex < len(modes)-1 {
					*pickIndex++
				}
				renderPickerBar(w, h, modes, *pickIndex)
				mu.Unlock()
			}
			i += 2
			continue
		}

		switch b {
		case '\t', 0x1C: // Tab or Ctrl-\: cycle
			mu.Lock()
			*pickIndex = (*pickIndex + 1) % len(modes)
			renderPickerBar(w, h, modes, *pickIndex)
			mu.Unlock()

		case '\r', '\n': // Enter: confirm selection
			mu.Lock()
			*nextMode = modes[*pickIndex]
			*pickingMode = false
			mu.Unlock()
			finish()
			return

		case 0x1B: // Escape (standalone): cancel
			mu.Lock()
			*pickingMode = false
			renderNormalBar(w, h, session, hostAgent, opts)
			mu.Unlock()

		case 0x11: // Ctrl-Q: quit even during pick
			finish()
			return
		}
	}
}

// --- Bar rendering ---

func getSize(fd int) (int, int) {
	w, h, err := term.GetSize(fd)
	if err != nil || h < 3 {
		h = 24
	}
	if err != nil || w < 10 {
		w = 80
	}
	return w, h
}

func renderNormalBar(width, height int, session, hostAgent string, opts *ProxyOptions) {
	fmt.Fprintf(os.Stdout, "\0337")              // save cursor
	fmt.Fprintf(os.Stdout, "\033[%d;1H", height) // move to bottom row

	left := fmt.Sprintf(" ⧉ nowbox | %s | %s", session, hostAgent)

	var shortcuts []string
	if opts != nil && opts.SaveFunc != nil {
		shortcuts = append(shortcuts, "^S Save")
	}
	if opts != nil && len(opts.Modes) > 0 {
		shortcuts = append(shortcuts, "^\\ Switch")
	}
	shortcuts = append(shortcuts, "^Q Quit")
	right := " " + strings.Join(shortcuts, "  ") + " "

	gap := width - len(left) - len(right)
	if gap < 1 {
		gap = 1
	}

	bar := left + strings.Repeat(" ", gap) + right
	if len(bar) > width {
		bar = bar[:width]
	}

	fmt.Fprintf(os.Stdout, "\033[7m%s\033[0m", bar)
	fmt.Fprintf(os.Stdout, "\0338") // restore cursor
}

func renderPickerBar(width, height int, modes []string, selected int) {
	fmt.Fprintf(os.Stdout, "\0337")              // save cursor
	fmt.Fprintf(os.Stdout, "\033[%d;1H", height) // move to bottom row

	// Build mode labels with selection highlight
	label := " Switch to: "
	var modeLabels []string
	for i, m := range modes {
		if i == selected {
			modeLabels = append(modeLabels, fmt.Sprintf("\033[1m[%s]\033[22m", m))
		} else {
			modeLabels = append(modeLabels, m)
		}
	}
	modesStr := strings.Join(modeLabels, "  ")
	left := label + modesStr

	right := " Enter  Esc "

	// Calculate gap using visual widths (strip ANSI for measurement)
	leftVis := len(label)
	for i, m := range modes {
		if i == selected {
			leftVis += len(m) + 2 // brackets
		} else {
			leftVis += len(m)
		}
		if i < len(modes)-1 {
			leftVis += 2 // "  " separator
		}
	}

	gap := width - leftVis - len(right)
	if gap < 1 {
		gap = 1
	}

	bar := left + strings.Repeat(" ", gap) + right
	fmt.Fprintf(os.Stdout, "\033[7m%s\033[0m", bar)
	fmt.Fprintf(os.Stdout, "\0338") // restore cursor
}
