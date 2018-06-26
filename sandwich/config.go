package sandwich

import (
	"errors"

	"github.com/sandwichcloud/deli-cli/api"
	"github.com/sandwichcloud/deli-cli/api/client"
	"golang.org/x/oauth2"
)

type Config struct {
	APIServer   string
	Token       string
	ProjectName string

	SandwichClient client.ClientInterface
}

func (c *Config) LoadAndValidate() error {

	c.SandwichClient = &client.SandwichClient{
		APIServer: &c.APIServer,
	}

	token := &oauth2.Token{
		AccessToken: c.Token,
		TokenType:   "Bearer",
	}

	c.SandwichClient.SetToken(token)
	if c.ProjectName != "" {
		_, err := c.SandwichClient.Project().Get(c.ProjectName)
		if err != nil {
			if apiError, ok := err.(api.APIErrorInterface); ok {
				if apiError.IsNotFound() {
					return errors.New("Configured project does not exist.")
				}
			}
			return err
		}
	}
	return nil
}
