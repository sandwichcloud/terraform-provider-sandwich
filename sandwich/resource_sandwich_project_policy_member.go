package sandwich

import (
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/sandwichcloud/deli-cli/api"
)

func resourceProjectPolicyMember() *schema.Resource {
	return &schema.Resource{
		Create: resourceProjectPolicyMemberCreate,
		Read:   resourceProjectPolicyMemberRead,
		Delete: resourceProjectPolicyMemberDelete,

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

func resourceProjectPolicyMemberCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	projectName, err := getProject(d, config)
	if err != nil {
		return err
	}
	d.Set("project_name", projectName)

	policyClient := config.SandwichClient.ProjectPolicy(projectName)

	role := d.Get("role").(string)
	member := d.Get("member").(string)

	err = iamReadModifyWrite(projectName, policyClient, func(policy *api.Policy) {
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

	d.SetId(projectName + "/" + role + "/" + member)
	return resourceProjectPolicyMemberRead(d, meta)
}

func resourceProjectPolicyMemberRead(d *schema.ResourceData, meta interface{}) error {
	projectName := strings.Split(d.Id(), "/")[0]
	role := strings.Split(d.Id(), "/")[1]
	member := strings.Split(d.Id(), "/")[2]

	config := meta.(*Config)
	policyClient := config.SandwichClient.ProjectPolicy(projectName)

	policy, err := policyClient.Get()
	if err != nil {
		return err
	}

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

func resourceProjectPolicyMemberDelete(d *schema.ResourceData, meta interface{}) error {
	projectName := strings.Split(d.Id(), "/")[0]
	role := strings.Split(d.Id(), "/")[1]
	member := strings.Split(d.Id(), "/")[2]

	config := meta.(*Config)
	policyClient := config.SandwichClient.ProjectPolicy(projectName)

	err := iamReadModifyWrite(projectName, policyClient, func(policy *api.Policy) {
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
