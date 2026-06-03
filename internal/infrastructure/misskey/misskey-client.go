package misskey

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"misskeyNotedel/internal/domain/model"
	"misskeyNotedel/internal/domain/repository"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type MisskeyClient struct {
	Token    string
	Host     string
	Endpoint string
	HTTP     *http.Client
}

func NewMisskeyClient() (*MisskeyClient, error) {
	_ = godotenv.Load()

	token := os.Getenv("TOKEN")
	host := os.Getenv("HOST")
	if token == "" || host == "" {
		return nil, fmt.Errorf("TOKEN or HOST not set in .env")
	}

	return &MisskeyClient{
		Token:    token,
		Host:     host,
		Endpoint: fmt.Sprintf("https://%s/api/", host),
		HTTP:     &http.Client{},
	}, nil
}

func (c *MisskeyClient) post(api string, args map[string]interface{}, result interface{}) error {
	args["i"] = c.Token
	body, err := json.Marshal(args)
	if err != nil {
		return err
	}

	resp, err := c.HTTP.Post(c.Endpoint+api, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyText := strings.TrimSpace(string(bodyBytes))
		if bodyText == "" {
			return fmt.Errorf("HTTP %d returned from %s", resp.StatusCode, api)
		}
		return fmt.Errorf("HTTP %d returned from %s: %s", resp.StatusCode, api, bodyText)
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

func (c *MisskeyClient) FetchUser() (*model.User, error) {
	var user model.User
	err := c.post("i", map[string]interface{}{}, &user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (c *MisskeyClient) FetchNotes(userID model.UserID, untilID model.NoteID, opts repository.FetchNotesOptions) ([]model.Note, error) {
	args := map[string]interface{}{
		"userId": userID,
		"limit":  100,
	}
	if opts.WithReplies {
		args["withReplies"] = true
	}
	if opts.WithChannelNotes {
		args["withChannelNotes"] = true
	}
	if untilID != "" {
		args["untilId"] = untilID
	}

	var notes []model.Note
	err := c.post("users/notes", args, &notes)
	return notes, err
}

func (c *MisskeyClient) DeleteNote(noteID model.NoteID) error {
	return c.post("notes/delete", map[string]interface{}{"noteId": noteID}, nil)
}

func (c *MisskeyClient) UnpinNote(noteID model.NoteID) error {
	return c.post("i/unpin", map[string]interface{}{"noteId": noteID}, nil)
}

func (c *MisskeyClient) FetchDriveFiles(folderID *model.DriveFolderID, untilID model.DriveFileID) ([]model.DriveFile, error) {
	args := map[string]interface{}{
		"limit": 100,
		"sort":  "-createdAt",
	}
	if folderID != nil {
		args["folderId"] = *folderID
	}
	if untilID != "" {
		args["untilId"] = untilID
	}

	var files []model.DriveFile
	err := c.post("drive/files", args, &files)
	return files, err
}

func (c *MisskeyClient) FetchDriveFolders(parentID *model.DriveFolderID, untilID model.DriveFolderID) ([]model.DriveFolder, error) {
	args := map[string]interface{}{
		"limit": 100,
	}
	if parentID != nil {
		args["folderId"] = *parentID
	}
	if untilID != "" {
		args["untilId"] = untilID
	}

	var folders []model.DriveFolder
	err := c.post("drive/folders", args, &folders)
	return folders, err
}

func (c *MisskeyClient) DeleteDriveFile(fileID model.DriveFileID) error {
	return c.post("drive/files/delete", map[string]interface{}{"fileId": fileID}, nil)
}

func (c *MisskeyClient) DriveFileHasAttachedNotes(fileID model.DriveFileID) (bool, error) {
	args := map[string]interface{}{
		"fileId": fileID,
		"limit":  1,
	}

	var notes []model.Note
	if err := c.post("drive/files/attached-notes", args, &notes); err != nil {
		return false, err
	}
	return len(notes) > 0, nil
}
