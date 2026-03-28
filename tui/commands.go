package tui

import (
	"fmt"
	"super-duper-fortnight/clkup"
	"super-duper-fortnight/dbstore"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/sync/errgroup"
)

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

func fetchInitDataCmd(client *clkup.APIClient, db *dbstore.DB, forceFetch bool) tea.Cmd {
	return func() tea.Msg {
		if !forceFetch {
			cachedUser := db.GetUser()
			cachedWorkspaces := db.GetWorkspaces()

			if cachedUser != nil && len(cachedWorkspaces) > 0 {
				return InitDataMsg{
					User:       *cachedUser,
					Workspaces: cachedWorkspaces,
				}
			}
		}

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
			return ErrMsg{fmt.Errorf("failed to fetch user: %w", err)}
		}

		for attempts := 0; attempts < 3; attempts++ {
			workspaces, err = client.GetAuthorizedWorkspaces()
			if err == nil {
				break
			}
			time.Sleep(1 * time.Second)
		}
		if err != nil {
			return ErrMsg{fmt.Errorf("workspace API error: %w", err)}
		}

		db.SaveUser(user)
		db.SaveWorkspaces(workspaces)

		return InitDataMsg{
			User:       user,
			Workspaces: workspaces,
		}
	}
}

func fetchHierarchyCmd(client *clkup.APIClient, db *dbstore.DB, teamID string) tea.Cmd {
	return func() tea.Msg {
		start := time.Now()
		var g errgroup.Group
		var mu sync.Mutex

		var finalSpaces []clkup.Space
		var finalTasks []clkup.Task
		var finalFolders []clkup.Folder
		var finalLists []clkup.List
		var finalCustomFields []clkup.CustomField

		g.Go(func() error {
			tasks, err := client.GetAllTasks(teamID)
			if err == nil {
				mu.Lock()
				finalTasks = tasks
				mu.Unlock()
			}
			return err
		})

		g.Go(func() error {
			fields, err := client.GetWorkspaceCustomFields(teamID)
			if err == nil {
				mu.Lock()
				finalCustomFields = append(finalCustomFields, fields...)
				mu.Unlock()
			}
			return nil
		})

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

				g.Go(func() error {
					fields, err := client.GetSpaceCustomFields(sID)
					if err == nil {
						mu.Lock()
						finalCustomFields = append(finalCustomFields, fields...)
						mu.Unlock()
					}
					return nil
				})

				g.Go(func() error {
					folders, err := client.GetFolders(sID)
					if err != nil {
						return err
					}

					mu.Lock()
					finalFolders = append(finalFolders, folders...)
					mu.Unlock()

					for _, folder := range folders {
						fID := string(folder.ID)

						g.Go(func() error {
							fields, err := client.GetFolderCustomFields(fID)
							if err == nil {
								mu.Lock()
								finalCustomFields = append(finalCustomFields, fields...)
								mu.Unlock()
							}
							return nil
						})

						g.Go(func() error {
							lists, err := client.GetLists(fID)
							if err != nil {
								return err
							}

							mu.Lock()
							finalLists = append(finalLists, lists...)
							mu.Unlock()

							for _, list := range lists {
								lID := string(list.ID)
								g.Go(func() error {
									fields, err := client.GetListCustomFields(lID)
									if err == nil {
										mu.Lock()
										finalCustomFields = append(finalCustomFields, fields...)
										mu.Unlock()
									}
									return nil
								})
							}

							return nil
						})
					}
					return nil
				})

				g.Go(func() error {
					folderlessLists, err := client.GetFolderlessLists(sID)
					if err != nil {
						return err
					}

					mu.Lock()
					finalLists = append(finalLists, folderlessLists...)
					mu.Unlock()

					for _, list := range folderlessLists {
						lID := string(list.ID)
						g.Go(func() error {
							fields, err := client.GetListCustomFields(lID)
							if err == nil {
								mu.Lock()
								finalCustomFields = append(finalCustomFields, fields...)
								mu.Unlock()
							}
							return nil
						})
					}

					return nil
				})
			}
			return nil
		})

		if err := g.Wait(); err != nil {
			return ErrMsg{err}
		}

		err := db.SyncWorkspaceData(teamID, finalSpaces, finalFolders, finalLists, finalTasks)
		if err != nil {
			return ErrMsg{fmt.Errorf("database sync failed: %w", err)}
		}

		err = db.SyncCustomFields(finalCustomFields)
		if err != nil {
			fmt.Printf("Warning: Custom Field sync failed: %v\n", err)
		}

		perf := clkup.CalculatePerformance(len(finalTasks), start)

		return FanOutCompleteMsg{
			TeamID:      teamID,
			Performance: perf,
		}
	}
}

func tickAutoSync(d time.Duration) tea.Cmd {
	if d == 0 {
		return nil
	}
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return autoSyncTickMsg(t)
	})
}
