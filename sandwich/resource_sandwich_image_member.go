package sandwich

import (
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/sandwichcloud/deli-cli/api"
)

func resourceImageMember() *schema.Resource {
	return &schema.Resource{
		Create: resourceImageMemberCreate,
		Read:   resourceImageMemberRead,
		Delete: resourceImageMemberDelete,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Read:   schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"image_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"project_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceImageMemberCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	imageClient := config.SandwichClient.Image()
	imageID := d.Get("image_id").(string)
	projectID := d.Get("project_id").(string)

	err := imageClient.MemberAdd(imageID, projectID)
	if err != nil {
		return err
	}

	d.SetId(imageID + "/" + projectID)
	return resourceImageMemberRead(d, meta)
}

func resourceImageMemberRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	imageClient := config.SandwichClient.Image()
	ids := strings.Split(d.Id(), "/")
	imageID := ids[0]
	projectID := ids[1]

	imageMembers, err := imageClient.MemberList(imageID)
	if err != nil {
		if apiError, ok := err.(api.APIErrorInterface); ok {
			if apiError.IsNotFound() {
				d.SetId("")
				return nil
			}
		}
		return err
	}

	for _, imageMember := range imageMembers.Members {
		if imageMember.ProjectID.String() == projectID {
			d.Set("image_id", imageID)
			d.Set("project_id", projectID)
			return nil
		}
	}

	d.SetId("")
	return nil
}

func resourceImageMemberDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	imageClient := config.SandwichClient.Image()
	ids := strings.Split(d.Id(), "/")
	imageID := ids[0]
	projectID := ids[1]

	err := imageClient.MemberRemove(imageID, projectID)
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}
