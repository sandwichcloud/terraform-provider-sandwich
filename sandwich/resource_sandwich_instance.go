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
			"project_name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"image_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"service_account_name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"network_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"region_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"zone_name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"flavor_name": {
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
			"keypair_names": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"volumes": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"size": {
							Type:     schema.TypeInt,
							Optional: true,
							ForceNew: true,
						},
						"auto_delete": {
							Type:     schema.TypeBool,
							Default:  true,
							Optional: true,
							ForceNew: true,
						},
					},
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
	projectName, err := getProject(d, config)
	if err != nil {
		return err
	}

	instanceClient := config.SandwichClient.Instance(projectName)
	name := d.Get("name").(string)
	imageName := d.Get("image_name").(string)
	serviceAccountName := d.Get("service_account_name").(string)
	networkName := d.Get("network_name").(string)
	regionName := d.Get("region_name").(string)
	zoneName := d.Get("zone_name").(string)
	flavorName := d.Get("flavor_name").(string)
	disk := d.Get("disk").(int)
	userData := d.Get("user_data").(string)
	d.Set("project_name", projectName)
	var keypairNames []string
	var initialVolumes []api.InstanceInitialVolume
	tags := map[string]string{}

	for _, keypairName := range d.Get("keypair_names").([]interface{}) {
		keypairNames = append(keypairNames, keypairName.(string))
	}

	for k, v := range d.Get("tags").(map[string]interface{}) {
		tags[k] = v.(string)
	}

	for _, volumeInfoInt := range d.Get("volumes").([]interface{}) {
		volumeInfo := volumeInfoInt.(map[string]interface{})
		initialVolumes = append(initialVolumes, api.InstanceInitialVolume{
			Size:       volumeInfo["size"].(int),
			AutoDelete: volumeInfo["auto_delete"].(bool),
		})
	}

	instance, err := instanceClient.Create(name, imageName, regionName, zoneName, networkName, serviceAccountName, flavorName, disk, keypairNames, initialVolumes, tags, userData)
	if err != nil {
		return err
	}

	d.Partial(true) // Things can still be created but error during a state change

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"ToCreate", "Creating"},
		Target:     []string{"Created"},
		Refresh:    InstanceRefreshFunc(instanceClient, instance.Name),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}
	d.SetId(instance.Name)
	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for instance (%s) to become ready: %s", instance.Name, err)
	}
	d.Partial(false) // There was no error during a state change so we should be safe

	return resourceInstanceRead(d, meta)
}

func resourceInstanceRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	projectName := d.Get("project_name").(string)
	instanceClient := config.SandwichClient.Instance(projectName)
	networkPortClient := config.SandwichClient.NetworkPort(projectName)

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

	d.Set("image_name", instance.ImageName)
	d.Set("service_account_name", instance.ServiceAccountName) // TODO: this

	networkPort, err := networkPortClient.Get(instance.NetworkPortID.String())
	if err != nil {
		return err
	}

	d.Set("network_name", networkPort.NetworkName)
	d.Set("region_name", instance.RegionName)
	d.Set("zone_name", instance.ZoneName)
	d.Set("flavor_name", instance.FlavorName)
	d.Set("disk", instance.Disk)
	d.Set("user_data", instance.UserData)
	var keypairNames []string
	tags := map[string]string{}

	for _, keypairName := range instance.KeypairNames {
		keypairNames = append(keypairNames, keypairName)
	}

	for k, v := range instance.Tags {
		tags[k] = v
	}

	d.Set("keypair_names", keypairNames)
	d.Set("tags", tags)

	return nil
}

func resourceInstanceDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	instanceClient := config.SandwichClient.Instance(d.Get("project_name").(string))

	err := instanceClient.Delete(d.Id())
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

func InstanceRefreshFunc(instanceClient client.InstanceClientInterface, instanceName string) func() (result interface{}, state string, err error) {
	return func() (result interface{}, state string, err error) {
		instance, err := instanceClient.Get(instanceName)
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
