package tui

import (
	"fmt"
	"strings"
	"time"

	"goclicu/clkup"
	"goclicu/dbstore"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/time/rate"
)

// style
var (
	baseStyle = lipgloss.NewStyle().Padding(0, 2)

	menuStyle = lipgloss.NewStyle().
			Width(30).
			PaddingRight(2).
			MarginRight(2).
			Border(lipgloss.NormalBorder(), false, true, false, false).
			BorderForeground(lipgloss.Color("#5A189A"))

	statBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#5A189A")).
			Padding(0, 2).
			MarginRight(2).
			Align(lipgloss.Center)

	statLabelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#9D4EDD"))
	statValueStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#E0AAFF"))
)

const (
	stateInit uiState = iota
	stateIdle
	stateFetchingPlan
	stateFetchingData
	stateLoaded
	stateResetConfirm
)

type resetCompleteMsg struct{ err error }

const (
	DepthWorkspaces ViewDepth = iota
	DepthSpaces
	DepthFolders
	DepthLists
	DepthTasks
	DepthTaskDetails
)

const (
	SyncOff SyncInterval = iota
	Sync5Min
	Sync15Min
	Sync30Min
)

// model
type dashboardModel struct {
	apiClient *clkup.APIClient
	db        *dbstore.DB
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
	activeTeamID string
	depth        ViewDepth

	// Cursors & Offsets
	cursorWorkspace  int
	cursorSpace      int
	cursorFolder     int
	cursorList       int
	cursorTask       int
	taskScrollOffset int

	focusRight bool

	// JSON Popup State
	showJSONPopup    bool
	jsonScrollOffset int
	jsonCopied       bool

	// Data Store
	user         clkup.User
	workspaces   []clkup.Workspace
	teamPerf     map[string]clkup.Performance
	syncInterval SyncInterval
}

func InitialModel(client *clkup.APIClient, db *dbstore.DB) dashboardModel {
	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7B2CBF"))
	logChan := make(chan string, 1000)
	client.LogChan = logChan

	return dashboardModel{
		apiClient:     client,
		db:            db,
		spinner:       s,
		state:         stateInit,
		status:        "Fetching User and Workspace data...",
		logChan:       logChan,
		teamPerf:      make(map[string]clkup.Performance),
		showJSONPopup: false,
		focusRight:    false,
		syncInterval:  SyncOff,
	}
}

func (m dashboardModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		fetchInitDataCmd(m.apiClient, m.db, false),
		waitForLog(m.logChan),
	)
}

func waitForLog(c chan string) tea.Cmd {
	return func() tea.Msg {
		return LogMsg(<-c)
	}
}

func (m dashboardModel) getCurrentJSON() string {
	_, _, rightRawText := m.getRightPane()
	if rightRawText == "" {
		return "{\n  \"error\": \"No JSON data available for this selection.\"\n}"
	}
	return rightRawText
}

