
-- Workspaces (Teams)
CREATE TABLE IF NOT EXISTS workspaces (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    raw_data TEXT NOT NULL
);

-- Spaces
CREATE TABLE IF NOT EXISTS spaces (
    id TEXT PRIMARY KEY,
    team_id TEXT NOT NULL,
    name TEXT NOT NULL,
    raw_data TEXT NOT NULL
);

-- Folders
CREATE TABLE IF NOT EXISTS folders (
    id TEXT PRIMARY KEY,
    space_id TEXT NOT NULL,
    name TEXT NOT NULL,
    raw_data TEXT NOT NULL
);

-- Lists (Folder ID can be empty for folderless lists)
CREATE TABLE IF NOT EXISTS lists (
    id TEXT PRIMARY KEY,
    folder_id TEXT, 
    space_id TEXT NOT NULL,
    name TEXT NOT NULL,
    raw_data TEXT NOT NULL
);

-- Tasks
CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,
    list_id TEXT NOT NULL,
    name TEXT NOT NULL,
    status TEXT NOT NULL,
    raw_data TEXT NOT NULL
);

-- Indexes for TUI
CREATE INDEX IF NOT EXISTS idx_spaces_team ON spaces(team_id);
CREATE INDEX IF NOT EXISTS idx_folders_space ON folders(space_id);
CREATE INDEX IF NOT EXISTS idx_lists_folder ON lists(folder_id);
CREATE INDEX IF NOT EXISTS idx_lists_space ON lists(space_id);
CREATE INDEX IF NOT EXISTS idx_tasks_list ON tasks(list_id);