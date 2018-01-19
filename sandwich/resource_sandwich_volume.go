package sandwich

import (
	"fmt"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/sandwichcloud/deli-cli/api"
	"github.com/sandwichcloud/deli-cli/api/client"
	"github.com/satori/go.uuid"
)

func resourceVolume() *schema.Resource {
	return &schema.Resource{
		Create: resourceVolumeCreate,
		Read:   resourceVolumeRead,
		Update: resourceVolumeUpdate,
		Delete: resourceVolumeDelete,

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
			"zone_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"size": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: false,
			},
			"cloned_from": {
				Type:     schema.TypeString,
				Required: false,
				Optional: true,
				ForceNew: false,
			},
			"attached_to": {
				Type:     schema.TypeString,
				Required: false,
				Optional: true,
				ForceNew: false,
				Default:  uuid.UUID{}.String(),
			},
		},
	}
}

func resourceVolumeCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	volumeClient := config.SandwichClient.Volume()
	name := d.Get("name").(string)
	zoneID := d.Get("zone_id").(string)
	size := d.Get("size").(int)
	clonedFrom := d.Get("cloned_from").(string)

	if len(clonedFrom) == 0 {
		volume, err := volumeClient.Create(name, zoneID, size)
		if err != nil {
			return err
		}
		d.Partial(true)
		d.SetId(volume.ID.String())
		stateConf := &resource.StateChangeConf{
			Pending:    []string{"ToCreate", "Creating"},
			Target:     []string{"Created"},
			Refresh:    VolumeStateRefreshFunc(volumeClient, volume.ID.String()),
			Timeout:    d.Timeout(schema.TimeoutCreate),
			Delay:      10 * time.Second,
			MinTimeout: 3 * time.Second,
		}
		_, err = stateConf.WaitForState()
		if err != nil {
			return fmt.Errorf("Error waiting for volume (%s) to become ready: %s", volume.ID.String(), err)
		}
		d.Partial(false)
	} else {
		volume, err := volumeClient.Get(clonedFrom)
		if err != nil {
			return err
		}
		volume, err = volumeClient.ActionClone(volume.ID.String(), name)
		if err != nil {
			return err
		}
		d.Partial(true)
		d.SetId(volume.ID.String())
		stateConf := &resource.StateChangeConf{
			Pending:    []string{"ToCreate", "Creating"},
			Target:     []string{"Created"},
			Refresh:    VolumeStateRefreshFunc(volumeClient, volume.ID.String()),
			Timeout:    d.Timeout(schema.TimeoutCreate),
			Delay:      10 * time.Second,
			MinTimeout: 3 * time.Second,
		}
		_, err = stateConf.WaitForState()
		if err != nil {
			return fmt.Errorf("Error waiting for volume (%s) to become ready: %s", volume.ID.String(), err)
		}
		d.Partial(false)
	}

	return resourceVolumeUpdate(d, meta)
}

func resourceVolumeRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	volumeClient := config.SandwichClient.Volume()

	volume, err := volumeClient.Get(d.Id())
	if err != nil {
		if apiError, ok := err.(api.APIErrorInterface); ok {
			if apiError.IsNotFound() {
				d.SetId("")
				return nil
			}
		}
		return err
	}

	d.Set("name", volume.Name)
	d.Set("zone_id", volume.ZoneID)
	d.Set("size", volume.Size)
	d.Set("attached_to", volume.AttachedTo)

	return nil
}

func resourceVolumeUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	volumeClient := config.SandwichClient.Volume()

	size := d.Get("size").(int)
	attachedToStr := d.Get("attached_to").(string)
	attachedTo, err := uuid.FromString(attachedToStr)
	if err != nil {
		return err
	}

	volume, err := volumeClient.Get(d.Id())
	if err != nil {
		if apiError, ok := err.(api.APIErrorInterface); ok {
			if apiError.IsNotFound() {
				d.SetId("")
				return nil
			}
		}
		return err
	}

	if volume.AttachedTo != attachedTo {
		err := volumeClient.ActionDetach(volume.ID.String())
		if err != nil {
			if apiError, ok := err.(api.APIError); ok {
				if apiError.StatusCode != 409 {
					return err
				}
			}
		}

		stateConf := &resource.StateChangeConf{
			Pending:    []string{"DETACHING"},
			Target:     []string{""},
			Refresh:    VolumeTaskRefreshFunc(volumeClient, volume.ID.String()),
			Timeout:    d.Timeout(schema.TimeoutCreate),
			Delay:      10 * time.Second,
			MinTimeout: 3 * time.Second,
		}
		_, err = stateConf.WaitForState()
		if err != nil {
			return fmt.Errorf("Error waiting for volume (%s) to detach: %s", volume.ID.String(), err)
		}
	}

	if volume.Size != size {
		err := volumeClient.ActionGrow(d.Id(), size)
		if err != nil {
			return err
		}
		stateConf := &resource.StateChangeConf{
			Pending:    []string{"GROWING"},
			Target:     []string{""},
			Refresh:    VolumeTaskRefreshFunc(volumeClient, volume.ID.String()),
			Timeout:    d.Timeout(schema.TimeoutCreate),
			Delay:      10 * time.Second,
			MinTimeout: 3 * time.Second,
		}
		_, err = stateConf.WaitForState()
		if err != nil {
			return fmt.Errorf("Error waiting for volume (%s) to grow: %s", volume.ID.String(), err)
		}
	}

	if attachedTo != (uuid.UUID{}) {
		err := volumeClient.ActionAttach(volume.ID.String(), attachedTo.String())
		if err != nil {
			return err
		}
		stateConf := &resource.StateChangeConf{
			Pending:    []string{"ATTACHING"},
			Target:     []string{""},
			Refresh:    VolumeTaskRefreshFunc(volumeClient, volume.ID.String()),
			Timeout:    d.Timeout(schema.TimeoutCreate),
			Delay:      10 * time.Second,
			MinTimeout: 3 * time.Second,
		}
		_, err = stateConf.WaitForState()
		if err != nil {
			return fmt.Errorf("Error waiting for volume (%s) to attach: %s", volume.ID.String(), err)
		}
	}

	return resourceVolumeRead(d, meta)
}

func resourceVolumeDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	volumeClient := config.SandwichClient.Volume()

	err := volumeClient.ActionDetach(d.Id())
	if err != nil {
		if apiError, ok := err.(api.APIError); ok {
			if apiError.StatusCode != 409 {
				return err
			}
		}
	}
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"DETACHING"},
		Target:     []string{""},
		Refresh:    VolumeTaskRefreshFunc(volumeClient, d.Id()),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}
	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for volume (%s) to detach: %s", d.Id(), err)
	}

	err = volumeClient.Delete(d.Id())
	if err != nil {
		return err
	}

	stateConf = &resource.StateChangeConf{
		Pending:    []string{"ToDelete", "Deleting"},
		Target:     []string{"Deleted"},
		Refresh:    VolumeStateRefreshFunc(volumeClient, d.Id()),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}
	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for volume (%s) to delete: %s", d.Id(), err)
	}

	d.SetId("")

	return nil
}

func VolumeStateRefreshFunc(volumeClient client.VolumeClientInterface, volumeID string) func() (result interface{}, state string, err error) {
	return func() (result interface{}, state string, err error) {
		volume, err := volumeClient.Get(volumeID)
		if err != nil {
			if apiError, ok := err.(api.APIErrorInterface); ok {
				if apiError.IsNotFound() {
					return volume, "Deleted", nil
				}
			}
			return nil, "", err
		}
		return volume, volume.State, nil
	}
}

func VolumeTaskRefreshFunc(volumeClient client.VolumeClientInterface, volumeID string) func() (result interface{}, state string, err error) {
	return func() (result interface{}, state string, err error) {
		volume, err := volumeClient.Get(volumeID)
		if err != nil {
			if apiError, ok := err.(api.APIErrorInterface); ok {
				if apiError.IsNotFound() {
					return volume, "Deleted", nil
				}
			}
			return nil, "", err
		}
		return volume, volume.Task, nil
	}
}
