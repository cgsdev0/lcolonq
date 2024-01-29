package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

type PickerModel struct {
	items []PickerItem
	item  int
}

type PickerItem struct {
	text    string
	checked bool
}

func (m *PickerModel) Init() tea.Cmd {
	return nil
}

func (m *PickerModel) Update(msg tea.Msg) (PickerModel, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.KeyMsg:
		return *m, m.handleKeyMsg(typed)
	}
	return *m, nil
}

func (m *PickerModel) handleKeyMsg(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc", "ctrl+c":
		return tea.Quit
	case " ", "enter":
		m.items[m.item].checked = !m.items[m.item].checked
	case "up":
		if m.item > 0 {
			m.item--
		}
	case "down":
		if m.item+1 < len(m.items) {
			m.item++
		}
	}
	return nil
}

func (m *PickerModel) View() string {
	return m.renderList("Choose your action:", m.items, m.item)
}

func (m *PickerModel) renderList(header string, items []PickerItem, selected int) string {
	out := "~ " + header + ":\n"
	for i, item := range items {
		sel := " "
		if i == selected {
			sel = ">"
		}
		check := " "
		if items[i].checked {
			check = "✓"
		}
		out += fmt.Sprintf("%s [%s] %s\n", sel, check, item.text)
	}
	return out
}
