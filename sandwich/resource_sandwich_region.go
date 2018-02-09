package sandwich

import (
	"fmt"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/sandwichcloud/deli-cli/api"
	"github.com/sandwichcloud/deli-cli/api/client"
)

func resourceRegion() *schema.Resource {
	return &schema.Resource{
		Create: resourceRegionCreate,
		Read:   resourceRegionRead,
		Update: resourceRegionUpdate,
		Delete: resourceRegionDelete,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Read:   schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"datacenter": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"image_datastore": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"image_folder": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
				ForceNew: true,
			},
			"schedulable": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
		},
	}
}

func resourceRegionCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	regionClient := config.SandwichClient.Region()
	name := d.Get("name").(string)
	datacenter := d.Get("datacenter").(string)
	imageDatastore := d.Get("image_datastore").(string)
	imageFolder := d.Get("image_folder").(string)

	region, err := regionClient.Create(name, datacenter, imageDatastore, imageFolder)
	if err != nil {
		return err
	}

	d.Partial(true) // Things can still be created but error during a state change

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"ToCreate", "Creating"},
		Target:     []string{"Created"},
		Refresh:    RegionRefreshFunc(regionClient, region.ID.String()),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}
	d.SetId(region.ID.String())
	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for region (%s) to become ready: %s", region.ID.String(), err)
	}
	d.Partial(false) // There was no error during a state change so we should be safe

	return resourceRegionUpdate(d, meta)
}

func resourceRegionRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	regionClient := config.SandwichClient.Region()

	region, err := regionClient.Get(d.Id())
	if err != nil {
		if apiError, ok := err.(api.APIErrorInterface); ok {
			if apiError.IsNotFound() {
				d.SetId("")
				return nil
			}
		}
		return err
	}

	d.Set("name", region.Name)
	d.Set("datacenter", region.Datacenter)
	d.Set("image_datastore", region.ImageDatastore)
	d.Set("image_folder", region.ImageFolder)
	d.Set("schedulable", region.Schedulable)

	return nil
}

func resourceRegionUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	regionClient := config.SandwichClient.Region()

	err := regionClient.ActionSchedule(d.Id(), d.Get("schedulable").(bool))
	if err != nil {
		return err
	}

	return resourceRegionRead(d, meta)
}

func resourceRegionDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	regionClient := config.SandwichClient.Region()

	err := regionClient.ActionSchedule(d.Id(), false)
	if err != nil {
		return err
	}

	err = regionClient.Delete(d.Id())
	if err != nil {
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"ToDelete", "Deleting"},
		Target:     []string{"Deleted"},
		Refresh:    RegionRefreshFunc(regionClient, d.Id()),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}
	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for region (%s) to delete: %s", d.Id(), err)
	}

	d.SetId("")
	return nil
}

func RegionRefreshFunc(regionClient client.RegionClientInterface, regionID string) func() (result interface{}, state string, err error) {
	return func() (result interface{}, state string, err error) {
		region, err := regionClient.Get(regionID)
		if err != nil {
			if apiError, ok := err.(api.APIErrorInterface); ok {
				if apiError.IsNotFound() {
					return region, "Deleted", nil
				}
			}
			return nil, "", err
		}
		return region, region.State, nil
	}
}
