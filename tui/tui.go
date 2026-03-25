package tui

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"super-duper-fortnight/clkup"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"
)

// style

var (
	baseStyle = lipgloss.NewStyle().Padding(1, 2)

	menuStyle = lipgloss.NewStyle().
			Width(30).
			PaddingRight(2).
			MarginRight(2).
			Border(lipgloss.NormalBorder(), false, true, false, false).
			BorderForeground(lipgloss.Color("#5A189A"))

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7B2CBF")).
			Bold(true).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			MarginBottom(1)

	statBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#5A189A")).
			Padding(0, 2).
			MarginRight(2)

	statLabelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#9D4EDD"))
	statValueStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#E0AAFF"))
)

// state

type uiState int

const (
	stateInit uiState = iota
	stateIdle
	stateFetchingPlan
	stateFetchingData
	stateLoaded
)

// MSGs

type LogMsg string

type InitDataMsg struct {
	User       clkup.User
	Workspaces []clkup.Workspace
}
type PlanLoadedMsg struct {
	TeamID string
	PlanID int
}
type FanOutCompleteMsg struct {
	Spaces      []clkup.Space
	Folders     []clkup.Folder
	Lists       []clkup.List
	Tasks       []clkup.Task
	Performance clkup.Performance
}
type ErrMsg struct{ err error }

// model
type dashboardModel struct {
	apiClient *clkup.APIClient
	spinner   spinner.Model
	logChan   chan string
	logs      []string

	width  int
	height int

	// State
	state  uiState
	status string
	err    error

	// Selection & Focus
	cursor       int
	activeTeamID string
	focusTable   bool

	// UI
	taskTable table.Model

	// Data Store
	user       clkup.User
	workspaces []clkup.Workspace
	spaces     []clkup.Space
	folders    []clkup.Folder
	lists      []clkup.List
	tasks      []clkup.Task

	perf clkup.Performance
}

func InitialModel(client *clkup.APIClient) dashboardModel {
	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7B2CBF"))
	logChan := make(chan string, 1000)
	client.LogChan = logChan

	return dashboardModel{
		apiClient: client,
		spinner:   s,
		state:     stateInit,
		status:    "Fetching User and Workspace data...",
		logChan:   logChan,
	}
}

func (m dashboardModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		fetchInitDataCmd(m.apiClient),
		waitForLog(m.logChan),
	)
}

func waitForLog(c chan string) tea.Cmd {
	return func() tea.Msg {
		return LogMsg(<-c)
	}
}

func (m dashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		if m.state == stateLoaded {
			m.taskTable.SetWidth(m.width - 36)
			m.taskTable.SetHeight(m.height - 15)
		}
		return m, nil

	case tea.KeyMsg:
		// Global quit
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		// right pane
		if m.focusTable {
			switch msg.String() {
			case "esc", "left":
				m.focusTable = false
				m.taskTable.Blur()
				return m, nil
			case "q":
				return m, tea.Quit
			default:
				// Pass all other keys (j, k, up, down) to the table
				m.taskTable, cmd = m.taskTable.Update(msg)
				return m, cmd
			}
		}

		// left pane
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.workspaces)-1 {
				m.cursor++
			}
		case "right":
			if m.state == stateLoaded && string(m.workspaces[m.cursor].ID) == m.activeTeamID {
				m.focusTable = true
				m.taskTable.Focus()
			}
		case "enter":
			if len(m.workspaces) > 0 && (m.state == stateIdle || m.state == stateLoaded) {
				selectedID := m.workspaces[m.cursor].ID

				// If they hit enter on the already-loaded workspace, just jump to the table
				if m.state == stateLoaded && string(selectedID) == m.activeTeamID {
					m.focusTable = true
					m.taskTable.Focus()
					return m, nil
				}

				// Otherwise, start a fresh fetch
				m.activeTeamID = string(selectedID)
				m.state = stateFetchingPlan
				m.status = fmt.Sprintf("Fetching plan for workspace '%s'...", m.workspaces[m.cursor].Name)

				m.spaces = nil
				m.folders = nil
				m.lists = nil
				m.tasks = nil

				return m, tea.Batch(m.spinner.Tick, fetchPlanCmd(m.apiClient, m.activeTeamID))
			}
		}

	case spinner.TickMsg:
		if m.state == stateInit || m.state == stateFetchingPlan || m.state == stateFetchingData {
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case InitDataMsg:
		m.user = msg.User
		m.workspaces = msg.Workspaces
		m.state = stateIdle
		return m, nil

	case PlanLoadedMsg:
		rpm := 100
		if msg.PlanID == 3 {
			rpm = 1000
		}
		if msg.PlanID == 4 {
			rpm = 10000
		}
		safeLimit := rate.Every(time.Minute / time.Duration(float64(rpm)*0.95))

		m.apiClient.Limiter = rate.NewLimiter(safeLimit, 1)
		m.state = stateFetchingData
		m.status = fmt.Sprintf("Fan-out fetch initiated at %d RPM...", rpm)

		return m, fetchHierarchyCmd(m.apiClient, msg.TeamID)

	case LogMsg:
		// Append the new log
		m.logs = append(m.logs, string(msg))

		if len(m.logs) > 8 {
			m.logs = m.logs[1:]
		}

		//call the command again to wait for the next log
		return m, waitForLog(m.logChan)

	case FanOutCompleteMsg:
		m.spaces = msg.Spaces
		m.folders = msg.Folders
		m.lists = msg.Lists
		m.tasks = msg.Tasks
		m.perf = msg.Performance

		// tasks table
		columns := []table.Column{
			{Title: "Task ID", Width: 12},
			{Title: "Status", Width: 15},
			{Title: "Name", Width: 60},
		}

		var rows []table.Row
		for _, t := range m.tasks {
			rows = append(rows, table.Row{string(t.Id), t.Status.Status, t.Name})
		}

		m.taskTable = table.New(
			table.WithColumns(columns),
			table.WithRows(rows),
			table.WithFocused(true),
			table.WithHeight(15),
		)

		// Styling the table
		s := table.DefaultStyles()
		s.Header = s.Header.
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			BorderBottom(true).
			Bold(false)
		s.Selected = s.Selected.
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57")).
			Bold(false)
		m.taskTable.SetStyles(s)

		m.state = stateLoaded
		m.focusTable = true
		return m, nil

	case ErrMsg:
		m.err = msg.err
		m.state = stateLoaded
		m.status = "API Error Encountered."
		return m, nil
	}

	return m, nil
}

