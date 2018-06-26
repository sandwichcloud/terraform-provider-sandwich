package sandwich

import (
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/sandwichcloud/deli-cli/api"
)

func resourceProjectPolicyBinding() *schema.Resource {
	return &schema.Resource{
		Create: resourceProjectPolicyBindingCreate,
		Read:   resourceProjectPolicyBindingRead,
		Update: resourceProjectPolicyBindingUpdate,
		Delete: resourceProjectPolicyBindingDelete,

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

func resourceProjectPolicyBindingCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	projectName, err := getProject(d, config)
	if err != nil {
		return err
	}
	d.Set("project_name", projectName)
	role := d.Get("role").(string)
	d.SetId(projectName + "/" + role)
	return resourceProjectPolicyBindingUpdate(d, meta)
}

func resourceProjectPolicyBindingRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	projectName := strings.Split(d.Id(), "/")[0]
	role := strings.Split(d.Id(), "/")[1]
	policyClient := config.SandwichClient.ProjectPolicy(projectName)

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

func resourceProjectPolicyBindingUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	projectName := strings.Split(d.Id(), "/")[0]
	role := strings.Split(d.Id(), "/")[1]
	policyClient := config.SandwichClient.ProjectPolicy(projectName)

	err := iamReadModifyWrite(projectName, policyClient, func(policy *api.Policy) {
		index := -1

		for i, binding := range policy.Bindings {
			if binding.Role == role {
				index = i
				break
			}
		}

		binding := api.PolicyBinding{
			Role:    role,
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

	return resourceProjectPolicyBindingRead(d, meta)
}

func resourceProjectPolicyBindingDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	projectName := strings.Split(d.Id(), "/")[0]
	role := strings.Split(d.Id(), "/")[1]
	policyClient := config.SandwichClient.ProjectPolicy(projectName)

	err := iamReadModifyWrite("system", policyClient, func(policy *api.Policy) {
		for i, binding := range policy.Bindings {
			if binding.Role == role {
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
