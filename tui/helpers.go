package tui

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"super-duper-fortnight/clkup"

	"github.com/charmbracelet/lipgloss"
)

func (m dashboardModel) getLeftPane() ([]ListItem, string, int) {
	switch m.depth {
	case DepthWorkspaces:
		var items []ListItem
		for _, w := range m.workspaces {
			items = append(items, ListItem{ID: string(w.ID), Name: w.Name, Type: "workspace"})
		}
		return items, "Workspaces", m.cursorWorkspace

	case DepthSpaces:
		var items []ListItem
		for _, s := range m.db.GetSpaces(m.activeTeamID) {
			items = append(items, ListItem{ID: string(s.ID), Name: s.Name, Type: "space"})
		}
		return items, "Spaces", m.cursorSpace

	case DepthFolders:
		space := m.getActiveSpace()
		if space == nil {
			return nil, "Folders & Standalone Lists", 0
		}
		var items []ListItem
		for _, f := range m.db.GetFolders(string(space.ID)) {
			items = append(items, ListItem{ID: string(f.ID), Name: fmt.Sprintf("📁 %s", f.Name), Type: "folder"})
		}
		for _, l := range m.db.GetFolderlessLists(string(space.ID)) {
			items = append(items, ListItem{ID: string(l.ID), Name: fmt.Sprintf("📄 %s", l.Name), Type: "list"})
		}
		return items, "Folders & Standalone Lists", m.cursorFolder

	case DepthLists:
		folder := m.getActiveFolder()
		if folder == nil {
			return nil, "Lists", 0
		}
		var items []ListItem
		for _, l := range m.db.GetListsByFolder(string(folder.ID)) {
			items = append(items, ListItem{ID: string(l.ID), Name: fmt.Sprintf("📄 %s", l.Name), Type: "list"})
		}
		return items, "Lists", m.cursorList

	case DepthTasks, DepthTaskDetails:
		var items []ListItem
		for _, t := range m.db.GetTasksByList(m.getActiveListID()) {
			items = append(items, ListItem{ID: string(t.Id), Name: t.Name, Type: "task", Subtitle: t.Status.Status})
		}
		return items, "Tasks", m.cursorTask
	}
	return nil, "", 0
}

func (m dashboardModel) getRightPane() ([]ListItem, string, string) {
	switch m.depth {
	case DepthWorkspaces:
		if len(m.workspaces) > 0 {
			hoveredWS := string(m.workspaces[m.cursorWorkspace].ID)
			spaces := m.db.GetSpaces(hoveredWS)
			if len(spaces) > 0 {
				var items []ListItem
				for _, s := range spaces {
					items = append(items, ListItem{ID: string(s.ID), Name: s.Name, Type: "space"})
				}
				return items, "Spaces Preview", ""
			}
		}
		return nil, "Instructions", "\n  <-- Press Enter to fetch Workspace data."

	case DepthSpaces:
		space := m.getActiveSpace()
		if space != nil {
			var items []ListItem
			for _, f := range m.db.GetFolders(string(space.ID)) {
				items = append(items, ListItem{ID: string(f.ID), Name: fmt.Sprintf("📁 %s", f.Name), Type: "folder"})
			}
			for _, l := range m.db.GetFolderlessLists(string(space.ID)) {
				items = append(items, ListItem{ID: string(l.ID), Name: fmt.Sprintf("📄 %s", l.Name), Type: "list"})
			}
			return items, "Folders & Standalone Lists", ""
		}
		return nil, "", ""

	case DepthFolders:
		space := m.getActiveSpace()
		if space == nil {
			return nil, "", ""
		}
		folders := m.db.GetFolders(string(space.ID))
		if m.cursorFolder < len(folders) {
			var items []ListItem
			for _, l := range m.db.GetListsByFolder(string(folders[m.cursorFolder].ID)) {
				items = append(items, ListItem{ID: string(l.ID), Name: fmt.Sprintf("📄 %s", l.Name), Type: "list"})
			}
			return items, "Lists", ""
		} else {
			// Hovering over a folderless list
			idx := m.cursorFolder - len(folders)
			lists := m.db.GetFolderlessLists(string(space.ID))
			if idx >= 0 && idx < len(lists) {
				var items []ListItem
				for _, t := range m.db.GetTasksByList(string(lists[idx].ID)) {
					items = append(items, ListItem{ID: string(t.Id), Name: t.Name, Type: "task", Subtitle: t.Status.Status})
				}
				return items, "Tasks", ""
			}
		}
		return nil, "", ""

	case DepthLists:
		var items []ListItem
		for _, t := range m.db.GetTasksByList(m.getHoveredListID()) {
			items = append(items, ListItem{ID: string(t.Id), Name: t.Name, Type: "task", Subtitle: t.Status.Status})
		}
		return items, "Tasks", ""

	case DepthTasks, DepthTaskDetails:
		t := m.getHoveredTask()
		if t != nil {
			b, _ := json.MarshalIndent(t, "", "  ")
			return nil, "Task Details JSON", string(b)
		}
		return nil, "Task Details", "No task selected"
	}
	return nil, "", ""
}

func OpenBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	case "linux":
		cmd = "xdg-open"
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

func (m dashboardModel) getStats() (spaces, folders, lists, tasks string) {
	teamID := m.activeTeamID
	if m.depth == DepthWorkspaces && len(m.workspaces) > 0 {
		teamID = string(m.workspaces[m.cursorWorkspace].ID)
	}

	if teamID == "" {
		return "-", "-", "-", "-"
	}

	// Leveraging SQLite COUNT() for instant stats without unmarshaling JSON
	switch m.depth {
	case DepthWorkspaces:
		var sCount, fCount, lCount, tCount int
		m.db.QueryRow(`SELECT COUNT(*) FROM spaces WHERE team_id = ?`, teamID).Scan(&sCount)
		m.db.QueryRow(`SELECT COUNT(*) FROM folders WHERE space_id IN (SELECT id FROM spaces WHERE team_id = ?)`, teamID).Scan(&fCount)
		m.db.QueryRow(`SELECT COUNT(*) FROM lists WHERE space_id IN (SELECT id FROM spaces WHERE team_id = ?)`, teamID).Scan(&lCount)
		m.db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE list_id IN (SELECT id FROM lists WHERE space_id IN (SELECT id FROM spaces WHERE team_id = ?))`, teamID).Scan(&tCount)
		if sCount == 0 {
			return "-", "-", "-", "-"
		}
		return fmt.Sprint(sCount), fmt.Sprint(fCount), fmt.Sprint(lCount), fmt.Sprint(tCount)

	case DepthSpaces:
		space := m.getActiveSpace()
		if space == nil {
			return "0", "0", "0", "0"
		}
		sID := string(space.ID)

		var fCount, lCount, tCount int
		m.db.QueryRow(`SELECT COUNT(*) FROM folders WHERE space_id = ?`, sID).Scan(&fCount)
		m.db.QueryRow(`SELECT COUNT(*) FROM lists WHERE space_id = ?`, sID).Scan(&lCount)
		m.db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE list_id IN (SELECT id FROM lists WHERE space_id = ?)`, sID).Scan(&tCount)
		return "1", fmt.Sprint(fCount), fmt.Sprint(lCount), fmt.Sprint(tCount)

	case DepthFolders:
		folder := m.getActiveFolder()
		if folder != nil {
			var lCount, tCount int
			m.db.QueryRow(`SELECT COUNT(*) FROM lists WHERE folder_id = ?`, string(folder.ID)).Scan(&lCount)
			m.db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE list_id IN (SELECT id FROM lists WHERE folder_id = ?)`, string(folder.ID)).Scan(&tCount)
			return "-", "1", fmt.Sprint(lCount), fmt.Sprint(tCount)
		}
		listID := m.getActiveListID()
		if listID != "" {
			var tCount int
			m.db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE list_id = ?`, listID).Scan(&tCount)
			return "-", "-", "1", fmt.Sprint(tCount)
		}
		return "-", "-", "-", "-"

	case DepthLists:
		listID := m.getHoveredListID()
		if listID != "" {
			var tCount int
			m.db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE list_id = ?`, listID).Scan(&tCount)
			return "-", "-", "1", fmt.Sprint(tCount)
		}
		return "-", "-", "-", "-"

	case DepthTasks, DepthTaskDetails:
		listID := m.getActiveListID()
		if listID != "" {
			var tCount int
			m.db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE list_id = ?`, listID).Scan(&tCount)
			return "-", "-", "-", fmt.Sprint(tCount)
		}
		return "-", "-", "-", "-"
	}

	return "-", "-", "-", "-"
}

