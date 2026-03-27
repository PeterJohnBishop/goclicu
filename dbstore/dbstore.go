package dbstore

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"super-duper-fortnight/clkup"

	_ "modernc.org/sqlite"
)

type DB struct {
	*sql.DB
}

func InitDB(filepath string) (*DB, error) {
	db, err := sql.Open("sqlite", filepath)
	if err != nil {
		return nil, err
	}

	schema := `
	CREATE TABLE IF NOT EXISTS config (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS workspaces (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		raw_data TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS spaces (
		id TEXT PRIMARY KEY,
		team_id TEXT NOT NULL,
		name TEXT NOT NULL,
		raw_data TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS folders (
		id TEXT PRIMARY KEY,
		space_id TEXT NOT NULL,
		name TEXT NOT NULL,
		raw_data TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS lists (
		id TEXT PRIMARY KEY,
		folder_id TEXT, 
		space_id TEXT NOT NULL,
		name TEXT NOT NULL,
		raw_data TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS tasks (
		id TEXT PRIMARY KEY,
		list_id TEXT NOT NULL,
		name TEXT NOT NULL,
		status TEXT NOT NULL,
		raw_data TEXT NOT NULL
	);

	-- Indexes for lightning-fast TUI navigation
	CREATE INDEX IF NOT EXISTS idx_spaces_team ON spaces(team_id);
	CREATE INDEX IF NOT EXISTS idx_folders_space ON folders(space_id);
	CREATE INDEX IF NOT EXISTS idx_lists_folder ON lists(folder_id);
	CREATE INDEX IF NOT EXISTS idx_lists_space ON lists(space_id);
	CREATE INDEX IF NOT EXISTS idx_tasks_list ON tasks(list_id);
	`

	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return &DB{db}, nil
}

func (db *DB) SaveToken(token string) error {
	query := `
		INSERT INTO config (key, value) 
		VALUES ('clickup_token', ?) 
		ON CONFLICT(key) DO UPDATE SET value=excluded.value;
	`
	_, err := db.Exec(query, token)
	return err
}

func (db *DB) GetToken() string {
	var token string
	db.QueryRow(`SELECT value FROM config WHERE key = 'clickup_token'`).Scan(&token)
	return token
}

func (db *DB) SyncWorkspaceData(teamID string, spaces []clkup.Space, folders []clkup.Folder, lists []clkup.List, tasks []clkup.Task) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Upsert Spaces
	spaceStmt, _ := tx.Prepare(`INSERT INTO spaces (id, team_id, name, raw_data) VALUES (?, ?, ?, ?) ON CONFLICT(id) DO UPDATE SET name=excluded.name, raw_data=excluded.raw_data;`)
	defer spaceStmt.Close()
	for _, s := range spaces {
		b, _ := json.Marshal(s)
		spaceStmt.Exec(string(s.ID), teamID, s.Name, string(b))
	}

	// Upsert Folders
	folderStmt, _ := tx.Prepare(`INSERT INTO folders (id, space_id, name, raw_data) VALUES (?, ?, ?, ?) ON CONFLICT(id) DO UPDATE SET name=excluded.name, raw_data=excluded.raw_data;`)
	defer folderStmt.Close()
	for _, f := range folders {
		b, _ := json.Marshal(f)
		folderStmt.Exec(string(f.ID), string(f.Space.Id), f.Name, string(b))
	}

	// Upsert Lists
	listStmt, _ := tx.Prepare(`INSERT INTO lists (id, folder_id, space_id, name, raw_data) VALUES (?, ?, ?, ?, ?) ON CONFLICT(id) DO UPDATE SET folder_id=excluded.folder_id, name=excluded.name, raw_data=excluded.raw_data;`)
	defer listStmt.Close()
	for _, l := range lists {
		b, _ := json.Marshal(l)

		// can be empty if it's a folderless list
		folderID := ""
		if l.Folder.Id != "" {
			folderID = string(l.Folder.Id)
		}
		listStmt.Exec(string(l.ID), folderID, string(l.Space.Id), l.Name, string(b))
	}

	// Upsert Tasks
	taskStmt, _ := tx.Prepare(`INSERT INTO tasks (id, list_id, name, status, raw_data) VALUES (?, ?, ?, ?, ?) ON CONFLICT(id) DO UPDATE SET list_id=excluded.list_id, name=excluded.name, status=excluded.status, raw_data=excluded.raw_data;`)
	defer taskStmt.Close()
	for _, t := range tasks {
		b, _ := json.Marshal(t)

		listID := string(t.List.Id)
		if listID == "" {
			listID = getListIDFromTask(t)
		}

		taskStmt.Exec(string(t.Id), listID, t.Name, t.Status.Status, string(b))
	}

	return tx.Commit()
}

func getListIDFromTask(t clkup.Task) string {
	b, err := json.Marshal(t)
	if err != nil {
		return ""
	}
	var temp map[string]interface{}
	json.Unmarshal(b, &temp)

	if listObj, ok := temp["list"].(map[string]interface{}); ok {
		if id, ok := listObj["id"].(string); ok {
			return id
		}
	}
	return ""
}

func (db *DB) GetSpaces(teamID string) []clkup.Space {
	rows, _ := db.Query(`SELECT raw_data FROM spaces WHERE team_id = ? ORDER BY name`, teamID)
	defer rows.Close()
	var res []clkup.Space
	for rows.Next() {
		var raw string
		rows.Scan(&raw)
		var item clkup.Space
		json.Unmarshal([]byte(raw), &item)
		res = append(res, item)
	}
	return res
}

func (db *DB) GetFolders(spaceID string) []clkup.Folder {
	rows, _ := db.Query(`SELECT raw_data FROM folders WHERE space_id = ? ORDER BY name`, spaceID)
	defer rows.Close()
	var res []clkup.Folder
	for rows.Next() {
		var raw string
		rows.Scan(&raw)
		var item clkup.Folder
		json.Unmarshal([]byte(raw), &item)
		res = append(res, item)
	}
	return res
}

func (db *DB) GetFolderlessLists(spaceID string) []clkup.List {
	rows, _ := db.Query(`SELECT raw_data FROM lists WHERE space_id = ? AND folder_id = '' ORDER BY name`, spaceID)
	defer rows.Close()
	var res []clkup.List
	for rows.Next() {
		var raw string
		rows.Scan(&raw)
		var item clkup.List
		json.Unmarshal([]byte(raw), &item)
		res = append(res, item)
	}
	return res
}

func (db *DB) GetListsByFolder(folderID string) []clkup.List {
	rows, _ := db.Query(`SELECT raw_data FROM lists WHERE folder_id = ? ORDER BY name`, folderID)
	defer rows.Close()
	var res []clkup.List
	for rows.Next() {
		var raw string
		rows.Scan(&raw)
		var item clkup.List
		json.Unmarshal([]byte(raw), &item)
		res = append(res, item)
	}
	return res
}

func (db *DB) GetTasksByList(listID string) []clkup.Task {
	rows, _ := db.Query(`SELECT raw_data FROM tasks WHERE list_id = ? ORDER BY name`, listID)
	defer rows.Close()
	var res []clkup.Task
	for rows.Next() {
		var raw string
		rows.Scan(&raw)
		var item clkup.Task
		json.Unmarshal([]byte(raw), &item)
		res = append(res, item)
	}
	return res
}