func (m dashboardModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\nPress 'q' to quit.", m.err)
	}

	var header string
	if m.state != stateInit {
		headerStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E0AAFF")).
			Padding(0, 1).
			MarginBottom(1).
			Italic(true)

		leftSide := fmt.Sprintf("%s | %s - %s", m.user.ID, m.user.Initials, m.user.Email)
		rightSide := fmt.Sprintf("[ %s ]", m.user.Timezone)

		spaceCount := m.width - lipgloss.Width(leftSide) - lipgloss.Width(rightSide) - 4

		var headerContent string
		if spaceCount > 0 {
			spacer := strings.Repeat(" ", spaceCount)
			headerContent = leftSide + spacer + rightSide
		} else {
			headerContent = leftSide + " " + rightSide
		}

		header = headerStyle.Render(headerContent)
	}

	// left pane
	var menuItems []string
	menuTitle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7B2CBF")).Render("Workspaces")
	menuItems = append(menuItems, menuTitle, "")

	if m.state == stateInit {
		menuItems = append(menuItems, "Loading...")
	} else {
		for i, w := range m.workspaces {
			cursor := "  "
			if m.cursor == i && !m.focusTable {
				cursor = "> "
			}

			style := lipgloss.NewStyle()
			if string(w.ID) == m.activeTeamID {
				style = style.Foreground(lipgloss.Color("#E0AAFF")).Bold(true)
			} else if m.cursor == i && !m.focusTable {
				style = style.Foreground(lipgloss.Color("#9D4EDD"))
			} else {
				style = style.Foreground(lipgloss.Color("240"))
			}

			menuItems = append(menuItems, fmt.Sprintf("%s%s", cursor, style.Render(w.Name)))
		}
	}

	leftPane := menuStyle.Render(strings.Join(menuItems, "\n"))

	// right pane
	var rightPane string

	if m.state == stateInit || m.state == stateFetchingPlan || m.state == stateFetchingData {
		rightPane = fmt.Sprintf("\n %s %s\n", m.spinner.View(), m.status)

	} else if m.state == stateIdle {
		rightPane = "\n\n  <-- Select a Workspace and press Enter to load data."

	} else if m.state == stateLoaded {
		activeName := "Unknown"
		for _, w := range m.workspaces {
			if string(w.ID) == m.activeTeamID {
				activeName = w.Name
				break
			}
		}

		title := titleStyle.Render(fmt.Sprintf("Dashboard | %s", activeName))

		statSpaces := statBoxStyle.Render(fmt.Sprintf("%s\n%s", statLabelStyle.Render("Spaces"), statValueStyle.Render(fmt.Sprint(len(m.spaces)))))
		statFolders := statBoxStyle.Render(fmt.Sprintf("%s\n%s", statLabelStyle.Render("Folders"), statValueStyle.Render(fmt.Sprint(len(m.folders)))))
		statLists := statBoxStyle.Render(fmt.Sprintf("%s\n%s", statLabelStyle.Render("Lists"), statValueStyle.Render(fmt.Sprint(len(m.lists)))))
		statTasks := statBoxStyle.Render(fmt.Sprintf("%s\n%s", statLabelStyle.Render("Tasks In Memory"), statValueStyle.Render(fmt.Sprint(len(m.tasks)))))

		statsRow := lipgloss.JoinHorizontal(lipgloss.Top, statSpaces, statFolders, statLists, statTasks)

		helpText := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("Press 'esc' or 'left' to return to menu • 'q' to quit")
		if !m.focusTable {
			helpText = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("Press 'enter' or 'right' to browse tasks • 'q' to quit")
		}

		rightPane = lipgloss.JoinVertical(lipgloss.Left,
			title,
			"\n",
			statsRow,
			"\n",
			m.taskTable.View(),
			"\n",
			helpText,
		)
	}

	logBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Width(m.width-4).
		Height(10).
		Padding(0, 1).
		MarginTop(1)

	logTitle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("--- Live API Event Log ---")
	logContent := strings.Join(m.logs, "\n")

	bottomPane := logBoxStyle.Render(lipgloss.JoinVertical(lipgloss.Left, logTitle, logContent))

	// combine layout
	topPanes := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	finalView := lipgloss.JoinVertical(lipgloss.Left, header, topPanes, bottomPane)

	if m.state == stateLoaded {
		perfStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E0AAFF")).
			Padding(0, 1).
			MarginTop(1).
			Italic(true)

		perfText := fmt.Sprintf("Fetch completed in %s | Tasks Per Second: %s | Est. RPM: %s",
			m.perf.Duration, m.perf.TPS, m.perf.RPM)

		footer := perfStyle.Render(perfText)
		finalView = lipgloss.JoinVertical(lipgloss.Left, finalView, footer)
	}

	return baseStyle.Render(finalView)

}