func (m dashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if m.state == stateResetConfirm {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "y", "Y":
				m.state = stateInit
				m.status = "Rebuilding database..."
				return m, resetDatabaseCmd(m.db)
			case "n", "N", "esc":
				m.state = stateLoaded
				return m, nil
			}
		}
		return m, nil
	}

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case clearCopyMsg:
		m.jsonCopied = false
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}

		switch msg.String() {

		case "ctrl+x": // Trigger Reset
			m.state = stateResetConfirm
			return m, nil

		case "o":
			url := m.getCurrentSelectionURL()
			if url != "" {
				err := OpenBrowser(url)
				if err != nil {
					m.logs = append(m.logs, fmt.Sprintf("Failed to open browser: %v", err))
				} else {
					m.logs = append(m.logs, fmt.Sprintf("Opened in browser: %s", url))
				}
				if len(m.logs) > 4 {
					m.logs = m.logs[len(m.logs)-4:]
				}
			} else {
				m.logs = append(m.logs, "No valid URL for current selection.")
				if len(m.logs) > 4 {
					m.logs = m.logs[len(m.logs)-4:]
				}
			}
			return m, nil

		// not neded at this time, uncomment for right pane focus if needed
		// case "tab":
		// 	// Toggle focus between left and right panes
		// 	m.focusRight = !m.focusRight
		// 	return m, nil

		case "J": // Shift + j
			m.showJSONPopup = !m.showJSONPopup
			m.jsonScrollOffset = 0
			m.jsonCopied = false
			return m, nil

		case "S": // Shift + s
			if m.showJSONPopup {
				rawJSON := m.getCurrentJSON()
				err := clipboard.WriteAll(rawJSON)
				if err == nil {
					m.jsonCopied = true
					return m, tea.Tick(time.Second*3, func(t time.Time) tea.Msg {
						return clearCopyMsg{}
					})
				} else {
					m.logs = append(m.logs, fmt.Sprintf("Clipboard error: %v", err))
				}
			}
			return m, nil

		case "j", "down":
			if m.showJSONPopup {
				m.jsonScrollOffset++
				return m, nil
			}

			if m.depth == DepthTaskDetails || m.focusRight {
				m.taskScrollOffset++
				return m, nil
			}

			leftItems, _, _ := m.getLeftPane()
			maxIdx := len(leftItems) - 1
			if maxIdx < 0 {
				break
			}
			switch m.depth {
			case DepthWorkspaces:
				if m.cursorWorkspace < maxIdx {
					m.cursorWorkspace++
				}
			case DepthSpaces:
				if m.cursorSpace < maxIdx {
					m.cursorSpace++
					m.cursorFolder, m.cursorList, m.cursorTask = 0, 0, 0
				}
			case DepthFolders:
				if m.cursorFolder < maxIdx {
					m.cursorFolder++
					m.cursorList, m.cursorTask = 0, 0
				}
			case DepthLists:
				if m.cursorList < maxIdx {
					m.cursorList++
					m.cursorTask = 0
				}
			case DepthTasks:
				if m.cursorTask < maxIdx {
					m.cursorTask++
					m.taskScrollOffset = 0
				}
			}

			m.taskScrollOffset = 0

		case "pgdown", "ctrl+d":
			if m.showJSONPopup {
				m.jsonScrollOffset += 10
				return m, nil
			}
			m.taskScrollOffset += 10
			return m, nil

		case "pgup", "ctrl+u":
			if m.showJSONPopup {
				if m.jsonScrollOffset > 10 {
					m.jsonScrollOffset -= 10
				} else {
					m.jsonScrollOffset = 0
				}
				return m, nil
			}
			if m.taskScrollOffset > 10 {
				m.taskScrollOffset -= 10
			} else {
				m.taskScrollOffset = 0
			}
			return m, nil

		case "k", "up":
			if m.showJSONPopup {
				if m.jsonScrollOffset > 0 {
					m.jsonScrollOffset--
				}
				return m, nil
			}

			if m.depth == DepthTaskDetails || m.focusRight {
				if m.taskScrollOffset > 0 {
					m.taskScrollOffset--
				}
				return m, nil
			}

			switch m.depth {
			case DepthWorkspaces:
				if m.cursorWorkspace > 0 {
					m.cursorWorkspace--
				}
			case DepthSpaces:
				if m.cursorSpace > 0 {
					m.cursorSpace--
					m.cursorFolder, m.cursorList, m.cursorTask = 0, 0, 0
				}
			case DepthFolders:
				if m.cursorFolder > 0 {
					m.cursorFolder--
					m.cursorList, m.cursorTask = 0, 0
				}
			case DepthLists:
				if m.cursorList > 0 {
					m.cursorList--
					m.cursorTask = 0
				}
			case DepthTasks:
				if m.cursorTask > 0 {
					m.cursorTask--
					m.taskScrollOffset = 0
				}
			}

			m.taskScrollOffset = 0

		case "l", "right", "enter", " ":
			if m.showJSONPopup {
				return m, nil
			}
			if m.depth == DepthTaskDetails || m.focusRight {
				return m, nil
			}
			if m.depth == DepthTasks {
				m.depth = DepthTaskDetails
				m.taskScrollOffset = 0
				return m, nil
			}

			leftItems, _, _ := m.getLeftPane()
			if len(leftItems) == 0 {
				return m, nil
			}

			switch m.depth {
			case DepthWorkspaces:
				if m.state == stateIdle || m.state == stateLoaded {
					selectedWS := string(m.workspaces[m.cursorWorkspace].ID)

					spaces := m.db.GetSpaces(selectedWS)
					if len(spaces) > 0 {
						m.activeTeamID = selectedWS
						m.depth = DepthSpaces
						m.cursorSpace, m.cursorFolder, m.cursorList, m.cursorTask = 0, 0, 0, 0
						return m, nil
					}

					m.activeTeamID = selectedWS
					m.state = stateFetchingPlan
					m.status = fmt.Sprintf("Fetching plan for workspace '%s'...", m.workspaces[m.cursorWorkspace].Name)

					return m, tea.Batch(m.spinner.Tick, fetchPlanCmd(m.apiClient, m.activeTeamID))
				}
			case DepthSpaces:
				m.depth = DepthFolders
				m.cursorFolder = 0
			case DepthFolders:
				item := leftItems[m.cursorFolder]
				if item.Type == "folder" {
					m.depth = DepthLists
					m.cursorList = 0
				} else if item.Type == "list" {
					m.depth = DepthTasks
					m.cursorTask = 0
				}
			case DepthLists:
				m.depth = DepthTasks
				m.cursorTask = 0
			}

		case "h", "left", "esc", "backspace":
			if m.showJSONPopup {
				m.showJSONPopup = false
				return m, nil
			}

			if m.focusRight {
				m.focusRight = false
				return m, nil
			}

			switch m.depth {
			case DepthSpaces:
				m.depth = DepthWorkspaces
			case DepthFolders:
				m.depth = DepthSpaces
			case DepthLists:
				m.depth = DepthFolders
			case DepthTasks:
				spaces := m.db.GetSpaces(m.activeTeamID)
				if len(spaces) > 0 && m.cursorSpace < len(spaces) {
					sID := string(spaces[m.cursorSpace].ID)
					folders := m.db.GetFolders(sID)
					if m.cursorFolder < len(folders) {
						m.depth = DepthLists
					} else {
						m.depth = DepthFolders
					}
				} else {
					m.depth = DepthFolders
				}
			case DepthTaskDetails:
				m.depth = DepthTasks
				m.taskScrollOffset = 0
			}

		case "r":
			if m.showJSONPopup {
				return m, nil
			}
			// if viewing the main Workspaces screen, refresh user and Workspace data
			if m.depth == DepthWorkspaces {
				m.state = stateInit
				m.status = "Force syncing User and Workspaces..."
				return m, tea.Batch(m.spinner.Tick, fetchInitDataCmd(m.apiClient, m.db, true)) // true = force bypass SQLite
			}

			// If viewing a Workspace, force sync the tasks and folders
			if (m.state == stateLoaded || m.state == stateIdle) && m.activeTeamID != "" {
				m.state = stateFetchingPlan
				m.status = "Force syncing workspace data from ClickUp..."
				return m, tea.Batch(m.spinner.Tick, fetchPlanCmd(m.apiClient, m.activeTeamID))
			}
		case "F": // Shift + f
			if m.showJSONPopup {
				return m, nil
			}
			switch m.syncInterval {
			case SyncOff:
				m.syncInterval = Sync5Min
			case Sync5Min:
				m.syncInterval = Sync15Min
			case Sync15Min:
				m.syncInterval = Sync30Min
			case Sync30Min:
				m.syncInterval = SyncOff
			}

			if m.syncInterval != SyncOff {
				return m, tickAutoSync(m.syncInterval.Duration())
			}
			return m, nil
		}

	case resetCompleteMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.status = "Database cleared. Re-fetching data..."
		return m, fetchInitDataCmd(m.apiClient, m.db, true)

	case spinner.TickMsg:
		if m.state == stateInit || m.state == stateFetchingPlan || m.state == stateFetchingData {
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case InitDataMsg:
		m.user = msg.User
		m.workspaces = msg.Workspaces
		m.state = stateIdle
		m.depth = DepthWorkspaces
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

		return m, fetchHierarchyCmd(m.apiClient, m.db, msg.TeamID)

	case LogMsg:
		m.logs = append(m.logs, string(msg))
		if len(m.logs) > 8 {
			m.logs = m.logs[1:]
		}
		return m, waitForLog(m.logChan)

	case FanOutCompleteMsg:
		if m.teamPerf == nil {
			m.teamPerf = make(map[string]clkup.Performance)
		}
		m.teamPerf[msg.TeamID] = msg.Performance

		if m.state == stateFetchingData || m.state == stateFetchingPlan {
			m.state = stateLoaded
			m.depth = DepthWorkspaces
		}
		return m, nil

	case autoSyncTickMsg:
		if m.syncInterval == SyncOff || m.activeTeamID == "" {
			return m, nil
		}

		return m, tea.Batch(
			tickAutoSync(m.syncInterval.Duration()),
			fetchHierarchyCmd(m.apiClient, m.db, m.activeTeamID),
		)

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

	safeTextWidth := m.width - 6
	if safeTextWidth < 20 {
		safeTextWidth = 20
	}

	// 2. HEADER
	var header string
	if m.state != stateInit {
		headerInnerWidth := safeTextWidth - 2

		leftSide := fmt.Sprintf("%s | %s - %s", m.user.ID, m.user.Initials, m.user.Email)
		rightTop := fmt.Sprintf("[%s]", m.user.Timezone)

		syncColor := "#5A189A"
		if m.syncInterval != SyncOff {
			syncColor = "#00FF00"
		}
		rightBot := lipgloss.NewStyle().Foreground(lipgloss.Color(syncColor)).Render(m.syncInterval.String())

		leftW := lipgloss.Width(leftSide)
		rightTopW := lipgloss.Width(rightTop)

		if leftW+rightTopW >= headerInnerWidth {
			availLeft := headerInnerWidth - rightTopW - 1
			if availLeft > 3 {
				runes := []rune(leftSide)
				leftSide = string(runes[:availLeft-1]) + "…"
			} else {
				leftSide = ""
			}
		}

		leftW = lipgloss.Width(leftSide)
		spaceCountTop := headerInnerWidth - leftW - rightTopW
		if spaceCountTop < 0 {
			spaceCountTop = 0
		}
		line1 := leftSide + strings.Repeat(" ", spaceCountTop) + rightTop

		rightBotW := lipgloss.Width(rightBot)
		spaceCountBot := headerInnerWidth - rightBotW
		if spaceCountBot < 0 {
			spaceCountBot = 0
		}
		line2 := strings.Repeat(" ", spaceCountBot) + rightBot

		headerStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E0AAFF")).
			Padding(0, 1).
			MarginBottom(1).
			Italic(true)

		headerContent := line1 + "\n" + line2
		header = headerStyle.Render(headerContent)
	}

	logInnerWidth := safeTextWidth - 4
	if logInnerWidth < 10 {
		logInnerWidth = 10
	}

	logBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Width(logInnerWidth).
		Height(4).
		Padding(0, 1)

	logTitle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("--- Live API Event Log ---")

	displayLogs := m.logs
	if len(displayLogs) > 3 {
		displayLogs = displayLogs[len(displayLogs)-3:]
	}

	var safeLogs []string
	for _, l := range displayLogs {
		runes := []rune(l)
		if len(runes) > logInnerWidth {
			safeLogs = append(safeLogs, string(runes[:logInnerWidth-3])+"...")
		} else {
			safeLogs = append(safeLogs, l)
		}
	}
	logContent := strings.Join(safeLogs, "\n")
	bottomPane := logBoxStyle.Render(lipgloss.JoinVertical(lipgloss.Left, logTitle, logContent))

	helpStr := "Nav: h j k l | Back: esc | Select: enter | JSON: Shift+J | Sync: r | Cycle Auto-Sync: SHIFT+F | Open: o | Quit: q | Reset Database: CTRL+X"

	if lipgloss.Width(helpStr) > safeTextWidth {
		// Medium width fallback
		helpStr = "Nav: hjkl | esc: back | enter: sel | J: json | r: sync | F: auto-sync | o: open | q: quit | ctrl+x: reset data"

		if lipgloss.Width(helpStr) > safeTextWidth {
			// Small width fallback
			helpStr = "hjkl:nav | esc:back | enter:sel | J:json | r:sync | F:auto-sync | o:web | q:quit | ctrl+x:reset"

			if lipgloss.Width(helpStr) > safeTextWidth {
				// Extreme squish fallback
				runes := []rune(helpStr)
				helpStr = string(runes[:safeTextWidth-1]) + "…"
			}
		}
	}
	helpText := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).MarginBottom(1).Render(helpStr)

	var footer string
	if (m.state == stateLoaded || m.state == stateIdle) && m.activeTeamID != "" {
		if perf, ok := m.teamPerf[m.activeTeamID]; ok {
			perfText := fmt.Sprintf("Last fetch completed in %s | Tasks Per Second: %s | Est. RPM: %s",
				perf.Duration, perf.TPS, perf.RPM)
			if lipgloss.Width(perfText) > safeTextWidth {
				runes := []rune(perfText)
				perfText = string(runes[:safeTextWidth-1]) + "…"
			}
			footer = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#E0AAFF")).
				MarginTop(1).
				Italic(true).
				Render(perfText)
		}
	}

	var bottomStack string
	if footer != "" {
		bottomStack = lipgloss.JoinVertical(lipgloss.Left, helpText, bottomPane, footer)
	} else {
		bottomStack = lipgloss.JoinVertical(lipgloss.Left, helpText, bottomPane)
	}

	if m.state == stateInit || m.state == stateFetchingPlan || m.state == stateFetchingData {
		loadingContent := lipgloss.NewStyle().Margin(2, 0).Render(fmt.Sprintf("%s %s", m.spinner.View(), m.status))
		if footer != "" {
			return baseStyle.Render(lipgloss.JoinVertical(lipgloss.Left, header, loadingContent, bottomPane, footer))
		}
		return baseStyle.Render(lipgloss.JoinVertical(lipgloss.Left, header, loadingContent, bottomPane))
	}

	sCount, fCount, lCount, tCount := m.getStats()
	statSpaces := statBoxStyle.Render(fmt.Sprintf("%s\n%s", statLabelStyle.Render("Spaces"), statValueStyle.Render(sCount)))
	statFolders := statBoxStyle.Render(fmt.Sprintf("%s\n%s", statLabelStyle.Render("Folders"), statValueStyle.Render(fCount)))
	statLists := statBoxStyle.Render(fmt.Sprintf("%s\n%s", statLabelStyle.Render("Lists"), statValueStyle.Render(lCount)))
	statTasks := statBoxStyle.Render(fmt.Sprintf("%s\n%s", statLabelStyle.Render("Tasks"), statValueStyle.Render(tCount)))

	statsRow := lipgloss.NewStyle().MarginBottom(1).Render(
		lipgloss.JoinHorizontal(lipgloss.Top, statSpaces, statFolders, statLists, statTasks),
	)

	breadcrumbs := lipgloss.NewStyle().MarginBottom(1).Render(m.getBreadcrumbs(safeTextWidth))

	topStack := lipgloss.JoinVertical(lipgloss.Left, header, statsRow, breadcrumbs)

	occupiedHeight := lipgloss.Height(topStack) + lipgloss.Height(bottomStack)

	paneHeight := m.height - occupiedHeight - 1
	if paneHeight < 5 {
		paneHeight = 5
	}

	paneWidthLeft := safeTextWidth / 2
	paneWidthRight := safeTextWidth - paneWidthLeft

	if paneWidthLeft < 10 {
		paneWidthLeft = 10
	}
	if paneWidthRight < 10 {
		paneWidthRight = 10
	}

	leftItems, leftTitle, leftCursor := m.getLeftPane()
	rightItems, rightTitle, rightRawText := m.getRightPane()

	leftActive := !m.focusRight && (m.depth != DepthTaskDetails)
	rightActive := m.focusRight || (m.depth == DepthTaskDetails)
	leftPane := renderPane(leftItems, leftTitle, "", leftCursor, 0, paneWidthLeft, paneHeight, leftActive)
	rightPane := renderPane(rightItems, rightTitle, rightRawText, -1, m.taskScrollOffset, paneWidthRight, paneHeight, rightActive)

	var centerContent string

	if m.state == stateResetConfirm {
		modalWidth := 60
		if modalWidth > safeTextWidth {
			modalWidth = safeTextWidth
		}

		content := lipgloss.NewStyle().
			Width(modalWidth).
			Align(lipgloss.Center).
			Render(
				lipgloss.JoinVertical(lipgloss.Center,
					lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Bold(true).Render("⚠️  WARNING: DATABASE RESET"),
					"\nThis will delete ALL locally cached tasks and settings.",
					"The application will need to re-sync everything.",
					"\nProceed? (y/n)",
				),
			)

		modalStyle := lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("#FF0000")).
			Padding(1, 4)

		centerContent = lipgloss.Place(m.width, paneHeight, lipgloss.Center, lipgloss.Center, modalStyle.Render(content))

	} else if m.showJSONPopup {
		modalWidth := safeTextWidth - 4

		headerText := "[ SHIFT+J: Close | SHIFT+S: Copy ]"
		copiedStatus := ""
		if m.jsonCopied {
			copiedStatus = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#00FF00")).
				Bold(true).
				Render(" [ Copied to Clipboard! ]")
		}

		modalHeader := lipgloss.JoinHorizontal(lipgloss.Top, headerText, copiedStatus)

		rawJSON := m.getCurrentJSON()
		jsonLines := strings.Split(rawJSON, "\n")

		usableHeight := paneHeight - 3
		if usableHeight < 1 {
			usableHeight = 1
		}

		maxScroll := len(jsonLines) - usableHeight
		if maxScroll < 0 {
			maxScroll = 0
		}
		if m.jsonScrollOffset > maxScroll {
			m.jsonScrollOffset = maxScroll
		}

		endIdx := m.jsonScrollOffset + usableHeight
		if endIdx > len(jsonLines) {
			endIdx = len(jsonLines)
		}

		var visibleJSON string
		if len(jsonLines) > 0 && m.jsonScrollOffset <= endIdx {
			visibleJSON = strings.Join(jsonLines[m.jsonScrollOffset:endIdx], "\n")
		}

		modalStyle := lipgloss.NewStyle().
			Width(modalWidth).
			Height(paneHeight-2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#9D4EDD")).
			Padding(0, 1)

		centerContent = lipgloss.JoinVertical(lipgloss.Center,
			lipgloss.NewStyle().Width(modalWidth).Align(lipgloss.Center).Render(modalHeader),
			modalStyle.Render(visibleJSON),
		)

	} else {
		splitPanes := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
		centerContent = splitPanes
	}

	return baseStyle.Render(lipgloss.JoinVertical(lipgloss.Left,
		topStack,
		centerContent,
		bottomStack,
	))
}
