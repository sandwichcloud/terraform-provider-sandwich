package sandwich

import (
	"time"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/sandwichcloud/deli-cli/api"
)

func resourceProjectMember() *schema.Resource {
	return &schema.Resource{
		Create: resourceProjectMemberCreate,
		Read:   resourceProjectMemberRead,
		Update: resourceProjectMemberUpdate,
		Delete: resourceProjectmemberDelete,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Read:   schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"username": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
			"driver": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
			"roles": {
				Type:     schema.TypeList,
				Required: false,
				Optional: true,
				ForceNew: false,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func resourceProjectMemberCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	projectClient := config.SandwichClient.Project()

	username := d.Get("username").(string)
	driver := d.Get("driver").(string)

	member, err := projectClient.AddMember(username, driver)
	if err != nil {
		return err
	}

	d.SetId(member.ID.String())

	return resourceProjectQuotaUpdate(d, meta)
}

func resourceProjectMemberRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	projectClient := config.SandwichClient.Project()

	member, err := projectClient.GetMember(d.Id())
	if err != nil {
		if apiError, ok := err.(api.APIErrorInterface); ok {
			if apiError.IsNotFound() {
				d.SetId("")
				return nil
			}
		}
	}

	d.Set("username", member.Username)
	d.Set("driver", member.Driver)
	d.Set("roles", member.Roles)

	return nil
}

func resourceProjectMemberUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	projectClient := config.SandwichClient.Project()

	roles := d.Get("roles").([]string)

	err := projectClient.UpdateMember(d.Id(), roles)
	if err != nil {
		return err
	}

	return resourceProjectQuotaRead(d, meta)
}

func resourceProjectmemberDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	projectClient := config.SandwichClient.Project()

	err := projectClient.RemoveMember(d.Id())
	if err != nil {
		return err
	}

	return nil
}
