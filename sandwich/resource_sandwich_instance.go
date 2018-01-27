package sandwich

import (
	"fmt"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/sandwichcloud/deli-cli/api"
	"github.com/sandwichcloud/deli-cli/api/client"
)

func resourceInstance() *schema.Resource {
	return &schema.Resource{
		Create: resourceInstanceCreate,
		Read:   resourceInstanceRead,
		Delete: resourceInstanceDelete,

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
			"image_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"service_account_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"network_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"region_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"zone_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"flavor_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"disk": {
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"keypair_ids": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"tags": {
				Type:     schema.TypeMap,
				Optional: true,
				Default:  map[string]string{},
				ForceNew: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"user_data": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
		},
	}
}

func resourceInstanceCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	instanceClient := config.SandwichClient.Instance()
	name := d.Get("name").(string)
	imageID := d.Get("image_id").(string)
	serviceAccountID := d.Get("service_account_id").(string)
	networkID := d.Get("network_id").(string)
	regionID := d.Get("region_id").(string)
	zoneID := d.Get("zone_id").(string)
	flavorID := d.Get("flavor_id").(string)
	disk := d.Get("disk").(int)
	userData := d.Get("user_data").(string)
	var keypairIDs []string
	tags := map[string]string{}

	for _, keypairID := range d.Get("keypair_ids").([]interface{}) {
		keypairIDs = append(keypairIDs, keypairID.(string))
	}

	for k, v := range d.Get("tags").(map[string]interface{}) {
		tags[k] = v.(string)
	}

	instance, err := instanceClient.Create(name, imageID, regionID, zoneID, networkID, serviceAccountID, flavorID, disk, keypairIDs, tags, userData)
	if err != nil {
		return err
	}

	d.Partial(true) // Things can still be created but error during a state change

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"ToCreate", "Creating"},
		Target:     []string{"Created"},
		Refresh:    InstanceRefreshFunc(instanceClient, instance.ID.String()),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}
	d.SetId(instance.ID.String())
	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for instance (%s) to become ready: %s", instance.ID.String(), err)
	}
	d.Partial(false) // There was no error during a state change so we should be safe

	return resourceInstanceRead(d, meta)
}

func resourceInstanceRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	instanceClient := config.SandwichClient.Instance()
	networkPortClient := config.SandwichClient.NetworkPort()

	instance, err := instanceClient.Get(d.Id())
	if err != nil {
		if apiError, ok := err.(api.APIErrorInterface); ok {
			if apiError.IsNotFound() {
				d.SetId("")
				return nil
			}
		}
		return err
	}

	d.Set("name", instance.Name)
	d.Set("image_id", instance.ImageID.String())
	d.Set("service_account_id", instance.ServiceAccountID.String()) // TODO: this

	networkPort, err := networkPortClient.Get(instance.NetworkPortID.String())
	if err != nil {
		return err
	}

	d.Set("network_id", networkPort.NetworkID.String())
	d.Set("region_id", instance.RegionID.String())
	d.Set("zone_id", instance.ZoneID.String())
	d.Set("flavor_id", instance.FlavorID.String())
	d.Set("disk", instance.Disk)
	d.Set("user_data", instance.UserData)
	var keypairIDs []string
	tags := map[string]string{}

	for _, keypair := range instance.KeypairIDs {
		keypairIDs = append(keypairIDs, keypair.String())
	}

	for k, v := range instance.Tags {
		tags[k] = v
	}

	d.Set("keypair_ids", keypairIDs)
	d.Set("tags", tags)

	return nil
}

func resourceInstanceDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	instanceClient := config.SandwichClient.Instance()

	err := instanceClient.Delete(d.Id())
	if err != nil {
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"ToDelete", "Deleting"},
		Target:     []string{"Deleted"},
		Refresh:    InstanceRefreshFunc(instanceClient, d.Id()),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}
	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for instance (%s) to delete: %s", d.Id(), err)
	}

	d.SetId("")
	return nil
}

func InstanceRefreshFunc(instanceClient client.InstanceClientInterface, instanceID string) func() (result interface{}, state string, err error) {
	return func() (result interface{}, state string, err error) {
		instance, err := instanceClient.Get(instanceID)
		if err != nil {
			if apiError, ok := err.(api.APIErrorInterface); ok {
				if apiError.IsNotFound() {
					return instance, "Deleted", nil
				}
			}
			return nil, "", err
		}
		return instance, instance.State, nil
	}
}
