package sandwich

import (
	"time"

	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/sandwichcloud/deli-cli/api"
)

func resourceSystemPolicy() *schema.Resource {
	return &schema.Resource{
		Create: resourceSystemPolicyCreate,
		Read:   resourceSystemPolicyRead,
		Update: resourceSystemPolicyUpdate,
		Delete: resourceSystemPolicyDelete,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Read:   schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
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

func resourceSystemPolicyCreate(d *schema.ResourceData, meta interface{}) error {
	id, err := uuid.GenerateUUID()
	if err != nil {
		return err
	}
	d.SetId(id)
	return resourceSystemPolicyUpdate(d, meta)
}

func resourceSystemPolicyRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	policyClient := config.SandwichClient.SystemPolicy()

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

func resourceSystemPolicyUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	policyClient := config.SandwichClient.SystemPolicy()

	err := iamReadModifyWrite("system", policyClient, func(policy *api.Policy) {
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

	return resourceSystemPolicyRead(d, meta)
}

func resourceSystemPolicyDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}
