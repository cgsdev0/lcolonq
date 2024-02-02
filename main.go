package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	goaway "github.com/TwiN/go-away"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"

	bm "github.com/charmbracelet/wish/bubbletea"

	lm "github.com/charmbracelet/wish/logging"
	"github.com/muesli/termenv"
)

func parsePort(port string) int {
	i, err := strconv.Atoi(port)
	if err != nil {
		return 23234
	}
	return i
}

var (
	host = "0.0.0.0"
	port = parsePort(os.Getenv("PORT"))
)

var (
	red      = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render
	green    = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render
	cyan     = lipgloss.NewStyle().Foreground(lipgloss.Color("36")).Render
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
		a.loadMeta(strings.Split(e.Name(), ".")[0])
	}
}

type Color struct {
	r byte
	g byte
	b byte
	a byte
}

func (c Color) toItem() int {
	return int(255 - c.g)
}

func (c Color) toEnemy() byte {
	return 255 - c.r
}

func (c Color) toNPC() byte {
	return 254 - c.b
}

func (c Color) toGateLevel() byte {
	return 255 - c.g
}

// 255 different items
func (c Color) isItem() bool {
	return c.g > 0 && c.a == 255 && c.r == 0 && c.b == 0
}

func (c Color) isGrass() bool {
	return c.b == 0 && c.a == 50 && c.r == 0 && c.g == 0
}
func (c Color) isCarpet() bool {
	return c.b == 0 && c.a == 150 && c.r == 0 && c.g == 0
}
func (c Color) isFence() bool {
	return c.b == 0 && c.a == 100 && c.r == 0 && c.g == 0
}
func (c Color) isSpawn() bool {
	return c.b == 255 && c.a == 255 && c.r == 0 && c.g == 0
}
func (c Color) isSecret() bool {
	return c.r == 255 && c.a == 255 && c.g == 255 && c.b == 0
}
func (c Color) isHole() bool {
	return c.r == 255 && c.a == 255 && c.b == 255 && c.g == 0
}
func (c Color) isEnemy() bool {
	return c.r > 0 && c.a == 255 && c.g == 0 && c.b == 0
}
func (c Color) isNPC() bool {
	return c.b > 0 && c.b < 255 && c.a == 255 && c.g == 0
}
func (c Color) isWall() bool {
	return c.r == 0 && c.a == 255 && c.g == 0 && c.b == 0
}
func (c Color) isHeal() bool {
	return c.r == 0 && c.a == 255 && c.g == 255 && c.b == 255
}
func (c Color) isGate() bool {
	return c.r == 0 && c.a == 255 && c.g < 255 && c.g > 200 && c.b == 255
}

func (c Color) render(destroyed bool, x int, y int) string {
	if c.isNPC() {
		switch c.toNPC() {
		case NPC_SIGN:
			return blue("S")
		default:
			return blue("N")
		}
	}
	if c.isCarpet() {
		return red("@")
	}
	if c.isFence() {
		return gray("+")
	}
	if c.isGrass() {
		hash := (y*14 + x*3) % 8
		switch hash {
		case 0:
			return green("\"")
		case 1:
			return green(",")
		case 2:
			return green("'")
		case 3:
			return green(".")
		default:
			return " "
		}
	}
	if c.isGate() {
		if destroyed {
			return " "
		}
		return cyan("D")
	}
	if c.isWall() {
		return "#"
	}
	if c.isSecret() {
		if destroyed {
			return darkgray("#")
		}
		return "#"
	}
	if c.isHole() && destroyed {
		return gray("X")
	}
	if c.isItem() && !destroyed {
		var letter string
		switch c.toItem() {
		case ITEM_POTION:
			letter = "P"
		case ITEM_SWORD1:
			letter = "S"
		case ITEM_KINGSLAYER:
			letter = "ʈ"
		case ITEM_BONECRUSHER:
			letter = "¶"
		case ITEM_HEAVY_ARMOR:
			letter = "H"
		case ITEM_LIGHT_ARMOR:
			letter = "A"
		default:
			letter = "I"
		}
		return yellow(letter)
	}
	if c.isEnemy() && !destroyed {
		var letter string
		switch c.toEnemy() {
		case ENEMY_BAT:
			letter = "b"
		case ENEMY_GHOSTS:
			letter = "G"
		case ENEMY_SKELETON:
			letter = "s"
		case ENEMY_MINOTAUR:
			letter = "M"
		case ENEMY_KING:
			letter = "K"
		}
		return red(letter)
	}
	return " "
}

