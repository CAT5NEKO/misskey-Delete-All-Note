package main

func (c *Client) UnpinNote(noteId string) error {
	return c.Post("i/unpin", map[string]interface{}{"noteId": noteId}, nil)
}

func (c *Client) DeleteNote(noteId string) error {
	return c.Post("notes/delete", map[string]interface{}{"noteId": noteId}, nil)
}

func (c *Client) FetchNotes(userId, untilId string) ([]Note, error) {
	args := map[string]interface{}{
		"userId": userId,
		"limit":  100,
		//"withChannelNotes": true,
		//"withReplies":      true,
		//"localOnly":        true,
		//"isSensitive":      true,
		//"isHidden":         true,
		//"allowPartial":     true,
	}
	if untilId != "" {
		args["untilId"] = untilId
	}

	var notes []Note
	err := c.Post("users/notes", args, &notes)
	return notes, err
}
