package chat

// Should logs non-fatal errors
func (c *Chat) Should(value interface{}, err error) (interface{}, error) {
	if err != nil {
		c.errorLog().Print(err)
	}
	return value, err
}

// ShouldOK logs non-fatal errors
func (c *Chat) ShouldOK(err error) error {
	if err != nil {
		c.errorLog().Print(err)
	}
	return err
}
