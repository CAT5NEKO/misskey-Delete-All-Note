package main

import (
	"fmt"
	"log"
	"time"
)

func main() {
	client, err := NewClient()
	if err != nil {
		log.Fatalf("Failed to initialize client: %v", err)
	}

	user, err := client.FetchUser()
	if err != nil {
		log.Fatalf("Failed to fetch user info: %v", err)
	}

	fmt.Printf("User: %s @%s (%d Notes)\n", user.Name, user.Username, user.NotesCount)

	for _, note := range user.PinnedNotes {
		if err := client.UnpinNote(note.Id); err != nil {
			fmt.Printf("Failed to unpin note %s: %v\n", note.Id, err)
		} else {
			fmt.Printf("Unpinned note: %s\n", note.Id)
		}
	}

	var allNotes []Note
	untilId := ""
	for {
		batch, err := client.FetchNotes(user.Id, untilId)
		if err != nil {
			log.Printf("Error fetching notes: %v", err)
			break
		}
		if len(batch) == 0 {
			break
		}
		untilId = batch[len(batch)-1].Id
		allNotes = append(allNotes, batch...)
		fmt.Printf("Fetched %d notes...\n", len(allNotes))
	}

	for i, note := range allNotes {
		if err := client.DeleteNote(note.Id); err != nil {
			fmt.Printf("Error deleting note %d/%d (%s): %v\n", i+1, len(allNotes), note.Id, err)
			fmt.Println("Retrying in 15 minutes...")
			time.Sleep(15 * time.Minute)
			i--
		} else {
			fmt.Printf("Deleted note %d/%d\n", i+1, len(allNotes))
			time.Sleep(1 * time.Second)
		}
	}

	fmt.Println("All notes deleted.")
}
