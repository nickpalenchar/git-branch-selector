package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	branches         []string
	filteredBranches []string
	cursor           int
	filter           textinput.Model
	width            int
	height           int
	visibleStart     int
	visibleEnd       int
}

func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "Filter branches..."
	ti.Focus()

	branches := getGitBranches()
	return model{
		branches:         branches,
		filteredBranches: branches,
		filter:           ti,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEnter:
			if len(m.filteredBranches) > 0 {
				selectedBranch := m.filteredBranches[m.cursor]
				if isWorkingDirectoryDirty() {
					fmt.Print("\nYour working directory has uncommitted changes.\n")
					fmt.Print("Stash changes before switching? (Y/n): ")
					var input string
					fmt.Scanln(&input)
					if input != "n" {
						exec.Command("git", "stash").Run()
					} else {
						return m, tea.Quit
					}
				}
				exec.Command("git", "checkout", selectedBranch).Run()
			}
			return m, tea.Quit
		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
				if m.cursor < m.visibleStart {
					m.visibleStart = m.cursor
					m.visibleEnd = m.visibleStart + m.height - 4
				}
			}
		case tea.KeyDown:
			if m.cursor < len(m.filteredBranches)-1 {
				m.cursor++
				if m.cursor >= m.visibleEnd {
					m.visibleEnd = m.cursor + 1
					m.visibleStart = m.visibleEnd - (m.height - 4)
				}
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.visibleEnd = m.height - 4
		if m.visibleEnd > len(m.filteredBranches) {
			m.visibleEnd = len(m.filteredBranches)
		}
		m.visibleStart = 0
	}

	m.filter, cmd = m.filter.Update(msg)
	m.filteredBranches = filterBranches(m.branches, m.filter.Value())
	if m.cursor >= len(m.filteredBranches) {
		m.cursor = 0
	}
	if m.visibleEnd > len(m.filteredBranches) {
		m.visibleEnd = len(m.filteredBranches)
	}
	if m.visibleStart > m.visibleEnd-(m.height-4) {
		m.visibleStart = m.visibleEnd - (m.height - 4)
	}

	return m, cmd
}

func (m model) View() string {
	if len(m.branches) == 0 {
		fmt.Println("No branches found.")
		os.Exit(1)
	}

	var s strings.Builder
	s.WriteString("Select a branch:\n\n")
	s.WriteString(m.filter.View())
	s.WriteString("\n\n")

	if len(m.filteredBranches) == 0 {
		s.WriteString("No matches found.\n")
		return s.String()
	}

	for i := m.visibleStart; i < m.visibleEnd; i++ {
		branch := m.filteredBranches[i]
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		style := lipgloss.NewStyle()
		if m.cursor == i {
			style = style.Foreground(lipgloss.Color("205"))
		}
		s.WriteString(fmt.Sprintf("%s %s\n", cursor, style.Render(branch)))
	}

	return s.String()
}

func main() {
	p := tea.NewProgram(initialModel())
	if err := p.Start(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}

func getGitBranches() []string {
	currentBranch, _ := exec.Command("git", "branch", "--show-current").Output()
	currentBranchStr := strings.TrimSpace(string(currentBranch))

	reflogOutput, _ := exec.Command("git", "reflog", "show", "--pretty=format:%gs").Output()
	lines := strings.Split(strings.TrimSpace(string(reflogOutput)), "\n")
	var branches []string
	seen := make(map[string]bool)
	for _, line := range lines {
		if strings.Contains(line, "checkout:") {
			parts := strings.Split(line, " ")
			if len(parts) > 0 {
				branch := parts[len(parts)-1]
				if !seen[branch] && branch != currentBranchStr {
					seen[branch] = true
					branches = append(branches, branch)
				}
			}
		}
	}
	if len(branches) > 17 {
		branches = branches[:17]
	}

	if len(branches) > 0 {
		return branches
	}

	allBranches, _ := exec.Command("git", "branch", "--format=%(refname:short)").Output()
	branches = strings.Split(strings.TrimSpace(string(allBranches)), "\n")
	var filtered []string
	for _, branch := range branches {
		if branch != currentBranchStr {
			filtered = append(filtered, branch)
		}
	}
	return filtered
}

func isWorkingDirectoryDirty() bool {
	status, _ := exec.Command("git", "status", "--porcelain=v1").Output()
	return strings.Contains(string(status), " M")
}

func filterBranches(branches []string, filter string) []string {
	if filter == "" {
		return branches
	}
	filter = strings.ToLower(filter)
	var filtered []string
	for _, branch := range branches {
		if strings.Contains(strings.ToLower(branch), filter) {
			filtered = append(filtered, branch)
		}
	}
	return filtered
}
