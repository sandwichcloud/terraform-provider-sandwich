package sandwich

import (
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/sandwichcloud/deli-cli/api"
)

func resourceSystemPolicyMember() *schema.Resource {
	return &schema.Resource{
		Create: resourceSystemPolicyMemberCreate,
		Read:   resourceSystemPolicyMemberRead,
		Delete: resourceSystemPolicyMemberDelete,

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
			"member": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceSystemPolicyMemberCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	policyClient := config.SandwichClient.SystemPolicy()

	role := d.Get("role").(string)
	member := d.Get("member").(string)

	err := iamReadModifyWrite("system", policyClient, func(policy *api.Policy) {
		index := -1
		binding := api.PolicyBinding{
			Role:    role,
			Members: []string{},
		}
		for i, b := range policy.Bindings {
			if b.Role == role {
				index = i
				binding = b
			}
		}
		binding.Members = append(binding.Members, member)

		if index == -1 {
			policy.Bindings = append(policy.Bindings, binding)
		} else {
			policy.Bindings[index] = binding
		}
	})
	if err != nil {
		return err
	}

	d.SetId(role + "/" + member)
	return resourceSystemPolicyMemberRead(d, meta)
}

func resourceSystemPolicyMemberRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	policyClient := config.SandwichClient.SystemPolicy()

	policy, err := policyClient.Get()
	if err != nil {
		return err
	}

	role := strings.Split(d.Id(), "/")[0]
	member := strings.Split(d.Id(), "/")[1]

	memberIndex := -1

	for _, binding := range policy.Bindings {
		if binding.Role == role {
			for i, m := range binding.Members {
				if member == m {
					memberIndex = i
					break
				}
			}
			break
		}
	}

	if memberIndex == -1 {
		d.SetId("")
	}
	return nil
}

func resourceSystemPolicyMemberDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	policyClient := config.SandwichClient.SystemPolicy()

	role := strings.Split(d.Id(), "/")[0]
	member := strings.Split(d.Id(), "/")[1]

	err := iamReadModifyWrite("system", policyClient, func(policy *api.Policy) {
		for i, binding := range policy.Bindings {
			if binding.Role == role {
				for j, m := range binding.Members {
					if member == m {
						binding.Members = append(binding.Members[:j], binding.Members[j+1:]...)
						break
					}
				}
				policy.Bindings[i] = binding
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
