package app

import (
	"encoding/base64"
	"encoding/json"
)

// paginationCursor is the cursor used for pagination.
type paginationCursor struct {
	Size int `json:"size"`
	Page int `json:"page"`
}

// PaginationCursor are the pagination information used for cursor based pagination.
type PaginationCursors struct {
	PrevCursor  string
	NextCursor  string
	HasNext     bool
	HasPrevious bool
}

func (c *paginationCursor) defaults() {
	if c.Size <= 0 {
		c.Size = 30
	}

	if c.Size > 100 {
		c.Size = 100
	}

	if c.Page <= 0 {
		c.Page = 1
	}
}

func paginationCursorToString(c paginationCursor) string {
	data, _ := json.Marshal(c)
	base64Data := base64.StdEncoding.EncodeToString(data)
	return base64Data
}

func paginationCursorFromString(encoded string) (*paginationCursor, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	var c paginationCursor
	err = json.Unmarshal(data, &c)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func paginateSlice[T any](items []T, cursorS string) (paginatedItems []T, cursors PaginationCursors) {
	cursor, err := paginationCursorFromString(cursorS)
	if err != nil {
		cursor = &paginationCursor{}
	}
	cursor.defaults()

	startIndex := (cursor.Page - 1) * cursor.Size
	if startIndex > len(items) {
		startIndex = len(items)
	}
	endIndex := startIndex + cursor.Size
	if endIndex > len(items) {
		endIndex = len(items)
	}
	paginatedItems = items[startIndex:endIndex]
	nextCursor := ""
	if endIndex < len(items) {
		nextCursor = paginationCursorToString(paginationCursor{
			Size: cursor.Size,
			Page: cursor.Page + 1,
		})
	}
	prevCursor := ""
	if cursor.Page > 1 {
		prevCursor = paginationCursorToString(paginationCursor{
			Size: cursor.Size,
			Page: cursor.Page - 1,
		})
	}

	return paginatedItems, PaginationCursors{
		PrevCursor:  prevCursor,
		NextCursor:  nextCursor,
		HasNext:     nextCursor != "",
		HasPrevious: prevCursor != "",
	}
}
