package clkup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"

	"sync/atomic"
	"time"
)

func CalculatePerformance(totalTasks int, start time.Time) Performance {
	elapsed := time.Since(start)
	pages := float64(totalTasks) / 100.0
	if pages < 1 {
		pages = 1
	}

	rpm := (pages / elapsed.Minutes())
	tps := float64(totalTasks) / elapsed.Seconds()

	return Performance{
		Duration: elapsed.Round(time.Millisecond).String(),
		RPM:      fmt.Sprintf("%.2f", rpm),
		TPS:      fmt.Sprintf("%.2f", tps),
	}
}

// OAUTH

var (
	ClientID     string
	ClientSecret string
	RedirectURI  string
)

func GetAccessToken(code string) (string, error) {
	if ClientID == "" {
		ClientID = os.Getenv("CLIENT_ID")
	}
	if ClientSecret == "" {
		ClientSecret = os.Getenv("CLIENT_SECRET")
	}

	url := "https://api.clickup.com/api/v2/oauth/token"

	reqData := map[string]string{
		"client_id":     ClientID,
		"client_secret": ClientSecret,
		"code":          code,
	}

	jsonData, err := json.Marshal(reqData)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Add("accept", "application/json")
	req.Header.Add("content-type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	var tokenResponse struct {
		AccessToken string `json:"access_token"`
	}

	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return "", fmt.Errorf("failed to parse token response: %w", err)
	}

	return tokenResponse.AccessToken, nil
}

// API methods

