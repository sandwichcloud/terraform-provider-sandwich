package sandwich

import (
	"fmt"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/sandwichcloud/deli-cli/api"
)

func resourceProjectRole() *schema.Resource {
	return &schema.Resource{
		Create: resourceProjectRoleCreate,
		Read:   resourceProjectRoleRead,
		Update: resourceProjectRoleUpdate,
		Delete: resourceProjectRoleDelete,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Read:   schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"project_name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"permissions": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func resourceProjectRoleCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	projectName, err := getProject(d, config)
	if err != nil {
		return err
	}
	roleClient := config.SandwichClient.ProjectRole(projectName)

	name := d.Get("name").(string)
	d.Set("project_name", projectName)

	var permissions []string
	for _, policyInt := range d.Get("permissions").([]interface{}) {
		permissions = append(permissions, policyInt.(string))
	}

	role, err := roleClient.Create(name, permissions)
	if err != nil {
		return err
	}

	d.Partial(true) // Things can still be created but error during a state change
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"ToCreate", "Creating"},
		Target:     []string{"Created"},
		Refresh:    RoleRefreshFunc(roleClient, role.Name),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}
	d.SetId(role.Name)
	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for project role (%s) to become ready: %s", role.Name, err)
	}
	d.Partial(false) // There was no error during a state change so we should be safe

	return resourceProjectRoleUpdate(d, meta)
}

func resourceProjectRoleRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	roleClient := config.SandwichClient.ProjectRole(d.Get("project_name").(string))

	role, err := roleClient.Get(d.Id())
	if err != nil {
		if apiError, ok := err.(api.APIErrorInterface); ok {
			if apiError.IsNotFound() {
				d.SetId("")
				return nil
			}
		}
		return err
	}

	d.Set("permissions", role.Permissions)

	return nil
}

func resourceProjectRoleUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	roleClient := config.SandwichClient.ProjectRole(d.Get("project_name").(string))

	var policies []string
	for _, policyInt := range d.Get("policies").([]interface{}) {
		policies = append(policies, policyInt.(string))
	}

	err := roleClient.Update(d.Id(), policies)
	if err != nil {
		if apiError, ok := err.(api.APIErrorInterface); ok {
			if apiError.IsNotFound() {
				d.SetId("")
				return nil
			}
		}
		return err
	}

	return resourceProjectRoleRead(d, meta)
}

func resourceProjectRoleDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	roleClient := config.SandwichClient.ProjectRole(d.Get("project_name").(string))

	err := roleClient.Delete(d.Id())
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
		Refresh:    RoleRefreshFunc(roleClient, d.Id()),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}
	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for project role (%s) to delete: %s", d.Id(), err)
	}
	d.SetId("")
	return nil
}
