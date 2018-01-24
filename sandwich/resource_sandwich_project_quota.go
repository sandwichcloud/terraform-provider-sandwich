package sandwich

import (
	"time"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/sandwichcloud/deli-cli/api"
	"github.com/satori/go.uuid"
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
	d.SetId(uuid.NewV4().String())
	return resourceProjectQuotaUpdate(d, meta)
}

func resourceProjectQuotaRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	projectClient := config.SandwichClient.Project()

	quota, err := projectClient.GetQuota()
	if err != nil {
		if apiError, ok := err.(api.APIErrorInterface); ok {
			if apiError.IsNotFound() {
				d.SetId("")
				return nil
			}
		}
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

	err := projectClient.SetQuota(vcpu, ram, disk)
	if err != nil {
		return err
	}
	return resourceProjectQuotaRead(d, meta)
}

func resourceProjectQuotaDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}