func (c *APIClient) GetAuthorizedUser() (User, error) {
	url := "https://api.clickup.com/api/v2/user"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return User{}, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return User{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return User{}, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var userResponse UserResponse
	if err := json.NewDecoder(resp.Body).Decode(&userResponse); err != nil {
		return User{}, fmt.Errorf("failed to decode JSON: %w", err)
	}

	return userResponse.User, nil
}

func (c *APIClient) GetWorkspaceCustomFields(team_id string) ([]CustomField, error) {
	url := fmt.Sprintf("https://api.clickup.com/api/v2/team/%s/field", team_id)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var cfResp cfResponse
	if err := json.NewDecoder(resp.Body).Decode(&cfResp); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	return cfResp.Fields, nil
}

func (c *APIClient) GetSpaceCustomFields(space_id string) ([]CustomField, error) {
	url := fmt.Sprintf("https://api.clickup.com/api/v2/space/%s/field", space_id)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var cfResp cfResponse
	if err := json.NewDecoder(resp.Body).Decode(&cfResp); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	return cfResp.Fields, nil
}

func (c *APIClient) GetFolderCustomFields(folder_id string) ([]CustomField, error) {
	url := fmt.Sprintf("https://api.clickup.com/api/v2/folder/%s/field", folder_id)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var cfResp cfResponse
	if err := json.NewDecoder(resp.Body).Decode(&cfResp); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	return cfResp.Fields, nil
}

func (c *APIClient) GetListCustomFields(list_id string) ([]CustomField, error) {
	url := fmt.Sprintf("https://api.clickup.com/api/v2/list/%s/field", list_id)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var cfResponse cfResponse
	if err := json.NewDecoder(resp.Body).Decode(&cfResponse); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	return cfResponse.Fields, nil
}

func (c *APIClient) GetPlan(teamID string) (PlanResponse, error) {
	url := fmt.Sprintf("https://api.clickup.com/api/v2/team/%s/plan", teamID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return PlanResponse{}, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return PlanResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return PlanResponse{}, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var planResponse PlanResponse
	if err := json.NewDecoder(resp.Body).Decode(&planResponse); err != nil {
		return PlanResponse{}, fmt.Errorf("failed to decode JSON: %w", err)
	}

	return planResponse, nil
}

func (c *APIClient) GetAuthorizedWorkspaces() ([]Workspace, error) {
	url := "https://api.clickup.com/api/v2/team"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	// bodyBytes, err := io.ReadAll(resp.Body)
	// if err != nil {
	// 	return nil, err
	// }

	// os.WriteFile("clickup_debug.json", bodyBytes, 0644)

	var teamsResponse TeamsResponse
	if err := json.NewDecoder(resp.Body).Decode(&teamsResponse); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	return teamsResponse.Teams, nil
}

func (c *APIClient) GetSpaces(teamID string) ([]Space, error) {
	url := fmt.Sprintf("https://api.clickup.com/api/v2/team/%s/space", teamID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var spaceResponse SpacesResponse
	if err := json.NewDecoder(resp.Body).Decode(&spaceResponse); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}
	return spaceResponse.Spaces, nil
}

func (c *APIClient) GetFolders(spaceID string) ([]Folder, error) {
	url := fmt.Sprintf("https://api.clickup.com/api/v2/space/%s/folder", spaceID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var foldersResponse FoldersResponse
	if err := json.NewDecoder(resp.Body).Decode(&foldersResponse); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	return foldersResponse.Folders, nil
}

func (c *APIClient) GetFolderlessLists(spaceID string) ([]List, error) {
	url := fmt.Sprintf("https://api.clickup.com/api/v2/space/%s/list", spaceID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var listsResponse ListsResponse
	if err := json.NewDecoder(resp.Body).Decode(&listsResponse); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	return listsResponse.Lists, nil
}

func (c *APIClient) GetLists(folderID string) ([]List, error) {
	url := fmt.Sprintf("https://api.clickup.com/api/v2/folder/%s/list", folderID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var listsResponse ListsResponse
	if err := json.NewDecoder(resp.Body).Decode(&listsResponse); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	return listsResponse.Lists, nil
}

// concurrent task fetching

func (c *APIClient) GetAllTasks(teamID string) ([]Task, error) {
	var allTasks []Task
	var wg sync.WaitGroup
	var fetchErr error
	var errMu sync.Mutex

	taskChan := make(chan []Task, 100)
	sem := make(chan struct{}, 20)

	var done int32

	go func() {
		for tasks := range taskChan {
			allTasks = append(allTasks, tasks...)
		}
	}()

	// start := time.Now()

	for page := 0; ; page++ {
		if atomic.LoadInt32(&done) == 1 {
			break
		}

		sem <- struct{}{}
		wg.Add(1)

		go func(p int) {
			defer wg.Done()
			defer func() { <-sem }()

			tasks, err := c.fetchPage(teamID, p)

			if err != nil {
				errMu.Lock()
				fetchErr = err
				errMu.Unlock()
				atomic.StoreInt32(&done, 1)
				return
			}

			if len(tasks) == 0 {
				atomic.StoreInt32(&done, 1)
				return
			}

			taskChan <- tasks
		}(page)
	}

	wg.Wait()
	close(taskChan)
	time.Sleep(100 * time.Millisecond)

	if fetchErr != nil {
		return nil, fetchErr
	}

	// performance := calculatePerformance(len(allTasks), start)
	// fmt.Printf("fetched %d tasks in %s seconds. RPM: %s\n",
	// 	len(allTasks), performance.Duration, performance.RPM)

	return allTasks, nil
}

func (c *APIClient) fetchPage(teamID string, page int) ([]Task, error) {
	url := fmt.Sprintf("https://api.clickup.com/api/v2/team/%s/task?page=%d&include_closed=true&subtasks=true", teamID, page)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		return nil, fmt.Errorf("rate limit hit on page %d", page)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("api request failed with status: %d", resp.StatusCode)
	}

	var tasksResponse TasksResponse
	if err := json.NewDecoder(resp.Body).Decode(&tasksResponse); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	return tasksResponse.Task, nil
}

func (c *APIClient) UpdateTask(task_id string, body UpdateTaskBody) (Task, error) {
	url := fmt.Sprintf("https://api.clickup.com/api/v2/task/%s", task_id)

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return Task{}, fmt.Errorf("failed to marshal update body: %w", err)
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return Task{}, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.Do(req)
	if err != nil {
		return Task{}, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBytes, _ := io.ReadAll(resp.Body)
		return Task{}, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBytes))
	}

	var updatedTask Task
	if err := json.NewDecoder(resp.Body).Decode(&updatedTask); err != nil {
		return Task{}, fmt.Errorf("failed to decode response: %w", err)
	}

	return updatedTask, nil
}

func (c *APIClient) DeleteTask(task_id string) error {
	url := fmt.Sprintf("https://api.clickup.com/api/v2/task/%s", task_id)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBytes))
	}

	return nil
}

func (c *APIClient) SetCustomField(taskID, fieldID string, payload SetCustomFieldPayload) error {
	url := fmt.Sprintf("https://api.clickup.com/api/v2/task/%s/field/%s", taskID, fieldID)

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal custom field payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	// req.Header.Set("Authorization", c.APIKey)

	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBytes))
	}

	return nil
}
