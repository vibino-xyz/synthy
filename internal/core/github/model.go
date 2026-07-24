package github

import "time"

const IDPrefix = "ghi" // github_installation ids

// Installation records that an organization has installed the Vibino GitHub
// App, and which GitHub account (org or user) it was installed on.
type Installation struct {
	Id             string    `json:"id" db:"id"`
	OrganizationId string    `json:"organization_id" db:"organization_id"`
	InstallationId int64     `json:"installation_id" db:"installation_id"`
	AccountLogin   string    `json:"account_login" db:"account_login"`
	AccountType    string    `json:"account_type" db:"account_type"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}