func (m *model) inCombatView() bool {
	switch m.state {
	case IN_COMBAT:
		return true
	case IN_NPC:
		return true
	}
	return false
}
func (a *app) loadLevel(world string) {
	file, err := os.Open("./map/" + world + ".txt")
	if err != nil {
		panic(err)
	}
	defer file.Close()
	tmp := [16][40]Color{}
	buf := make([]byte, 4)
	i := 0
	for {
		n, err := file.Read(buf)
		if n == 0 || err != nil {
			break
		}
		j := i / 40
		k := i % 40
		c := Color{
			r: buf[0],
			g: buf[1],
			b: buf[2],
			a: buf[3],
		}
		if c.isSpawn() {
			fmt.Println("i found spawn")
			a.StartPos = Position{
				x:     k,
				y:     j,
				world: world,
			}
		}
		tmp[j][k] = c
		i++
	}
	a.world[world] = tmp
}
func (a *app) loadMeta(world string) {
	file, err := os.Open("./meta/" + world + ".txt")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	i := -1
	for scanner.Scan() {
		i++
		line := scanner.Text()
		if i == 0 {
			a.links[world] = []string{line}
			continue
		}
		if i < 4 {
			a.links[world] = append(a.links[world], line)
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		coords := strings.Split(parts[0], "x")
		x, err := strconv.Atoi(coords[0])
		if err != nil {
			panic(err)
		}
		y, err := strconv.Atoi(coords[1])
		if err != nil {
			panic(err)
		}
		a.dialogue[Position{
			world: world,
			x:     x,
			y:     y,
		}] = parts[1]
	}

	if err := scanner.Err(); err != nil {
		panic(err)
	}
}

func MiddlewareWithProgramHandler(bth bm.ProgramHandler, cp termenv.Profile) wish.Middleware {
	// XXX: This is a hack to make sure the default Termenv output color
	// profile is set before the program starts. Ideally, we want a Lip Gloss
	// renderer per session.
	lipgloss.SetColorProfile(cp)
	return func(sh ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			p := bth(s)
			if p != nil {
				_, windowChanges, _ := s.Pty()
				ctx, cancel := context.WithCancel(s.Context())
				go func() {
					for {
						select {
						case <-ctx.Done():
							if p != nil {
								p.Send(DisconnectMsg{})
								p.Quit()
								return
							}
						case w := <-windowChanges:
							if p != nil {
								p.Send(tea.WindowSizeMsg{Width: w.Width, Height: w.Height})
							}
						}
					}
				}()
				if _, err := p.Run(); err != nil {
					log.Error("app exit with error", "error", err)
				}
				// p.Kill() will force kill the program if it's still running,
				// and restore the terminal to its original state in case of a
				// tui crash
				p.Kill()
				cancel()
			}
			sh(s)
		}
	}
}
func main() {
	a := new(app)
	a.links = make(map[string]([]string))
	a.dialogue = make(map[Position]string)
	a.world = make(map[string]([16][40]Color))
	a.loadLevels()
	a.Positions = make(map[string]Position)
	a.Levels = make(map[string]int)
	a.Chats = make(map[string]string)
	a.Chans = make(map[string](chan tea.Msg))
	go func() {
		fmt.Println("I am the server!")
		for {
			updates := []tea.Msg{}
			updated := false
			time.Sleep(time.Millisecond * 100)
			a.ChansMutex.Lock()
			a.StateMutex.Lock()
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
					case levelMsg:
						a.Levels[msg.id] = msg.level
					case ChatMsg:
						a.Chats[msg.id] = msg.msg
						updated = true
					case HoleMsg:
						updates = append(updates, msg)
						updated = true
					case DeadMsg:
						delete(a.Positions, msg.id)
						delete(a.Chats, msg.id)
						delete(a.Levels, msg.id)
						updated = true
					case moveMsg:
						// fmt.Printf("got a move msg from %s\n", msg.id)
						a.Positions[msg.id] = msg.pos
						updated = true
					}
				}
			}
			a.StateMutex.Unlock()
			a.ChansMutex.Unlock()
			if updated {
				a.send2(rerenderMsg{updates: updates})
			}
		}
	}()
	s, err := wish.NewServer(
		wish.WithAddress(fmt.Sprintf("%s:%d", host, port)),
		wish.WithHostKeyPath(".ssh/term_info_ed25519"),
		wish.WithMiddleware(
			MiddlewareWithProgramHandler(a.ProgramHandler, termenv.ANSI256),
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
	errMsg   error
	levelMsg struct {
		id    string
		level int
	}
	moveMsg struct {
		id  string
		pos Position
	}
	rerenderMsg struct {
		updates []tea.Msg
	}
	RespawnMsg struct {
	}
	RunMsg struct {
	}
	MeleeMsg struct {
	}
	HealingMsg struct {
	}
	EnemyMsg struct {
	}
	DefeatEnemyMsg struct {
	}
	YourTurnMsg struct {
	}
	ChatMsg struct {
		id  string
		msg string
	}
	HoleMsg struct {
		id  string
		pos Position
	}
	DisconnectMsg struct {
	}
	DeadMsg struct {
		id string
	}
	ChatClearMsg struct {
	}
)

var ChatClearCmd tea.Cmd = func() tea.Msg {
	time.Sleep(time.Second * 2)
	return ChatClearMsg{}
}
var RunCmd tea.Cmd = func() tea.Msg {
	return RunMsg{}
}

var DeadCmd tea.Cmd = func() tea.Msg {
	time.Sleep(time.Second * 2)
	return DeadMsg{}
}

var HealingCmd tea.Cmd = func() tea.Msg {
	return HealingMsg{}
}
var MeleeCmd tea.Cmd = func() tea.Msg {
	return MeleeMsg{}
}

var YourTurnCmd tea.Cmd = func() tea.Msg {
	time.Sleep(time.Millisecond * 1200)
	return YourTurnMsg{}
}
var EnemyCmd tea.Cmd = func() tea.Msg {
	time.Sleep(time.Millisecond * 1200)
	return EnemyMsg{}
}
var DefeatEnemyCmd tea.Cmd = func() tea.Msg {
	time.Sleep(time.Millisecond * 1200)
	return DefeatEnemyMsg{}
}

// app contains a wish server and the list of running programs.
type app struct {
	*ssh.Server
	progs      []*tea.Program
	Positions  map[string]Position
	Levels     map[string]int
	Chats      map[string]string
	Chans      map[string](chan tea.Msg)
	ChansMutex sync.Mutex
	StateMutex sync.RWMutex
	world      map[string]([16][40]Color)
	links      map[string]([]string)
	dialogue   map[Position](string)
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
		term:           pty.Term,
		width:          pty.Window.Width,
		height:         pty.Window.Height,
		pos:            a.StartPos,
		roomStart:      a.StartPos,
		health:         7,
		maxHealth:      7,
		level:          1,
		xp:             0,
		state:          OVERWORLD,
		destroyed:      map[Position]bool{},
		percent:        0.0,
		progress:       progress.New(progress.WithSolidFill("63"), progress.WithColorProfile(termenv.ANSI256)),
		progressHealth: progress.New(progress.WithSolidFill("1"), progress.WithColorProfile(termenv.ANSI256)),
		inventory:      NewInventory(),
		chat:           textinput.New(),
	}
	m.chat.CharLimit = 30
	m.chat.Placeholder = "press T to chat"
	m.app = a
	m.id = s.RemoteAddr().String() + s.User()
	a.StateMutex.Lock()
	a.Positions[m.id] = m.pos
	a.Levels[m.id] = 1
	a.Chats[m.id] = ""
	a.StateMutex.Unlock()
	m.progress.Width = 19
	m.progress.ShowPercentage = false
	m.progressHealth.Width = 19
	m.progressHealth.ShowPercentage = false
	if strings.Split(s.RemoteAddr().String(), ":")[0] == "127.0.0.1" {
		m.hacks = true
	}

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

// States
const (
	OVERWORLD = iota
	IN_INVENTORY
	IN_COMBAT
	IN_NPC
)

type model struct {
	*app
	serverChan     chan tea.Msg
	id             string
	term           string
	width          int
	height         int
	pos            Position
	roomStart      Position
	prev           Position
	health         int
	maxHealth      int
	level          int
	xp             int
	state          int
	falling        bool
	enemy          *Enemy
	npc            *NPC
	destroyed      map[Position]bool
	chattext       string
	text           string
	combattext     string
	selection      int
	picker         PickerModel
	progress       progress.Model
	progressHealth progress.Model
	percent        float64
	inventory      Inventory
	hacks          bool
	chat           textinput.Model
	allowchat      bool
}

func (m model) Init() tea.Cmd {
	m.inventory.Init()
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
		m.roomStart = m.pos
		m.text = ""
	}
}

