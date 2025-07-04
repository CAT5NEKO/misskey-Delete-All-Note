package main

func (c *Client) FetchUser() (*User, error) {
	var user User
	err := c.Post("i", map[string]interface{}{}, &user)
	return &user, err
}
