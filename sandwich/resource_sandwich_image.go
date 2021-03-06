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
			"project_name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"region_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"file_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceImageCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	projectName, err := getProject(d, config)
	if err != nil {
		return err
	}

	imageClient := config.SandwichClient.Image(projectName)
	name := d.Get("name").(string)
	regionName := d.Get("region_name").(string)
	fileName := d.Get("file_name").(string)
	d.Set("project_name", projectName)

	image, err := imageClient.Create(name, regionName, fileName)
	if err != nil {
		return err
	}

	d.SetId(image.Name)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"ToCreate", "Creating"},
		Target:     []string{"Created"},
		Refresh:    ImageRefreshFunc(imageClient, image.Name),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for image (%s) to become ready: %s", image.Name, err)
	}

	return resourceImageRead(d, meta)
}

func resourceImageRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	imageClient := config.SandwichClient.Image(d.Get("project_name").(string))

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

	d.Set("region_name", image.RegionName)
	d.Set("file_name", image.FileName)

	return nil
}

func resourceImageDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	imageClient := config.SandwichClient.Image(d.Get("project_name").(string))

	err := imageClient.Delete(d.Id())
	if err != nil {
		if apiError, ok := err.(api.APIErrorInterface); ok {
			if apiError.IsNotFound() {
				d.SetId("")
				return nil
			}
		}
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

func ImageRefreshFunc(imageClient client.ImageClientInterface, imageName string) func() (result interface{}, state string, err error) {
	return func() (result interface{}, state string, err error) {
		image, err := imageClient.Get(imageName)
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
