package sandwich

import (
	"fmt"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/sandwichcloud/deli-cli/api"
)

func resourceProjectServiceAccount() *schema.Resource {
	return &schema.Resource{
		Create: resourceProjectServiceAccountCreate,
		Read:   resourceProjectServiceAccountRead,
		Delete: resourceProjectServiceAccountDelete,

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
			"email": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"project_name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
		},
	}
}

func resourceProjectServiceAccountCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	projectName, err := getProject(d, config)
	if err != nil {
		return err
	}
	serviceAccountClient := config.SandwichClient.ProjectServiceAccount(projectName)

	name := d.Get("name").(string)
	d.Set("project_name", projectName)

	serviceAccount, err := serviceAccountClient.Create(name)
	if err != nil {
		return err
	}

	d.Set("email", serviceAccount.Email)

	d.Partial(true) // Things can still be created but error during a state change
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"ToCreate", "Creating"},
		Target:     []string{"Created"},
		Refresh:    SerivceAccountRefreshFunc(serviceAccountClient, serviceAccount.Name),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}
	d.SetId(serviceAccount.Name)
	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for project service account (%s) to become ready: %s", serviceAccount.Name, err)
	}
	d.Partial(false) // There was no error during a state change so we should be safe

	return resourceProjectServiceAccountRead(d, meta)
}

func resourceProjectServiceAccountRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	serviceAccountClient := config.SandwichClient.ProjectServiceAccount(d.Get("project_name").(string))

	serviceAccount, err := serviceAccountClient.Get(d.Id())
	if err != nil {
		if apiError, ok := err.(api.APIErrorInterface); ok {
			if apiError.IsNotFound() {
				d.SetId("")
				return nil
			}
		}
		return err
	}

	d.Set("name", serviceAccount.Name)

	return nil
}

func resourceProjectServiceAccountDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	serviceAccountClient := config.SandwichClient.ProjectServiceAccount(d.Get("project_name").(string))

	err := serviceAccountClient.Delete(d.Id())
	if err != nil {
		if apiError, ok := err.(api.APIErrorInterface); ok {
			if apiError.IsNotFound() {
				d.SetId("")
				return nil
			}
		}
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"ToDelete", "Deleting"},
		Target:     []string{"Deleted"},
		Refresh:    SerivceAccountRefreshFunc(serviceAccountClient, d.Id()),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}
	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for project service account (%s) to delete: %s", d.Id(), err)
	}

	d.SetId("")
	return nil
}
