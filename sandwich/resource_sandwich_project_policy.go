package sandwich

import (
	"time"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/sandwichcloud/deli-cli/api"
)

func resourceProjectPolicy() *schema.Resource {
	return &schema.Resource{
		Create: resourceProjectPolicyCreate,
		Read:   resourceProjectPolicyRead,
		Update: resourceProjectPolicyUpdate,
		Delete: resourceProjectPolicyDelete,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Read:   schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"project_name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"binding": {
				Type:     schema.TypeList,
				Required: true,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"role": {
							Type:     schema.TypeString,
							Required: true,
						},
						"members": {
							Type:     schema.TypeList,
							Required: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},
		},
	}
}

func resourceProjectPolicyCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	projectName, err := getProject(d, config)
	if err != nil {
		return err
	}
	d.Set("project_name", projectName)
	d.SetId(projectName)
	return resourceProjectPolicyUpdate(d, meta)
}

func resourceProjectPolicyRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	policyClient := config.SandwichClient.ProjectPolicy(d.Id())

	policy, err := policyClient.Get()
	if err != nil {
		return err
	}

	bindings := make([]map[string]interface{}, 0)
	for _, binding := range policy.Bindings {
		bindings = append(bindings, map[string]interface{}{
			"role":    binding.Role,
			"members": binding.Members,
		})
	}

	d.Set("binding", bindings)

	return nil
}

func resourceProjectPolicyUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	policyClient := config.SandwichClient.ProjectPolicy(d.Id())

	err := iamReadModifyWrite(d.Id(), policyClient, func(policy *api.Policy) {
		newBindings := make([]api.PolicyBinding, 0)

		bindings := d.Get("binding").([]interface{})
		for _, b := range bindings {
			binding := b.(map[string]interface{})
			newBinding := api.PolicyBinding{
				Role:    binding["role"].(string),
				Members: []string{},
			}
			for _, member := range binding["members"].([]interface{}) {
				newBinding.Members = append(newBinding.Members, member.(string))
			}
			newBindings = append(newBindings, newBinding)
		}
		policy.Bindings = newBindings
	})
	if err != nil {
		return err
	}

	return resourceProjectPolicyRead(d, meta)
}

func resourceProjectPolicyDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}
