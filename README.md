# GOCLICU for ClickUp

# Install

brew install PeterJohnBishop/tap/goclicu
- run with 'goclicu'

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

Example of a task:

![screenshot](https://github.com/PeterJohnBishop/goclicu/blob/main/Assets/task_example.png?raw=true)

Example of the same task in raw JSON mode:

![screenshot](https://github.com/PeterJohnBishop/goclicu/blob/main/Assets/json_example.png?raw=true)

# in progress

- Some Custom Fields still default to RAW json, as I haven't added formatting for those types yet.
- Add indicator that a Workspace is accessable, but data hasn't been pulled.
- More dashboard cards/stats  
- graphs that adjust over hierarchy selection and/or time range (slider?) like DataDog
- Direct API Create, Update, and Delete options
- Attachment viewer/previews

Looks great with [Posting](https://posting.sh/). Just sayin.

![screenshot](https://github.com/PeterJohnBishop/CLI_ckUp/blob/main/Assets/with_posting.png?raw=true)

