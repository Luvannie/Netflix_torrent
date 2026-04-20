package settings

import "strconv"

type StorageProfile struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	BasePath string `json:"basePath"`
	Priority int    `json:"priority"`
	Active   bool   `json:"active"`
}

type CreateStorageProfileRequest struct {
	Name     string `json:"name"`
	BasePath string `json:"basePath"`
	Priority *int   `json:"priority"`
	Active   *bool  `json:"active"`
}

type UpdateStorageProfileRequest struct {
	Name     *string `json:"name"`
	BasePath *string `json:"basePath"`
	Priority *int    `json:"priority"`
	Active   *bool   `json:"active"`
}

type NotFoundError struct {
	ID int64
}

func (e NotFoundError) Error() string {
	return "Storage profile not found with ID: " + strconv.FormatInt(e.ID, 10)
}

type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return e.Message
}