func (m *model) xpCurve(x int) int {
	return int(math.Pow(1+0.5, float64(x-1))*1000) - 1000
}

func (m *model) pickupItems() {
	if _, ok := m.destroyed[m.pos]; ok {
		return
	}
	cell := m.app.world[m.pos.world][m.pos.y][m.pos.x]
	if cell.isItem() {
		m.destroyed[m.pos] = true
		item := m.inventory.AddItem(cell.toItem())
		m.text = fmt.Sprintf("Found %s!", item.name)
	}
}

func (m *model) doHeals() {
	cell := m.app.world[m.pos.world][m.pos.y][m.pos.x]
	if cell.isHeal() {
		m.health = m.maxHealth
		m.text = "You feel refreshed."
	}
}
func (m *model) revealSecrets() {
	cell := m.app.world[m.pos.world][m.pos.y][m.pos.x]
	if cell.isSecret() {
		m.destroyed[m.pos] = true
	}
}

func (m *model) checkTraps() tea.Cmd {
	cell := m.app.world[m.pos.world][m.pos.y][m.pos.x]
	if cell.isHole() {
		m.destroyed[m.pos] = true
		m.text = "You fell in a hole!"
		m.falling = true
		m.send(HoleMsg{id: m.id, pos: m.pos})
		var cmd tea.Cmd = func() tea.Msg {
			time.Sleep(time.Second)
			return RespawnMsg{}
		}
		return cmd
	}
	return nil
}