func (m dashboardModel) getActiveSpace() *clkup.Space {
	spaces := m.db.GetSpaces(m.activeTeamID)
	if m.cursorSpace >= 0 && m.cursorSpace < len(spaces) {
		return &spaces[m.cursorSpace]
	}
	return nil
}

func (m dashboardModel) getActiveFolder() *clkup.Folder {
	space := m.getActiveSpace()
	if space == nil {
		return nil
	}
	folders := m.db.GetFolders(string(space.ID))
	if m.cursorFolder >= 0 && m.cursorFolder < len(folders) {
		return &folders[m.cursorFolder]
	}
	return nil
}

func (m dashboardModel) getActiveListID() string {
	space := m.getActiveSpace()
	if space == nil {
		return ""
	}
	folders := m.db.GetFolders(string(space.ID))

	if m.cursorFolder < len(folders) {
		lists := m.db.GetListsByFolder(string(folders[m.cursorFolder].ID))
		if m.cursorList >= 0 && m.cursorList < len(lists) {
			return string(lists[m.cursorList].ID)
		}
	} else {
		idx := m.cursorFolder - len(folders)
		lists := m.db.GetFolderlessLists(string(space.ID))
		if idx >= 0 && idx < len(lists) {
			return string(lists[idx].ID)
		}
	}
	return ""
}

func (m dashboardModel) getHoveredListID() string {
	folder := m.getActiveFolder()
	if folder == nil {
		return ""
	}
	lists := m.db.GetListsByFolder(string(folder.ID))
	if m.cursorList >= 0 && m.cursorList < len(lists) {
		return string(lists[m.cursorList].ID)
	}
	return ""
}

func (m dashboardModel) getHoveredTask() *clkup.Task {
	tasks := m.db.GetTasksByList(m.getActiveListID())
	if m.cursorTask >= 0 && m.cursorTask < len(tasks) {
		return &tasks[m.cursorTask]
	}
	return nil
}

