package main

// An example Bubble Tea server. This will put an ssh session into alt screen
// and continually print up to date terminal information.

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	bm "github.com/charmbracelet/wish/bubbletea"
	lm "github.com/charmbracelet/wish/logging"
	"github.com/muesli/termenv"
)

const (
	host = "localhost"
	port = 23234
)

var (
	red      = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render
	green    = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render
	yellow   = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render
	blue     = lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Render
	gray     = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render
	darkgray = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render
)

func (a *app) loadLevels() {
	entries, err := os.ReadDir("./map")
	if err != nil {
		log.Fatal(err)
	}

	for _, e := range entries {
		a.loadLevel(strings.Split(e.Name(), ".")[0])
	}
}

func (a *app) loadLevel(world string) {
	tmp := [16][40]uint8{}
	file, err := os.Open("./map/" + world + ".txt")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	i := -1
	for scanner.Scan() {
		i++
		line := scanner.Text()
		if i == 16 {
			a.links[world] = []string{line}
			continue
		}
		if i > 16 {
			a.links[world] = append(a.links[world], line)
			continue
		}
		for j, c := range []byte(line) {
			switch c {
			case 'S':
				tmp[i][j] = '.'
				a.StartPos = Position{
					x:     j,
					y:     i,
					world: world,
				}
			default:
				tmp[i][j] = c
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	a.world[world] = tmp
}

func main() {
	a := new(app)
	a.links = make(map[string]([]string))
	a.world = make(map[string]([16][40]byte))
	a.loadLevels()
	a.Positions = make(map[string]Position)
	a.Chans = make(map[string](chan tea.Msg))
	go func() {
		fmt.Println("I am the server!")
		fmt.Printf("%v\n", a.Chans)
		for {
			updated := false
			a.ChansMutex.Lock()
			time.Sleep(time.Millisecond * 100)
			for _, ch := range a.Chans {
			loop:
				for {
					// fmt.Printf("length of channel %s: %d\n", id, len(ch))
					var thing tea.Msg
					select {
					case thing = <-ch:
					default:
						break loop
					}
					switch msg := thing.(type) {
					case moveMsg:
						// fmt.Printf("got a move msg from %s\n", msg.id)
						a.StateMutex.Lock()
						a.Positions[msg.id] = msg.pos
						a.StateMutex.Unlock()
						updated = true
					}
				}
			}
			a.ChansMutex.Unlock()
			if updated {
				a.send2(rerenderMsg{})
			}
		}
	}()
	s, err := wish.NewServer(
		wish.WithAddress(fmt.Sprintf("%s:%d", host, port)),
		wish.WithHostKeyPath(".ssh/term_info_ed25519"),
		wish.WithMiddleware(
			bm.MiddlewareWithProgramHandler(a.ProgramHandler, termenv.ANSI256),
			lm.Middleware(),
		),
	)
	if err != nil {
		log.Error("could not start server", "error", err)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	log.Info("Starting SSH server", "host", host, "port", port)
	go func() {
		if err = s.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Error("could not start server", "error", err)
			done <- nil
		}
	}()

	<-done
	log.Info("Stopping SSH server")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer func() { cancel() }()
	if err := s.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		log.Error("could not stop server", "error", err)
	}
}

type (
	errMsg  error
	moveMsg struct {
		id  string
		pos Position
	}
	rerenderMsg struct {
	}
	RespawnMsg struct {
	}
)

// app contains a wish server and the list of running programs.
type app struct {
	*ssh.Server
	progs      []*tea.Program
	Positions  map[string]Position
	Chans      map[string](chan tea.Msg)
	ChansMutex sync.Mutex
	StateMutex sync.RWMutex
	world      map[string]([16][40]uint8)
	links      map[string]([]string)
	StartPos   Position
}

func (a *app) send2(msg tea.Msg) {
	for _, p := range a.progs {
		go p.Send(msg)
	}
}

// send dispatches a message to the server thread.
func (m *model) send(msg tea.Msg) {
	m.serverChan <- msg
}

func (a *app) ProgramHandler(s ssh.Session) *tea.Program {
	pty, _, active := s.Pty()
	if !active {
		wish.Fatalln(s, "terminal is not active")
	}

	m := model{
		term:      pty.Term,
		width:     pty.Window.Width,
		height:    pty.Window.Height,
		pos:       a.StartPos,
		health:    7,
		maxHealth: 7,
		level:     1,
		xp:        0,
		combat:    false,
		destroyed: map[Position]bool{},
	}
	m.app = a
	m.id = s.RemoteAddr().String() + s.User()
	a.StateMutex.Lock()
	a.Positions[m.id] = m.pos
	a.StateMutex.Unlock()

	ch := make(chan tea.Msg, 100)
	a.ChansMutex.Lock()
	defer a.ChansMutex.Unlock()
	a.Chans[m.id] = ch
	m.serverChan = ch
	p := tea.NewProgram(m, tea.WithOutput(s), tea.WithInput(s), tea.WithAltScreen())
	a.progs = append(a.progs, p)

	return p
}

type Position struct {
	world string
	x     int
	y     int
}

// Just a generic tea.Model to demo terminal information of ssh.
type model struct {
	*app
	serverChan chan tea.Msg
	id         string
	term       string
	width      int
	height     int
	pos        Position
	health     int
	maxHealth  int
	level      int
	xp         int
	combat     bool
	falling    bool
	enemy      *Enemy
	destroyed  map[Position]bool
	text       string
	selection  int
	picker     PickerModel
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m *model) doWarp() {
	warp := -1
	if m.pos.y < 0 {
		warp = 0
		m.pos.y = len(m.app.world[m.pos.world]) - 1
	}
	if m.pos.x < 0 {
		warp = 3
		m.pos.x = len(m.app.world[m.pos.world][m.pos.y]) - 1
	}
	if m.pos.y >= len(m.app.world[m.pos.world]) {
		warp = 2
		m.pos.y = 0
	}
	if m.pos.x >= len(m.app.world[m.pos.world][m.pos.y]) {
		warp = 1
		m.pos.x = 0
	}
	if warp != -1 {
		m.pos.world = m.app.links[m.pos.world][warp]
	}
}

func (m *model) revealSecrets() {
	cell := m.app.world[m.pos.world][m.pos.y][m.pos.x]
	if cell == '$' {
		m.destroyed[m.pos] = true
	}
}

func (m *model) checkTraps() tea.Cmd {
	cell := m.app.world[m.pos.world][m.pos.y][m.pos.x]
	if cell == 'x' {
		m.destroyed[m.pos] = true
		m.text = "You fell in a hole!"
		m.falling = true
		var cmd tea.Cmd = func() tea.Msg {
			time.Sleep(time.Second)
			return RespawnMsg{}
		}
		return cmd
	}
	return nil
}

func (m *model) startCombat() {
	if _, ok := m.destroyed[m.pos]; ok {
		return
	}
	cell := m.app.world[m.pos.world][m.pos.y][m.pos.x]
	if cell == 'b' {
		m.text = ""
		m.combat = true
		m.enemy = letterToEnemy(cell)
	}
}

func (m *model) isBlocked() bool {
	if _, ok := m.destroyed[m.pos]; ok {
		return false
	}
	if m.pos.y < 0 {
		return false
	}
	if m.pos.x < 0 {
		return false
	}
	if m.pos.y >= len(m.app.world[m.pos.world]) {
		return false
	}
	if m.pos.x >= len(m.app.world[m.pos.world][m.pos.y]) {
		return false
	}
	cell := m.app.world[m.pos.world][m.pos.y][m.pos.x]
	if cell == '#' {
		return true
	}
	return false
}

type Enemy struct {
	health    int
	maxhealth int
	name      string
}

func letterToEnemy(c byte) *Enemy {
	switch c {
	case 'b':
		return &Enemy{
			name:      "a bat",
			health:    5,
			maxhealth: 5,
		}
	default:
		// this should never happen!
		return &Enemy{
			name:      "MISSINGNO",
			health:    100,
			maxhealth: 100,
		}
	}
}
func (m *model) move(x int, y int) tea.Cmd {
	var cmd tea.Cmd
	if m.falling {
		return nil
	}
	if m.combat {
		return nil
	}
	m.pos.x += x
	m.pos.y += y
	if m.isBlocked() {
		m.pos.x -= x
		m.pos.y -= y
	} else {
		m.doWarp()
		m.startCombat()
		cmd = m.checkTraps()
		m.revealSecrets()
		m.send(moveMsg{
			id:  m.id,
			pos: m.pos,
		})
	}
	return cmd
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width

	case RespawnMsg:
		m.falling = false
		m.pos = m.app.StartPos
		m.text = ""
		m.send(moveMsg{
			id:  m.id,
			pos: m.pos,
		})
	case moveMsg:
		// do something
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if m.combat {
				m.combat = false
				m.destroyed[m.pos] = true
			}
		case "left", "h", "a":
			cmd = m.move(-1, 0)
		case "right", "l", "d":
			cmd = m.move(1, 0)
		case "up", "k", "w":
			cmd = m.move(0, -1)
		case "down", "j", "s":
			cmd = m.move(0, 1)
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	cmds = append(cmds, cmd)
	if m.combat {
		m.picker, cmd = m.picker.Update(msg)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	var mainBox = lipgloss.NewStyle().Width(40).Height(16).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("63"))
	players := []Position{}
	m.app.StateMutex.RLock()
	for id, pos := range m.app.Positions {
		if m.pos.world != pos.world {
			continue
		}
		if id == m.id {
			continue
		}
		players = append(players, pos)
	}
	m.app.StateMutex.RUnlock()
	var s string
	if !m.combat {
		for r, row := range m.app.world[m.pos.world] {
		outer:
			for c, cell := range row {
				for r == m.pos.y && c == m.pos.x {
					s += green("U")
					continue outer
				}
				if _, ok := m.destroyed[Position{x: c, y: r, world: m.pos.world}]; ok {
					if cell == 'x' {
						cell = 'X'
					} else if cell == '$' {
						cell = 'Z'
					} else {
						cell = '.'
					}
				}
				for _, pos := range players {
					if r == pos.y && c == pos.x {
						s += blue("O")
						continue outer
					}
				}
				if cell == 'x' {
					s += " "
				} else if cell == 'Z' {
					s += darkgray("#")
				} else if cell == 'X' {
					s += gray("X")
				} else if cell == 'b' {
					s += red("b")
				} else if cell == '.' {
					s += " "
				} else if cell == '$' {
					s += "#"
				} else {
					s += string([]byte{cell})
				}
			}
			s += "\n"
		}
		last := len(s) - 1
		s = s[:last]
		s = mainBox.Render(s)
	} else {
		// combat oh no
		s += fmt.Sprintf("You have entered combat with %s!\n\n", m.enemy.name)
		s += fmt.Sprintf("Enemy health: %d / %d\n", m.enemy.health, m.enemy.maxhealth)
		s += m.picker.View()
		s = mainBox.Render(s)
	}

	s += "\n"
	s += fmt.Sprintf(" Level:  %d\n", m.level)
	s += fmt.Sprintf(" Health: %d / %d\n", m.health, m.maxHealth)
	s += red(fmt.Sprintf("\n           %s\n", m.text))
	return s
}
