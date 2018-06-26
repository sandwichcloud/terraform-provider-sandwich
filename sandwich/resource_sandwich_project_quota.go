package sandwich

import (
	"fmt"
	"time"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/sandwichcloud/deli-cli/api"
)

func resourceProjectQuota() *schema.Resource {
	return &schema.Resource{
		Create: resourceProjectQuotaCreate,
		Read:   resourceProjectQuotaRead,
		Update: resourceProjectQuotaUpdate,
		Delete: resourceProjectQuotaDelete,

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
			"vcpu": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: false,
			},
			"ram": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: false,
			},
			"disk": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: false,
			},
		},
	}
}

func resourceProjectQuotaCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	projectClient := config.SandwichClient.Project()
	projectName, err := getProject(d, config)
	if err != nil {
		return err
	}

	_, err = projectClient.GetQuota(projectName)
	if apiError, ok := err.(api.APIErrorInterface); ok {
		if apiError.IsNotFound() {
			d.SetId("")
			return fmt.Errorf("Could not find quota for project %s", projectName)
		}
	}

	d.Set("project_name", projectName)
	d.SetId(projectName)
	return resourceProjectQuotaUpdate(d, meta)
}

func resourceProjectQuotaRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	projectClient := config.SandwichClient.Project()

	quota, err := projectClient.GetQuota(d.Id())
	if err != nil {
		if apiError, ok := err.(api.APIErrorInterface); ok {
			if apiError.IsNotFound() {
				d.SetId("")
				return nil
			}
		}
		return err
	}

	d.Set("vcpu", quota.VCPU)
	d.Set("ram", quota.Ram)
	d.Set("disk", quota.Disk)
	return nil
}

func resourceProjectQuotaUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	projectClient := config.SandwichClient.Project()
	vcpu := d.Get("vcpu").(int)
	ram := d.Get("ram").(int)
	disk := d.Get("disk").(int)

	err := projectClient.SetQuota(d.Id(), vcpu, ram, disk)
	if err != nil {
		return err
	}
	return resourceProjectQuotaRead(d, meta)
}

func resourceProjectQuotaDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}
