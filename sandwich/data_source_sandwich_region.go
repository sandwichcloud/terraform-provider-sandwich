package sandwich

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
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

	regionList, err := regionClient.List(d.Get("name").(string), 2, "")
	if err != nil {
		return err
	}

	if len(regionList.Regions) == 0 {
		return fmt.Errorf("Returned no results")
	}

	if len(regionList.Regions) > 1 {
		return fmt.Errorf("Returned more than one result")
	}

	region := regionList.Regions[0]

	d.SetId(region.ID.String())
	d.Set("datacenter", region.Datacenter)
	d.Set("image_datastore", region.ImageDatastore)
	d.Set("image_folder", region.ImageFolder)
	d.Set("schedulable", region.Schedulable)

	return nil
}