func fetchPlanCmd(client *clkup.APIClient, teamID string) tea.Cmd {
	return func() tea.Msg {
		plan, err := client.GetPlan(teamID)
		if err != nil {
			return ErrMsg{err}
		}
		return PlanLoadedMsg{
			TeamID: teamID,
			PlanID: plan.PlanID,
		}
	}
}

func fetchInitDataCmd(client *clkup.APIClient) tea.Cmd {
	return func() tea.Msg {
		var user clkup.User
		var workspaces []clkup.Workspace
		var err error

		for attempts := 0; attempts < 3; attempts++ {
			user, err = client.GetAuthorizedUser()
			if err == nil {
				break
			}
			time.Sleep(1 * time.Second)
		}
		if err != nil {
			return ErrMsg{fmt.Errorf("failed to fetch user after 3 attempts: %w", err)}
		}

		for attempts := 0; attempts < 3; attempts++ {
			workspaces, err = client.GetAuthorizedWorkspaces()
			if err == nil {
				break
			}
			time.Sleep(1 * time.Second)
		}
		if err != nil {
			return ErrMsg{fmt.Errorf("workspace API error after 3 attempts: %w", err)}
		}

		if len(workspaces) == 0 {
			return ErrMsg{fmt.Errorf("success, but workspace array was empty")}
		}

		return InitDataMsg{
			User:       user,
			Workspaces: workspaces,
		}
	}
}
func fetchHierarchyCmd(client *clkup.APIClient, teamID string) tea.Cmd {
	return func() tea.Msg {
		start := time.Now()
		var g errgroup.Group
		var mu sync.Mutex
		var finalLists []clkup.List
		var finalSpaces []clkup.Space
		var finalFolders []clkup.Folder
		var finalTasks []clkup.Task

		// concurrent task fetch
		g.Go(func() error {
			tasks, err := client.GetAllTasks(teamID)
			if err == nil {
				finalTasks = tasks
			}
			return err
		})

		// concurrent hierarchy fetch
		g.Go(func() error {
			spaces, err := client.GetSpaces(teamID)
			if err != nil {
				return err
			}

			mu.Lock()
			finalSpaces = append(finalSpaces, spaces...)
			mu.Unlock()

			for _, space := range spaces {
				sID := string(space.ID)

				// Fetch Folders for this Space
				g.Go(func() error {
					folders, err := client.GetFolders(sID)
					if err != nil {
						return err
					}

					mu.Lock()
					finalFolders = append(finalFolders, folders...)
					mu.Unlock()

					// Fan out into each folder to get its Lists
					for _, folder := range folders {
						fID := string(folder.ID)
						g.Go(func() error {
							lists, err := client.GetLists(fID)
							if err != nil {
								return err
							}

							mu.Lock()
							finalLists = append(finalLists, lists...)
							mu.Unlock()
							return nil
						})
					}
					return nil
				})

				// Fetch Folderless Lists for this Space concurrently
				g.Go(func() error {
					folderlessLists, err := client.GetFolderlessLists(sID)
					if err != nil {
						return err
					}

					mu.Lock()
					finalLists = append(finalLists, folderlessLists...)
					mu.Unlock()
					return nil
				})
			}

			return nil
		})

		if err := g.Wait(); err != nil {
			return ErrMsg{err}
		}

		perf := clkup.CalculatePerformance(len(finalTasks), start)

		return FanOutCompleteMsg{
			Spaces:      finalSpaces,
			Folders:     finalFolders,
			Lists:       finalLists,
			Tasks:       finalTasks,
			Performance: perf,
		}
	}
}
