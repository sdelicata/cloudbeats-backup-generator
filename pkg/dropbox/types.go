package dropbox

// Account represents the response from /users/get_current_account.
type Account struct {
	AccountID string `json:"account_id"`
}

// ListFolderResponse represents the response from /files/list_folder.
type ListFolderResponse struct {
	Entries []Entry `json:"entries"`
	Cursor  string  `json:"cursor"`
	HasMore bool    `json:"has_more"`
}

// Entry represents a file or folder entry from Dropbox.
type Entry struct {
	Tag         string `json:".tag"`
	ID          string `json:"id"`
	Name        string `json:"name"`
	PathLower   string `json:"path_lower"`
	PathDisplay string `json:"path_display"`
}
