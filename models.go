package main

import "time"

type User struct {
	Name        string `json:"name"`
	Username    string `json:"username"`
	NotesCount  int    `json:"notesCount"`
	Id          string `json:"id"`
	PinnedNotes []Note `json:"pinnedNotes"`
}

type Note struct {
	Id        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
}
