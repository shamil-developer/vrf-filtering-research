package progress

import (
	"fmt"
	"os"
	"sync"

	"github.com/charmbracelet/lipgloss"
)

type Line struct {
	mu     sync.Mutex
	active bool
}

func NewLine() *Line {
	return &Line{}
}

func (l *Line) Update(
	message string,
) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.active = true

	fmt.Fprintf(os.Stdout, "\r\033[2K%s", style(message))
}

func (l *Line) Done(
	message string,
) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.active {
		fmt.Fprint(os.Stdout, "\r\033[2K")
	}

	l.active = false

	if message != "" {
		fmt.Fprintln(os.Stdout, style(message))
	}
}

func style(
	message string,
) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")).
		Render(message)
}
