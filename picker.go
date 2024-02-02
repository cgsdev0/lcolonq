package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type PickerModel struct {
	items []PickerItem
	item  int
}

type PickerItem struct {
	text string
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
	case "enter":
		switch strings.Split(m.items[m.item].text, " ")[0] {
		case "continue":
			return RunCmd
		case "run":
			return RunCmd
		case "melee":
			return MeleeCmd
		case "healing":
			return HealingCmd
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

func (m *PickerModel) View() string {
	var out string
	for i, item := range m.items {
		if i == m.item {
			out += blue(fmt.Sprintf("%s %s", ">", item.text)) + "\n"
		} else {
			out += fmt.Sprintf("  %s\n", item.text)
		}
	}
	return out
}
