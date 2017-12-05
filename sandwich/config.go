package sandwich

import (
	"github.com/sandwichcloud/deli-cli/api/client"
	"github.com/sandwichcloud/deli-cli/utils"
)

type Config struct {
	APIServer  string
	Username   string
	Password   string
	AuthMethod string
	ProjectID  string

	SandwichClient client.ClientInterface
}

func (c *Config) LoadAndValidate() error {

	c.SandwichClient = &client.SandwichClient{
		APIServer: &c.APIServer,
	}

	apiDiscover, err := c.SandwichClient.Auth().DiscoverAuth()
	if err != nil {
		return err
	}
	if c.AuthMethod == "" {
		c.AuthMethod = *apiDiscover.Default
	}

	token, err := utils.Login(c.SandwichClient.Auth(), c.Username, c.Password, c.AuthMethod, false)
	if err != nil {
		return err
	}

	c.SandwichClient.SetToken(token)
	if c.ProjectID != "" { // If project is given scope to the project
		project, err := c.SandwichClient.Project().Get(c.ProjectID)
		if err != nil {
			return err
		}
		token, err = c.SandwichClient.Auth().ScopeToken(project)
		if err != nil {
			return err
		}
		c.SandwichClient.SetToken(token)
	}
	return nil
}
