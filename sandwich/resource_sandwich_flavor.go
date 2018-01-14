package sandwich

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/sandwichcloud/deli-cli/api"
)

func resourceFlavor() *schema.Resource {
	return &schema.Resource{
		Create: resourceFlavorCreate,
		Read:   resourceFlavorRead,
		Delete: resourceFlavorDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"vcpus": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
			"ram": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
			"disk": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceFlavorCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	flavorClient := config.SandwichClient.Flavor()
	name := d.Get("name").(string)
	vcpus := d.Get("vcpus").(int)
	ram := d.Get("ram").(int)
	disk := d.Get("disk").(int)

	flavor, err := flavorClient.Create(name, vcpus, ram, disk)
	if err != nil {
		return err
	}

	d.SetId(flavor.ID.String())

	return resourceFlavorRead(d, meta)
}

func resourceFlavorRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	flavorClient := config.SandwichClient.Flavor()

	flavor, err := flavorClient.Get(d.Id())
	if err != nil {
		if apiError, ok := err.(api.APIErrorInterface); ok {
			if apiError.IsNotFound() {
				d.SetId("")
				return nil
			}
		}
		return err
	}

	d.Set("name", flavor.Name)
	d.Set("vcpus", flavor.VCPUS)
	d.Set("ram", flavor.Ram)
	d.Set("disk", flavor.Disk)

	return nil
}

func resourceFlavorDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	flavorClient := config.SandwichClient.Flavor()

	err := flavorClient.Delete(d.Id())
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}