func (m dashboardModel) getBreadcrumbs(maxWidth int) string {
	if m.state != stateLoaded && m.state != stateIdle {
		return ""
	}
	crumbs := []string{}
	if len(m.workspaces) > 0 && m.cursorWorkspace < len(m.workspaces) {
		crumbs = append(crumbs, m.workspaces[m.cursorWorkspace].Name)
	}

	space := m.getActiveSpace()
	if space != nil {
		if m.depth >= DepthSpaces {
			crumbs = append(crumbs, space.Name)
		}

		folders := m.db.GetFolders(string(space.ID))
		if m.depth >= DepthFolders {
			if m.cursorFolder < len(folders) {
				crumbs = append(crumbs, folders[m.cursorFolder].Name)
			} else {
				idx := m.cursorFolder - len(folders)
				lists := m.db.GetFolderlessLists(string(space.ID))
				if idx >= 0 && idx < len(lists) {
					crumbs = append(crumbs, lists[idx].Name)
				}
			}
		}
		if m.depth >= DepthLists {
			if m.cursorFolder < len(folders) {
				lists := m.db.GetListsByFolder(string(folders[m.cursorFolder].ID))
				if m.cursorList >= 0 && m.cursorList < len(lists) {
					crumbs = append(crumbs, lists[m.cursorList].Name)
				}
			}
		}
		if m.depth >= DepthTasks {
			t := m.getHoveredTask()
			if t != nil {
				crumbs = append(crumbs, t.Name)
			}
		}
	}

	crumbStr := strings.Join(crumbs, " > ")
	if lipgloss.Width(crumbStr) > maxWidth {
		runes := []rune(crumbStr)
		crumbStr = "…" + string(runes[len(runes)-(maxWidth-1):])
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#9D4EDD")).Render(crumbStr)
}

func (m dashboardModel) getCurrentSelectionURL() string {
	if m.state != stateLoaded && m.state != stateIdle {
		return ""
	}
	teamID := m.activeTeamID

	switch m.depth {
	case DepthWorkspaces:
		if len(m.workspaces) > 0 && m.cursorWorkspace < len(m.workspaces) {
			return fmt.Sprintf("https://app.clickup.com/%s", m.workspaces[m.cursorWorkspace].ID)
		}
	case DepthSpaces:
		space := m.getActiveSpace()
		if space != nil {
			return fmt.Sprintf("https://app.clickup.com/%s/v/s/%s", teamID, space.ID)
		}
	case DepthFolders:
		space := m.getActiveSpace()
		if space != nil {
			folders := m.db.GetFolders(string(space.ID))
			if m.cursorFolder < len(folders) {
				return fmt.Sprintf("https://app.clickup.com/%s/v/f/%s", teamID, folders[m.cursorFolder].ID)
			} else {
				idx := m.cursorFolder - len(folders)
				lists := m.db.GetFolderlessLists(string(space.ID))
				if idx >= 0 && idx < len(lists) {
					return fmt.Sprintf("https://app.clickup.com/%s/v/l/li/%s", teamID, lists[idx].ID)
				}
			}
		}
	case DepthLists:
		lID := m.getHoveredListID()
		if lID != "" {
			return fmt.Sprintf("https://app.clickup.com/%s/v/l/li/%s", teamID, lID)
		}
	case DepthTasks, DepthTaskDetails:
		t := m.getHoveredTask()
		if t != nil {
			return fmt.Sprintf("https://app.clickup.com/t/%s", t.Id)
		}
	}
	return ""
}

func renderPane(items []ListItem, title string, rawText string, cursor int, scrollOffset int, totalWidth int, totalHeight int, isActive bool) string {
	innerW := totalWidth - 2
	if innerW < 5 {
		innerW = 5
	}

	innerH := totalHeight - 2
	if innerH < 3 {
		innerH = 3
	}

	paneStyle := lipgloss.NewStyle().
		Width(innerW).
		Height(innerH).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#5A189A"))

	if isActive {
		paneStyle = paneStyle.BorderForeground(lipgloss.Color("#E0AAFF"))
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7B2CBF"))

	var uiLines []string

	titleRunes := []rune(title)
	if len(titleRunes) > innerW {
		title = string(titleRunes[:innerW-1]) + "…"
	}
	uiLines = append(uiLines, titleStyle.Render(title))
	uiLines = append(uiLines, "") // Spacer line

	if rawText != "" {
		lines := strings.Split(rawText, "\n")
		maxLines := innerH - 2
		if maxLines <= 0 {
			maxLines = 1
		}

		startIdx := scrollOffset
		if startIdx > len(lines)-maxLines {
			startIdx = len(lines) - maxLines
		}
		if startIdx < 0 {
			startIdx = 0
		}

		endIdx := startIdx + maxLines
		if endIdx > len(lines) {
			endIdx = len(lines)
		}

		if startIdx < len(lines) {
			for _, line := range lines[startIdx:endIdx] {
				line = strings.ReplaceAll(line, "\t", "  ")

				runes := []rune(line)
				if len(runes) > innerW {
					uiLines = append(uiLines, string(runes[:innerW-1])+"…")
				} else {
					uiLines = append(uiLines, line)
				}
			}
		}

	} else {
		maxLines := innerH - 2
		if maxLines <= 0 {
			maxLines = 1
		}
		startIdx := 0
		if cursor >= maxLines {
			startIdx = cursor - maxLines + 1
		}
		for i := startIdx; i < len(items) && i < startIdx+maxLines; i++ {
			item := items[i]
			prefix := "  "
			style := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

			if i == cursor && isActive {
				prefix = "> "
				style = style.Foreground(lipgloss.Color("#E0AAFF")).Bold(true)
			} else if i == cursor && !isActive {
				style = style.Foreground(lipgloss.Color("#9D4EDD"))
			}

			nameStr := item.Name
			if item.Subtitle != "" {
				nameStr += " [" + item.Subtitle + "]"
			}

			maxNameW := innerW - 2
			if maxNameW < 3 {
				maxNameW = 3
			}

			runes := []rune(nameStr)
			if len(runes) > maxNameW {
				nameStr = string(runes[:maxNameW-1]) + "…"
			}

			uiLines = append(uiLines, prefix+style.Render(nameStr))
		}
	}

	content := strings.Join(uiLines, "\n")
	return paneStyle.Render(content)
}
