package sandwich

import (
	"time"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/sandwichcloud/deli-cli/api"
)

func resourceSystemPolicyBinding() *schema.Resource {
	return &schema.Resource{
		Create: resourceSystemPolicyBindingCreate,
		Read:   resourceSystemPolicyBindingRead,
		Update: resourceSystemPolicyBindingUpdate,
		Delete: resourceSystemPolicyBindingDelete,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Read:   schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"role": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"members": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func resourceSystemPolicyBindingCreate(d *schema.ResourceData, meta interface{}) error {
	role := d.Get("role").(string)
	d.SetId(role)
	return resourceSystemPolicyBindingUpdate(d, meta)
}

func resourceSystemPolicyBindingRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	role := d.Id()
	policyClient := config.SandwichClient.SystemPolicy()

	policy, err := policyClient.Get()
	if err != nil {
		return err
	}

	index := -1

	for i, binding := range policy.Bindings {
		if binding.Role == role {
			index = i
			d.Set("members", binding.Members)
			break
		}
	}

	if index == -1 {
		d.SetId("")
	}

	return nil
}

func resourceSystemPolicyBindingUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	policyClient := config.SandwichClient.SystemPolicy()

	err := iamReadModifyWrite("system", policyClient, func(policy *api.Policy) {
		index := -1

		for i, binding := range policy.Bindings {
			if binding.Role == d.Id() {
				index = i
				break
			}
		}

		binding := api.PolicyBinding{
			Role:    d.Get("role").(string),
			Members: make([]string, 0),
		}
		for _, member := range d.Get("members").([]interface{}) {
			binding.Members = append(binding.Members, member.(string))
		}

		if index == -1 {
			policy.Bindings = append(policy.Bindings, binding)
		} else {
			policy.Bindings[index] = binding
		}
	})
	if err != nil {
		return err
	}

	return resourceSystemPolicyBindingRead(d, meta)
}

func resourceSystemPolicyBindingDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	policyClient := config.SandwichClient.SystemPolicy()

	err := iamReadModifyWrite("system", policyClient, func(policy *api.Policy) {
		for i, binding := range policy.Bindings {
			if binding.Role == d.Id() {
				policy.Bindings = append(policy.Bindings[:i], policy.Bindings[i+1:]...)
				break
			}
		}
	})
	if err != nil {
		return err
	}

	d.SetId("")

	return nil
}
