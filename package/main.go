package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type branchItem string

func (i branchItem) Title() string       { return string(i) }
func (i branchItem) Description() string { return "" }
func (i branchItem) FilterValue() string { return string(i) }

type compactDelegate struct {
	list.DefaultDelegate
}

func (d compactDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(branchItem)
	if !ok {
		return
	}

	str := fmt.Sprintf("  %s", i.Title())
	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render(">", strings.TrimLeft(s[0], " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

var (
	itemStyle         = lipgloss.NewStyle().Padding(0, 0, 0, 0)
	selectedItemStyle = lipgloss.NewStyle().Padding(0, 0, 0, 0).Foreground(lipgloss.Color("205"))
)

type model struct {
	list list.Model
}

func initialModel() model {
	items := getGitBranches()
	listItems := make([]list.Item, len(items))
	for i, item := range items {
		listItems[i] = branchItem(item)
	}
	l := list.New(listItems, compactDelegate{}, 0, 0)
	l.Title = "Select a branch"
	l.SetFilteringEnabled(true)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = lipgloss.NewStyle().MarginLeft(0)
	l.SetShowHelp(false)

	return model{
		list: l,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEnter:
			if len(m.list.Items()) > 0 {
				selectedBranch := m.list.SelectedItem().(list.DefaultItem).Title()
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
		}
	case tea.WindowSizeMsg:
		h, v := lipgloss.NewStyle().Margin(1, 2).GetFrameSize()
		m.list.SetWidth(msg.Width - h)
		m.list.SetHeight(msg.Height - v)
	}

	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if len(m.list.Items()) == 0 {
		fmt.Println("No branches found.")
		os.Exit(1)
	}

	return lipgloss.NewStyle().Margin(1, 1).Render(m.list.View())
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
