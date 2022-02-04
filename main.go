package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/indent"
	"github.com/muesli/reflow/wordwrap"
	"github.com/muesli/reflow/wrap"
)

// You generally won't need this unless you're processing stuff with
// complicated ANSI escape sequences. Turn it on if you notice flickering.
//
// Also keep in mind that high performance rendering only works for programs
// that use the full size of the terminal. We're enabling that below with
// tea.EnterAltScreen().
const (
	// sockets information
	connHost = "localhost"
	connPort = "21212"
	connType = "tcp"

	// bubbletea
	useHighPerformanceRenderer = false
)

var (
	titleStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Right = "├"
		return lipgloss.NewStyle().BorderStyle(b).Padding(0, 1)
	}()

	infoStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Left = "┤"
		return titleStyle.Copy().BorderStyle(b)
	}()

	boldStyle = lipgloss.NewStyle().
			Bold(true)
)

type model struct {
	content  string
	ready    bool
	viewport viewport.Model
}

type remoteMessage struct {
	name    string
	message string
}

func initialModel() model {
	return model{
		content: "",
	}
}

func handleConnection(conn net.Conn, p *tea.Program) {
	reader := bufio.NewReader(conn)
	name := ""

	for {
		buffer, err := reader.ReadBytes('\n')

		if err != nil {
			conn.Close()
			return
		}

		str := string(buffer[:len(buffer)-1])
		if index := strings.Index(str, "::name::"); index != -1 {
			name = strings.TrimSpace(str[index+8:]) + ": "
		} else {
			p.Send(remoteMessage{name: name, message: str})
		}
	}
}

func startListener(p *tea.Program) {
	go func() {
		l, err := net.Listen(connType, connHost+":"+connPort)
		if err != nil {
			fmt.Println("Error listening to port:", err.Error())
			os.Exit(1)
		}
		defer l.Close()

		for {
			c, err := l.Accept()
			if err != nil {
				fmt.Println("Error connecting:", err.Error())
				return
			}

			go handleConnection(c, p)
		}
	}()
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {

	// Is it a key press?
	case tea.KeyMsg:
		// Cool, what was the actual key pressed?
		switch msg.String() {

		// These keys should exit the program.
		case "ctrl+c", "q":
			return m, tea.Quit

		case "c":
			m.content = ""
			m.viewport.SetContent("")
			return m, nil
		}

	case tea.WindowSizeMsg:
		headerHeight := lipgloss.Height(m.headerView())
		footerHeight := lipgloss.Height(m.footerView())
		verticalMarginHeight := headerHeight + footerHeight

		if !m.ready {
			// Since this program is using the full size of the viewport we
			// need to wait until we've received the window dimensions before
			// we can initialize the viewport. The initial dimensions come in
			// quickly, though asynchronously, which is why we wait for them
			// here.
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
			m.viewport.YPosition = headerHeight
			m.viewport.HighPerformanceRendering = useHighPerformanceRenderer
			m.viewport.SetContent(m.content)
			m.ready = true

			// This is only necessary for high performance rendering, which in
			// most cases you won't need.
			//
			// Render the viewport one line below the header.
			m.viewport.YPosition = headerHeight + 1
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMarginHeight
		}

		if useHighPerformanceRenderer {
			// Render (or re-render) the whole viewport. Necessary both to
			// initialize the viewport and when the window is resized.
			//
			// This is needed for high-performance rendering only.
			cmds = append(cmds, viewport.Sync(m.viewport))
		}

	case remoteMessage:
		if len(msg.message) > 0 {
			output := msg.message
			if len(msg.name) > 0 {
				output = boldStyle.Render(msg.name) + msg.message
			}
			output = wordwrap.String(output, m.viewport.Width-(4+len(msg.name)))
			output = indent.String(" "+output, 4)
			output = strings.TrimLeft(output, " ")
			m.content += wrap.String(output+"\n", m.viewport.Width)
			m.viewport.SetContent(m.content)
		}
		return m, nil
	}

	// Handle keyboard and mouse events in the viewport
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	// Send the UI for rendering
	return fmt.Sprintf("%s\n%s\n%s", m.headerView(), m.viewport.View(), m.footerView())
}

func (m model) headerView() string {
	title := titleStyle.Render("Debug listener -- Press q to quit, c to clear the output area.")
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func (m model) footerView() string {
	info := infoStyle.Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(info)))
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

var p *tea.Program

func main() {
	model := initialModel()
	p = tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	startListener(p)
	if err := p.Start(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
