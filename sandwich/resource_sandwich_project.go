package sandwich

import (
	"fmt"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/sandwichcloud/deli-cli/api"
	"github.com/sandwichcloud/deli-cli/api/client"
)

func resourceProject() *schema.Resource {
	return &schema.Resource{
		Create: resourceProjectCreate,
		Read:   resourceProjectRead,
		Delete: resourceProjectDelete,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Read:   schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceProjectCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	projectClient := config.SandwichClient.Project()
	name := d.Get("name").(string)

	project, err := projectClient.Create(name)
	if err != nil {
		return err
	}

	d.Partial(true) // Things can still be created but error during a state change

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"CREATING"},
		Target:     []string{"CREATED"},
		Refresh:    ProjectRefreshFunc(projectClient, project.ID.String()),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}
	d.SetId(project.ID.String())
	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for project (%s) to become ready: %s", project.ID.String(), err)
	}
	d.Partial(false) // There was no error during a state change so we should be safe

	return resourceProjectRead(d, meta)
}

func resourceProjectRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	projectClient := config.SandwichClient.Project()

	project, err := projectClient.Get(d.Id())
	if err != nil {
		if apiError, ok := err.(api.APIErrorInterface); ok {
			if apiError.IsNotFound() {
				d.SetId("")
				return nil
			}
		}
		return err
	}

	d.Set("name", project.Name)

	return nil
}

func resourceProjectDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	projectClient := config.SandwichClient.Project()

	err := projectClient.Delete(d.Id())
	if err != nil {
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"DELETING"},
		Target:     []string{"DELETED"},
		Refresh:    ProjectRefreshFunc(projectClient, d.Id()),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}
	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for project (%s) to delete: %s", d.Id(), err)
	}

	d.SetId("")
	return nil
}

func ProjectRefreshFunc(projectClient client.ProjectClientInterface, projectID string) func() (result interface{}, state string, err error) {
	return func() (result interface{}, state string, err error) {
		project, err := projectClient.Get(projectID)
		if err != nil {
			if apiError, ok := err.(api.APIErrorInterface); ok {
				if apiError.IsNotFound() {
					return project, "DELETED", nil
				}
			}
			return nil, "", err
		}
		return project, project.State, nil
	}
}
