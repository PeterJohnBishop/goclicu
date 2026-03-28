# super-duper-fortnight

- On launch OAuth authentication is requried to generate a ClickUp API token, and define the Workspaces the Dashbord will have access to. The token is saved to the SQLite database for local retrieval.

- (fetchInitDataCmd & fetchPlanCmd) Your user data, authorized Workspaces, and the Workspace plans are requested. Plan data determines the target rate limit. The user data and Workspace data is saved to the SQLite database for local retrieval.

- (fetchHierarchyCmd) Once a Workspace is selected, I use a concurrent fan-out approach in Go to fetch the Workspace data. I trigger two parallel streams: one drills down the Workspace hierarchy (Spaces to Folders to Lists) generating separate Goroutines for every request, while the other concurrently paginates through all of the task requests. Mutex locks prevent data overwrites on the final data stores. Once this process completes, all of the data is saved to the SQLite database for local retrieval. 

# key bindings

- Navigate with H J K L
- esc to go back
- space or enter to select
- o to open selection in ClickUp
- SHIFT+J to toggle raw JSON
- TAB switch focus between left and right panes
- r to re-sync data (at the Workspace level, re-sync the Workspace list. else, re-sync the Workspace data.) 
- SHIFT+F to cycle auto-sync OFF, 5m, 15m, or 30m
- q to quitchos

![screenshot](https://github.com/PeterJohnBishop/super-duper-fortnight/blob/main/Assets/task_example.png?raw=true)