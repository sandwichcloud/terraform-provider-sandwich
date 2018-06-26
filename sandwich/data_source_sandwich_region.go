package sandwich

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/sandwichcloud/deli-cli/api"
)

func dataSourceRegion() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceRegionRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"datacenter": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"image_datastore": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"image_folder": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"schedulable": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func dataSourceRegionRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	regionClient := config.SandwichClient.Region()

	regionName := d.Get("name").(string)
	region, err := regionClient.Get(regionName)
	if err != nil {
		if apiError, ok := err.(api.APIErrorInterface); ok {
			if apiError.IsNotFound() {
				return fmt.Errorf("Could not find a region with the name of %s", regionName)
			}
		}
		return err
	}

	d.SetId(region.Name)
	d.Set("datacenter", region.Datacenter)
	d.Set("image_datastore", region.ImageDatastore)
	d.Set("image_folder", region.ImageFolder)
	d.Set("schedulable", region.Schedulable)

	return nil
}
