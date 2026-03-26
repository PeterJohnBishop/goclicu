# super-duper-fortnight

- On launch OAuth authentication is requried to generate a ClickUp API token, and define the Workspaces the Dashbord will have access to. 

- Your user data, authorized Workspaces, and the Workspace plans are requested. Plan data determines the target rate limit.

- Once a Workspace is selected, I use a concurrent fan-out approach in Go to fetch the Workspace data. I trigger two parallel streams: one drills down the Workspace hierarchy (Spaces to Folders to Lists) generating separate Goroutines for every request, while the other concurrently paginates through all of the task requests. Mutex locks prevent data overwrites on the final data stores.

(screenshot)[https://github.com/PeterJohnBishop/super-duper-fortnight/blob/main/Assets/filemanager.png?raw=true]!