func (m *model) updateOptions() {
	m.picker.item = 0
	if m.state == IN_NPC {
		m.picker.items = []PickerItem{
			{
				text: "continue",
			},
		}
		return
	}
	m.picker.items = []PickerItem{
		{
			text: fmt.Sprintf("melee attack (%s)", m.inventory.Weapon().name),
		},
		{
			text: "run away",
		},
	}
	count := m.inventory.Count(ITEM_POTION)
	if count > 0 {
		m.picker.items = append(m.picker.items, PickerItem{
			text: fmt.Sprintf("healing potion (%d)", count),
		})
	}
}
func (m *model) startCombat() {
	if _, ok := m.destroyed[m.pos]; ok {
		return
	}
	cell := m.app.world[m.pos.world][m.pos.y][m.pos.x]
	if cell.isEnemy() {
		m.text = ""
		m.combattext = ""
		m.state = IN_COMBAT
		m.enemy = createEnemy(cell.toEnemy())
		m.updateOptions()
	}
}

func (m *model) isBlocked() bool {
	_, ok := m.destroyed[m.pos]
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
	if cell.isNPC() {
		m.state = IN_NPC
		m.npc = createNPC(cell.toNPC(), m.app.dialogue[m.pos])
		m.combattext = ""
		m.updateOptions()
		return true
	}
	if cell.isFence() {
		return true
	}
	if cell.isGate() && m.level < int(cell.toGateLevel()) {
		m.text = fmt.Sprintf("You must be level %d.", cell.toGateLevel())
		return true
	}
	if cell.isGate() {
		m.text = "The door opened!"
		return false
	}
	if ok {
		if cell.isHole() {
			return true
		} else {
			return false
		}
	}
	if cell.isWall() {
		return true
	}
	return false
}

func (m *model) playerAc() int {
	return m.inventory.ArmorClass()
}

func (m *model) playerAttack() int {
	var advantage bool
	if m.enemy.id == ENEMY_KING && m.inventory.Weapon().id == ITEM_KINGSLAYER {
		advantage = true
	}
	roll1 := Roll(fmt.Sprintf("1d20+%d", m.inventory.Weapon().attackMod+4))
	if !advantage {
		return roll1
	}
	roll2 := Roll(fmt.Sprintf("1d20+%d", m.inventory.Weapon().attackMod+4))
	if roll1 > roll2 {
		return roll1
	}
	return roll2
}

func (m *model) playerDamage() int {
	return Roll(m.inventory.Weapon().dmg)
}

