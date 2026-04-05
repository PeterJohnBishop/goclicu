package clkup

import (
	"bytes"
	"encoding/json"
)

// types generated through https://quicktype.io/ with minor modification for duplicate definitions

// response types

type FlexID string

// clickup IDs are inconsistently typed so this ensures the value always unmarshalls correctly
func (fid *FlexID) UnmarshalJSON(b []byte) error {
	if bytes.Equal(b, []byte("null")) {
		return nil
	}

	if len(b) > 0 && b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*fid = FlexID(s)
		return nil
	}

	*fid = FlexID(b)
	return nil
}

type UserResponse struct {
	User User `json:"user"`
}

type PlanResponse struct {
	PlanName string `json:"plan_name"`
	PlanID   int    `json:"plan_id"`
}

type TeamsResponse struct {
	Teams []Workspace `json:"teams"`
}

type SpacesResponse struct {
	Spaces []Space `json:"spaces"`
}

type FoldersResponse struct {
	Folders []Folder `json:"folders"`
}

type ListsResponse struct {
	Lists []List `json:"lists"`
}

type TasksResponse struct {
	Task []Task `json:"tasks"`
}

type cfResponse struct {
	Fields []CustomField `json:"fields"`
}

// hierarchy types

type Workspace struct {
	ID      FlexID   `json:"id"`
	Name    string   `json:"name"`
	Color   string   `json:"color"`
	Avatar  string   `json:"avatar"`
	Members []Member `json:"members"`
}

type Space struct {
	ID                FlexID        `json:"id"`
	Name              string        `json:"name"`
	Private           bool          `json:"private"`
	Color             string        `json:"color"`
	Avatar            string        `json:"avatar"`
	AdminCanManage    bool          `json:"admin_can_manage"`
	Archived          bool          `json:"archived"`
	Members           []SpaceMember `json:"members"`
	Statuses          []Status      `json:"statuses"`
	MultipleAssignees bool          `json:"multiple_assignees"`
	Features          Features      `json:"features"`
}

type Folder struct {
	ID               FlexID        `json:"id"`
	Name             string        `json:"name"`
	Orderindex       int64         `json:"orderindex"`
	OverrideStatuses bool          `json:"override_statuses"`
	Hidden           bool          `json:"hidden"`
	Space            SpaceLocation `json:"space"`
	TaskCount        string        `json:"task_count"`
	Lists            []interface{} `json:"lists"`
}

type List struct {
	ID               FlexID         `json:"id"`
	Name             string         `json:"name"`
	Orderindex       int64          `json:"orderindex"`
	Content          string         `json:"content"`
	Status           Status         `json:"status"`
	Priority         Priority       `json:"priority"`
	Assignee         FlexID         `json:"assignee"`
	TaskCount        int            `json:"task_count"`
	DueDate          string         `json:"due_date"`
	StartDate        string         `json:"start_date"`
	Folder           FolderLocation `json:"folder"`
	Space            SpaceLocation  `json:"space"`
	Archived         bool           `json:"archived"`
	OverrideStatuses bool           `json:"override_statuses"`
	PermissionLevel  string         `json:"permission_level"`
}

type Task struct {
	Points          any            `json:"points"`
	DateDone        any            `json:"date_done"`
	GroupAssignees  []any          `json:"group_assignees"`
	Id              FlexID         `json:"id"`
	Orderindex      string         `json:"orderindex"`
	Parent          any            `json:"parent"`
	Sharing         Sharing        `json:"sharing"`
	TextContent     string         `json:"text_content"`
	TopLevelParent  any            `json:"top_level_parent"`
	Assignees       []any          `json:"assignees"`
	Space           SpaceLocation  `json:"space"`
	StartDate       any            `json:"start_date"`
	Tags            []any          `json:"tags"`
	TeamId          FlexID         `json:"team_id"`
	Url             string         `json:"url"`
	Watchers        []Watcher      `json:"watchers"`
	CustomFields    []CustomField  `json:"custom_fields"`
	CustomItemId    int            `json:"custom_item_id"`
	LinkedTasks     []any          `json:"linked_tasks"`
	Project         Project        `json:"project"`
	Status          Status         `json:"status"`
	Checklists      []any          `json:"checklists"`
	DateUpdated     string         `json:"date_updated"`
	Dependencies    []any          `json:"dependencies"`
	Description     string         `json:"description"`
	List            ListLocation   `json:"list"`
	PermissionLevel string         `json:"permission_level"`
	Priority        any            `json:"priority"`
	TimeSpent       int            `json:"time_spent"`
	Name            string         `json:"name"`
	TimeEstimate    any            `json:"time_estimate"`
	Attachments     []any          `json:"attachments"`
	Creator         Creator        `json:"creator"`
	DateCreated     string         `json:"date_created"`
	Locations       []any          `json:"locations"`
	Archived        bool           `json:"archived"`
	DateClosed      any            `json:"date_closed"`
	CustomId        any            `json:"custom_id"`
	DueDate         any            `json:"due_date"`
	Folder          FolderLocation `json:"folder"`
}

// user types

type User struct {
	ID                FlexID `json:"id"`
	Username          string `json:"username"`
	Email             string `json:"email"`
	Color             string `json:"color"`
	ProfilePicture    string `json:"profilePicture"`
	Initials          string `json:"initials"`
	WeekStartDay      int64  `json:"week_start_day"`
	GlobalFontSupport bool   `json:"global_font_support"`
	Timezone          string `json:"timezone"`
}

type Member struct {
	User WorkspaceUser `json:"user"`
}

