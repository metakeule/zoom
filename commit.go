package zoom

import "fmt"

type CommitMessage struct {
	Host    string
	User    string
	App     string
	Version string
	Command string
	Details string
}

func (c CommitMessage) String() string {
	var triggered string

	if c.Host != "" {
		triggered = " on " + c.Host
	}

	if c.User != "" {
		triggered = fmt.Sprintf("triggered by %#v", c.User) + triggered
	}

	return fmt.Sprintf(
		"%s %s %s\nversion: %s\n%s\n",
		c.App,
		c.Command,
		triggered,
		c.Version,
		c.Details,
	)
}
