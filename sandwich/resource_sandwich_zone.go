package sandwich

import (
	"fmt"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/sandwichcloud/deli-cli/api"
	"github.com/sandwichcloud/deli-cli/api/client"
)

func resourceZone() *schema.Resource {
	return &schema.Resource{
		Create: resourceZoneCreate,
		Read:   resourceZoneRead,
		Update: resourceZoneUpdate,
		Delete: resourceZoneDelete,

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
			"region_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"vm_cluster": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"vm_datastore": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"vm_folder": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
				ForceNew: true,
			},
			"core_provision_percent": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  100,
				ForceNew: true,
			},
			"ram_provision_percent": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  100,
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

func resourceZoneCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	zoneClientClient := config.SandwichClient.Zone()
	name := d.Get("name").(string)
	regionID := d.Get("region_id").(string)
	vmCluster := d.Get("vm_cluster").(string)
	vmDatastore := d.Get("vm_datastore").(string)
	vmFolder := d.Get("vm_folder").(string)
	coreProvisionPercent := d.Get("core_provision_percent").(int)
	ramProvisionPercent := d.Get("ram_provision_percent").(int)

	zone, err := zoneClientClient.Create(name, regionID, vmCluster, vmDatastore, vmFolder, coreProvisionPercent, ramProvisionPercent)
	if err != nil {
		return err
	}

	d.Partial(true) // Things can still be created but error during a state change

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"ToCreate", "Creating"},
		Target:     []string{"Created"},
		Refresh:    ZoneRefreshFunc(zoneClientClient, zone.ID.String()),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}
	d.SetId(zone.ID.String())
	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for zone (%s) to become ready: %s", zone.ID.String(), err)
	}
	d.Partial(false) // There was no error during a state change so we should be safe

	return resourceZoneUpdate(d, meta)
}

func resourceZoneRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	zoneClient := config.SandwichClient.Zone()

	zone, err := zoneClient.Get(d.Id())
	if err != nil {
		if apiError, ok := err.(api.APIErrorInterface); ok {
			if apiError.IsNotFound() {
				d.SetId("")
				return nil
			}
		}
		return err
	}

	d.Set("name", zone.Name)
	d.Set("region_id", zone.RegionID.String())
	d.Set("vm_cluster", zone.VMCluster)
	d.Set("vm_datastore", zone.VMDatastore)
	d.Set("vm_folder", zone.VMFolder)
	d.Set("core_provision_percent", zone.CoreProvisionPercent)
	d.Set("ram_provision_percent", zone.RamProvisionPercent)

	return nil
}

func resourceZoneUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	zoneClient := config.SandwichClient.Zone()

	err := zoneClient.ActionSchedule(d.Id(), d.Get("schedulable").(bool))
	if err != nil {
		return err
	}

	return resourceZoneRead(d, meta)
}

func resourceZoneDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	zoneClient := config.SandwichClient.Zone()

	err := zoneClient.Delete(d.Id())
	if err != nil {
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"ToDelete", "Deleting"},
		Target:     []string{"Deleted"},
		Refresh:    ZoneRefreshFunc(zoneClient, d.Id()),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}
	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for zone (%s) to delete: %s", d.Id(), err)
	}

	d.SetId("")
	return nil
}

func ZoneRefreshFunc(zoneClient client.ZoneClientInterface, zoneID string) func() (result interface{}, state string, err error) {
	return func() (result interface{}, state string, err error) {
		zone, err := zoneClient.Get(zoneID)
		if err != nil {
			if apiError, ok := err.(api.APIErrorInterface); ok {
				if apiError.IsNotFound() {
					return zone, "Deleted", nil
				}
			}
			return nil, "", err
		}
		return zone, zone.State, nil
	}
}
