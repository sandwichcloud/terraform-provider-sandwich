package sandwich

import (
	"fmt"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/sandwichcloud/deli-cli/api"
	"github.com/sandwichcloud/deli-cli/api/client"
)

func resourceImage() *schema.Resource {
	return &schema.Resource{
		Create: resourceImageCreate,
		Read:   resourceImageRead,
		Delete: resourceImageDelete,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Read:   schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"region_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"file_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			//"visibility": {
			//	Type:     schema.TypeString,
			//	Optional: true,
			//	Default:  "PRIVATE",
			//	ForceNew: true,
			//},
		},
	}
}

func resourceImageCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	imageClient := config.SandwichClient.Image()
	name := d.Get("name").(string)
	regionID := d.Get("region_id").(string)
	fileName := d.Get("file_name").(string)
	//visibility := d.Get("visibility").(string)

	image, err := imageClient.Create(name, regionID, fileName, "")
	if err != nil {
		return err
	}

	d.Partial(true) // Things can still be created but error during a state change

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"ToCreate", "Creating"},
		Target:     []string{"Created"},
		Refresh:    ImageRefreshFunc(imageClient, image.ID.String()),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}
	d.SetId(image.ID.String())
	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for image (%s) to become ready: %s", image.ID.String(), err)
	}
	d.Partial(false) // There was no error during a state change so we should be safe

	return resourceImageRead(d, meta)
}

func resourceImageRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	imageClient := config.SandwichClient.Image()

	image, err := imageClient.Get(d.Id())
	if err != nil {
		if apiError, ok := err.(api.APIErrorInterface); ok {
			if apiError.IsNotFound() {
				d.SetId("")
				return nil
			}
		}
		return err
	}

	d.Set("name", image.Name)
	d.Set("region_id", image.RegionID.String())
	d.Set("file_name", image.FileName)
	//d.Set("visibility", image.Visibility)

	return nil
}

func resourceImageDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	imageClient := config.SandwichClient.Image()

	err := imageClient.Delete(d.Id())
	if err != nil {
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"ToDelete", "Deleting"},
		Target:     []string{"Deleted"},
		Refresh:    ImageRefreshFunc(imageClient, d.Id()),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}
	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for image (%s) to delete: %s", d.Id(), err)
	}

	d.SetId("")
	return nil
}

func ImageRefreshFunc(imageClient client.ImageClientInterface, imageID string) func() (result interface{}, state string, err error) {
	return func() (result interface{}, state string, err error) {
		image, err := imageClient.Get(imageID)
		if err != nil {
			if apiError, ok := err.(api.APIErrorInterface); ok {
				if apiError.IsNotFound() {
					return image, "Deleted", nil
				}
			}
			return nil, "", err
		}
		return image, image.State, nil
	}
}
