package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	ITEM_POTION = iota
	ITEM_MUG
	ITEM_SWORD1
	ITEM_HEAVY_ARMOR
	ITEM_LIGHT_ARMOR
	ITEM_KINGSLAYER
	ITEM_BONECRUSHER
)

type InventoryItem struct {
	id        int
	attackMod int
	dmg       string
	ac        int
	name      string
	qty       int
	heals     string
	equipped  bool
}

type Inventory struct {
	items []InventoryItem
	item  int
}

func (m *Inventory) Count(id int) int {
	for _, j := range m.items {
		if j.id == id {
			return j.qty
		}
	}
	return 0
}
func (m *Inventory) Consume(id int) {
	for s, j := range m.items {
		if j.id == id {
			j.qty--
			if j.qty == 0 {
				m.items = append(m.items[:s], m.items[s+1:]...)
			}
			return
		}
	}
}
func (m *Inventory) AddItem(id int) InventoryItem {
	for i, j := range m.items {
		if j.id == id {
			m.items[i].qty++
			return j
		}
	}
	var item InventoryItem
	switch id {
	case ITEM_POTION:
		item = InventoryItem{
			id:    id,
			qty:   1,
			name:  "Healing potion",
			heals: "2d4+2",
		}
	case ITEM_MUG:
		item = InventoryItem{
			id:   id,
			qty:  1,
			name: "Empty mug",
			dmg:  "1d4",
		}
	case ITEM_LIGHT_ARMOR:
		item = InventoryItem{
			id:   id,
			qty:  1,
			ac:   13,
			name: "Leather Armor",
		}
	case ITEM_HEAVY_ARMOR:
		item = InventoryItem{
			id:   id,
			qty:  1,
			ac:   16,
			name: "Heavy Armor",
		}
	case ITEM_SWORD1:
		item = InventoryItem{
			id:        id,
			qty:       1,
			attackMod: 1,
			name:      "Sword",
			dmg:       "1d6",
		}
	case ITEM_BONECRUSHER:
		item = InventoryItem{
			id:        id,
			qty:       1,
			attackMod: 2,
			name:      "Bonecrusher",
			dmg:       "1d8",
		}
	case ITEM_KINGSLAYER:
		item = InventoryItem{
			id:        id,
			qty:       1,
			attackMod: 3,
			name:      "Kingslayer",
			dmg:       "1d10+1",
		}
	}
	m.items = append(m.items, item)
	return item
}
func NewInventory() Inventory {
	var m Inventory
	m.items = []InventoryItem{}
	m.AddItem(ITEM_MUG)
	return m
}

func (m *Inventory) Init() tea.Cmd {
	return nil
}

func (m *Inventory) Update(msg tea.Msg) (Inventory, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.KeyMsg:
		return *m, m.handleKeyMsg(typed)
	}
	return *m, nil
}

func (m *Inventory) handleKeyMsg(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "enter":
		i := m.items[m.item]
		if i.dmg != "" {
			if i.equipped {
				m.items[m.item].equipped = false
			} else {
				// unequip existing stuff
				for j, it := range m.items {
					if it.dmg != "" && it.equipped {
						m.items[j].equipped = false
					}
				}
				m.items[m.item].equipped = true
			}
		} else if i.ac != 0 {
			if i.equipped {
				m.items[m.item].equipped = false
			} else {
				// unequip existing stuff
				for j, it := range m.items {
					if it.ac != 0 && it.equipped {
						m.items[j].equipped = false
					}
				}
				m.items[m.item].equipped = true
			}
		}
	case "up", "k", "w":
		m.item--
		if m.item < 0 {
			m.item = len(m.items) - 1
		}
	case "down", "j", "s":
		m.item++
		if m.item >= len(m.items) {
			m.item = 0
		}
	}
	return nil
}

var defaultWeapon = InventoryItem{
	name:      "fists",
	dmg:       "1d2+1",
	attackMod: 0,
}

func (m *Inventory) ArmorClass() int {
	for _, it := range m.items {
		if it.ac != 0 && it.equipped {
			return it.ac
		}
	}
	return 10
}

func (m *Inventory) Weapon() InventoryItem {
	for _, it := range m.items {
		if it.dmg != "" && it.equipped {
			return it
		}
	}
	return defaultWeapon
}

func (m *Inventory) View() string {
	var out string
	out += "Inventory\n\n"
	for i, item := range m.items {
		var eq string
		if item.equipped {
			eq = "(Equipped)"
		}
		if i == m.item {
			out += blue(fmt.Sprintf("%s %dx %s %s", ">", item.qty, item.name, eq)) + "\n"
			var description string
			if item.dmg != "" {
				description = fmt.Sprintf("+%d Weapon (%s dmg)", item.attackMod, item.dmg)
			} else if item.ac != 0 {
				description = fmt.Sprintf("Armor (%d AC)", item.ac)
			} else if item.heals != "" {
				description = fmt.Sprintf("Healing (%s HP)", item.heals)
			} else {
				description = "Item"
			}
			out += fmt.Sprintf("    -> %s", description) + "\n"
		} else {
			out += fmt.Sprintf("  %dx %s %s\n", item.qty, item.name, eq)
		}
	}
	return out
}
