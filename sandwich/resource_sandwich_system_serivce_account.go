package sandwich

import (
	"fmt"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/sandwichcloud/deli-cli/api"
	"github.com/sandwichcloud/deli-cli/api/client"
)

func resourceSystemServiceAccount() *schema.Resource {
	return &schema.Resource{
		Create: resourceSystemServiceAccountCreate,
		Read:   resourceSystemServiceAccountRead,
		Delete: resourceSystemServiceAccountDelete,

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
		},
	}
}

func resourceSystemServiceAccountCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	serviceAccountClient := config.SandwichClient.SystemServiceAccount()

	name := d.Get("name").(string)

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
		return fmt.Errorf("Error waiting for system service account (%s) to become ready: %s", serviceAccount.Name, err)
	}
	d.Partial(false) // There was no error during a state change so we should be safe

	return resourceSystemServiceAccountRead(d, meta)
}

func resourceSystemServiceAccountRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	serviceAccountClient := config.SandwichClient.SystemServiceAccount()

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

func resourceSystemServiceAccountDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	serviceAccountClient := config.SandwichClient.SystemServiceAccount()

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
		return fmt.Errorf("Error waiting for system service account (%s) to delete: %s", d.Id(), err)
	}

	d.SetId("")
	return nil
}

func SerivceAccountRefreshFunc(serviceAccountClient client.ServiceAccountClientInterface, serviceAccountId string) func() (result interface{}, state string, err error) {
	return func() (result interface{}, state string, err error) {
		serviceAccount, err := serviceAccountClient.Get(serviceAccountId)
		if err != nil {
			if apiError, ok := err.(api.APIErrorInterface); ok {
				if apiError.IsNotFound() {
					return serviceAccount, "Deleted", nil
				}
			}
			return nil, "", err
		}
		return serviceAccount, serviceAccount.State, nil
	}
}
