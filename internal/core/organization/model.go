package organization

import "time"

type Organization struct {
	Id          string    `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Slug        string    `json:"slug" db:"slug"`
	OwnerId     string    `json:"owner_id" db:"owner_id"`
	Description string    `json:"description" db:"description"`
	ProfileUrl  string    `json:"profile_url" db:"profile_url"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}