type WorkspaceUser struct {
	ID             FlexID `json:"id"`
	Username       string `json:"username"`
	Color          string `json:"color"`
	ProfilePicture string `json:"profilePicture"`
}

type SpaceMember struct {
	User SpaceUser `json:"user"`
}

type SpaceUser struct {
	ID             FlexID `json:"id"`
	Username       string `json:"username"`
	Color          string `json:"color"`
	ProfilePicture string `json:"profilePicture"`
	Initials       string `json:"initials"`
}

type Watcher struct {
	Id             FlexID `json:"id"`
	Initials       string `json:"initials"`
	ProfilePicture any    `json:"profilePicture"`
	Username       string `json:"username"`
	Color          string `json:"color"`
	Email          string `json:"email"`
}

// property types

type SpaceStatus struct {
	Status     string `json:"status"`
	Type       string `json:"type"`
	OrderIndex int    `json:"orderindex"`
	Color      string `json:"color"`
}

type Features struct {
	DueDates          DueDatesFeature `json:"due_dates"`
	TimeTracking      FeatureEnabled  `json:"time_tracking"`
	Tags              FeatureEnabled  `json:"tags"`
	TimeEstimates     FeatureEnabled  `json:"time_estimates"`
	Checklists        FeatureEnabled  `json:"checklists"`
	CustomFields      FeatureEnabled  `json:"custom_fields"`
	RemapDependencies FeatureEnabled  `json:"remap_dependencies"`
	DependencyWarning FeatureEnabled  `json:"dependency_warning"`
	Portfolios        FeatureEnabled  `json:"portfolios"`
}

type FeatureEnabled struct {
	Enabled bool `json:"enabled"`
}

type DueDatesFeature struct {
	Enabled            bool `json:"enabled"`
	StartDate          bool `json:"start_date"`
	RemapDueDates      bool `json:"remap_due_dates"`
	RemapClosedDueDate bool `json:"remap_closed_due_date"`
}

type SpaceLocation struct {
	Id FlexID `json:"id"`
}

type Status struct {
	Orderindex int    `json:"orderindex"`
	Status     string `json:"status"`
	Type       string `json:"type"`
	Color      string `json:"color"`
	Id         FlexID `json:"id"`
}

type ListLocation struct {
	Access bool   `json:"access"`
	Id     FlexID `json:"id"`
	Name   string `json:"name"`
}

type Priority struct {
	Priority string `json:"priority"`
	Color    string `json:"color"`
}

type Sharing struct {
	Public               bool     `json:"public"`
	PublicFields         []string `json:"public_fields"`
	PublicShareExpiresOn any      `json:"public_share_expires_on"`
	SeoOptimized         bool     `json:"seo_optimized"`
	Token                any      `json:"token"`
}

type CustomField struct {
	Name           string     `json:"name"`
	Required       bool       `json:"required"`
	Type           string     `json:"type"`
	TypeConfig     TypeConfig `json:"type_config"`
	DateCreated    string     `json:"date_created"`
	HideFromGuests bool       `json:"hide_from_guests"`
	Id             FlexID     `json:"id"`
	Value          any        `json:"value"`
}

type TypeConfig struct {
	Options []CustomFieldOption `json:"options"`
}

type CustomFieldOption struct {
	ID         string `json:"id"`
	Name       string `json:"name"`  // Used by drop_down
	Label      string `json:"label"` // Used by labels
	Color      string `json:"color"`
	OrderIndex any    `json:"orderindex"` // Can be int or string from ClickUp
}

type Project struct {
	Id     FlexID `json:"id"`
	Name   string `json:"name"`
	Access bool   `json:"access"`
	Hidden bool   `json:"hidden"`
}

type Creator struct {
	Color          string `json:"color"`
	Email          string `json:"email"`
	Id             FlexID `json:"id"`
	ProfilePicture any    `json:"profilePicture"`
	Username       string `json:"username"`
}

type FolderLocation struct {
	Name   string `json:"name"`
	Access bool   `json:"access"`
	Hidden bool   `json:"hidden"`
	Id     FlexID `json:"id"`
}

type Performance struct {
	Duration string
	RPM      string
	TPS      string
}

type UpdateTaskBody struct {
	CustomItemID    int64                `json:"custom_item_id,omitempty"`
	Name            string               `json:"name,omitempty"`
	Description     string               `json:"description,omitempty"`
	MarkdownContent string               `json:"markdown_content,omitempty"`
	Status          string               `json:"status,omitempty"`
	Priority        int64                `json:"priority,omitempty"`
	DueDate         int64                `json:"due_date,omitempty"`
	DueDateTime     bool                 `json:"due_date_time,omitempty"`
	Parent          string               `json:"parent,omitempty"`
	TimeEstimate    int64                `json:"time_estimate,omitempty"`
	StartDate       int64                `json:"start_date,omitempty"`
	StartDateTime   int64                `json:"start_date_time,omitempty"`
	Points          int64                `json:"points,omitempty"`
	Assignees       UpdateAssignees      `json:"assignees,omitempty"`
	GroupAssignees  UpdateGroupAssignees `json:"group_assignees,omitempty"`
	Watchers        UpdateAssignees      `json:"watchers,omitempty"`
	Archived        bool                 `json:"archived,omitempty"`
}

type UpdateAssignees struct {
	Add []int64 `json:"add,omitempty"`
	Rem []int64 `json:"rem,omitempty"`
}

type UpdateGroupAssignees struct {
	Add []string `json:"add,omitempty"`
	Rem []string `json:"rem,omitempty"`
}

type SetCustomFieldPayload struct {
	Value        any `json:"value"`
	ValueOptions any `json:"value_options,omitempty"`
}