func (m *model) move(x int, y int) tea.Cmd {
	var cmd tea.Cmd
	if m.falling {
		return nil
	}
	if m.state != OVERWORLD {
		return nil
	}
	m.prev = m.pos
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
		m.doHeals()
		m.pickupItems()
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

	case HoleMsg:
		if m.pos.world == msg.pos.world {
			m.destroyed[msg.pos] = true
		}
	case rerenderMsg:
		for _, u := range msg.updates {
			m2, cmd := m.Update(u)
			if m3, ok := m2.(model); ok {
				m = m3
			}
			cmds = append(cmds, cmd)
		}

	case HealingMsg:
		m.inventory.Consume(ITEM_POTION)
		m.updateOptions()
		healing := Roll("2d4+2")
		if healing > m.maxHealth-m.health {
			healing = m.maxHealth - m.health
		}
		m.combattext = fmt.Sprintf("You healed for %d!", healing)
		m.health += healing
		cmd = EnemyCmd
	case MeleeMsg:
		m.updateOptions()
		hit := m.playerAttack() >= m.enemy.ac
		cmd = EnemyCmd
		if !hit {
			m.combattext = "You missed!"
		} else {
			dmg := m.playerDamage()
			m.combattext = fmt.Sprintf("You dealt %d damage!", dmg)
			m.enemy.health -= dmg
			if m.enemy.health <= 0 {
				m.enemy.health = 0
				cmd = DefeatEnemyCmd
			}
		}
	case DefeatEnemyMsg:
		xpGained := m.enemy.level * 250
		m.xp += xpGained
		for m.xp >= m.xpCurve(m.level+1) {
			m.level++
			m.send(levelMsg{
				id:    m.id,
				level: m.level,
			})
			rolledHealth := Roll("1d6+1")
			m.maxHealth += rolledHealth
			m.health += rolledHealth
		}
		base := m.xpCurve(m.level)
		next := m.xpCurve(m.level + 1)
		xpNeeded := next - base
		xpHave := m.xp - base
		m.percent = float64(xpHave) / float64(xpNeeded)
		fmt.Printf("have %d want %d percent %f\n", xpHave, xpNeeded, m.percent)
		m.state = OVERWORLD
		m.destroyed[m.pos] = true
		m.text = fmt.Sprintf("You defeated %s!", m.enemy.name)

	case EnemyMsg:
		hit := Roll(m.enemy.attack) >= m.enemy.ac
		if !hit {
			m.combattext = fmt.Sprintf("%s attacked, but missed!", m.enemy.name)
		} else {
			dmg := Roll(m.enemy.damage)
			m.combattext = fmt.Sprintf("%s dealt %d damage!", m.enemy.name, dmg)
			m.health -= dmg
		}
		if m.health <= 0 {
			m.health = 0
			m.combattext = "You died!"
			cmd = DeadCmd
		} else {
			cmd = YourTurnCmd
		}

	case DeadMsg:
		m.pos = m.app.StartPos
		m.text = ""
		m.state = OVERWORLD
		m.send(moveMsg{
			id:  m.id,
			pos: m.pos,
		})

	case YourTurnMsg:
		m.combattext = ""
	case RunMsg:
		m.updateOptions()
		m.state = OVERWORLD
		m.pos = m.prev
		m.send(moveMsg{
			id:  m.id,
			pos: m.pos,
		})
	case RespawnMsg:
		m.falling = false
		m.pos = m.roomStart
		m.text = ""
		m.send(moveMsg{
			id:  m.id,
			pos: m.pos,
		})
	case DisconnectMsg:
		m.send(DeadMsg{
			id: m.id,
		})
	case ChatClearMsg:
		m.chattext = ""
		m.send(ChatMsg{
			id:  m.id,
			msg: "",
		})
	case tea.KeyMsg:
		if !m.chat.Focused() {
			switch msg.String() {
			// case "enter":
			// 	if m.combat {
			// 		m.combat = false
			// 		m.destroyed[m.pos] = true
			// 	}
			case "!":
				m.allowchat = !m.allowchat
			case "t":
				m.chat.Focus()
				return m, nil
			case " ":
				if m.hacks {
					m.text = m.pos.world
				}
			case "0":
				if m.hacks {
					m.level++
					m.send(levelMsg{
						id:    m.id,
						level: m.level,
					})
					rolledHealth := Roll("1d6+1")
					m.maxHealth += rolledHealth
					m.health += rolledHealth
					m.text = "Level up (cheats)"
				}

			case "i", "e", "esc", "q", "tab":
				if m.state == OVERWORLD {
					m.inventory.item = 0
					m.state = IN_INVENTORY
				} else if m.state == IN_INVENTORY {
					m.state = OVERWORLD
				}
			case "left", "h", "a":
				cmd = m.move(-1, 0)
			case "right", "l", "d":
				cmd = m.move(1, 0)
			case "up", "k", "w":
				cmd = m.move(0, -1)
			case "down", "j", "s":
				cmd = m.move(0, 1)
			case "ctrl+c":
				m.send(DeadMsg{
					id: m.id,
				})
				return m, tea.Quit
			}
		}
	}
	cmds = append(cmds, cmd)
	if m.state == IN_INVENTORY {
		m.inventory, cmd = m.inventory.Update(msg)
		cmds = append(cmds, cmd)
	}
	if m.inCombatView() && len(m.combattext) == 0 {
		m.picker, cmd = m.picker.Update(msg)
		cmds = append(cmds, cmd)
	}
	if m.chat.Focused() {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				m.send(ChatMsg{
					id:  m.id,
					msg: m.chat.Value(),
				})
				m.chattext = m.chat.Value()
				m.chat.Blur()
				m.chat.SetValue("")
				return m, ChatClearCmd
			}
		}
	}

	// update chat
	m.chat, cmd = m.chat.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

type Player struct {
	pos   Position
	level int
	chat  string
}

func (m model) View() string {
	var chatBubble = lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("63"))
	var mainBox = lipgloss.NewStyle().Width(40).Height(16).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("63"))
	players := []Player{}
	m.app.StateMutex.RLock()
	for id, pos := range m.app.Positions {
		if m.pos.world != pos.world {
			continue
		}
		if id == m.id {
			continue
		}
		players = append(players, Player{pos: pos, level: m.app.Levels[id], chat: m.app.Chats[id]})
	}
	m.app.StateMutex.RUnlock()
	var s string
	if m.state == IN_INVENTORY {
		s = m.inventory.View()
		s = mainBox.Render(s)
	} else if !m.inCombatView() {
		for r, row := range m.app.world[m.pos.world] {
		outer:
			for c, cell := range row {
				for r == m.pos.y && c == m.pos.x {
					s += green("U")
					continue outer
				}
				_, destroyed := m.destroyed[Position{x: c, y: r, world: m.pos.world}]
				if (cell.isEnemy() || cell.isSecret()) && !destroyed {
					s += cell.render(destroyed, c, r)
					continue outer
				}
				for _, p := range players {
					if r == p.pos.y && c == p.pos.x {
						levelT := fmt.Sprintf("%d", p.level)
						if p.level > 9 {
							levelT = "+"
						}
						s += blue(levelT)
						continue outer
					}
				}
				s += cell.render(destroyed, c, r)
			}
			s += "\n"
		}
		last := len(s) - 1
		s = s[:last]
		s = mainBox.Render(s)
		if m.allowchat {
			if m.chattext != "" {
				s = lipgloss.PlaceOverlay(m.pos.x, m.pos.y-2, chatBubble.Render(m.chattext), s)
			}
			for _, p := range players {
				if p.chat != "" {
					s = lipgloss.PlaceOverlay(p.pos.x, p.pos.y-2, chatBubble.Render(goaway.Censor(p.chat)), s)
				}
			}
		}
	} else {
		// combat oh no
		if m.state == IN_COMBAT {
			s += fmt.Sprintf("You entered combat with %s!\n\n", m.enemy.name)
			s += fmt.Sprintf("Enemy health: %d / %d\n", m.enemy.health, m.enemy.maxhealth)
			s += m.enemy.art + "\n"
		} else {
			s += fmt.Sprintf("You see %s.\n\n", m.npc.name)
			s += m.npc.art + "\n"
			s += fmt.Sprintf("\"%s\"\n\n", m.npc.dialogue)
		}
		if len(m.combattext) > 0 {
			s += m.combattext
		} else {
			s += m.picker.View()
		}
		s = mainBox.Render(s)
	}

	s += "\n"
	var xpBar string
	var healthBar string
	xpBar += " " + m.progress.ViewAs(m.percent)
	xpBar += fmt.Sprintf("\n Level:  %d\n ", m.level)
	healthBar += "  " + m.progressHealth.ViewAs(float64(m.health)/float64(m.maxHealth))
	healthBar += fmt.Sprintf("\n  Health: %d / %d\n", m.health, m.maxHealth)
	bars := lipgloss.JoinHorizontal(lipgloss.Top, xpBar, healthBar)
	s += bars
	s += red(fmt.Sprintf("\n           %s", m.text)) + "\n"
	if m.allowchat {
		s += m.chat.View()
	} else {
		s += gray("Chat disabled (press '!' to enable)")
	}
	return s
}
