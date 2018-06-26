package sandwich

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/sandwichcloud/deli-cli/api"
)

func resourceKeypair() *schema.Resource {
	return &schema.Resource{
		Create: resourceKeypairCreate,
		Read:   resourceKeypairRead,
		Delete: resourceKeypairDelete,

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
			"public_key": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceKeypairCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	projectName, err := getProject(d, config)
	if err != nil {
		return err
	}

	keypairClient := config.SandwichClient.Keypair(projectName)
	name := d.Get("name").(string)
	public_key := d.Get("public_key").(string)
	d.Set("project_name", projectName)

	keypair, err := keypairClient.Create(name, public_key)
	if err != nil {
		return err
	}

	d.SetId(keypair.Name)

	return resourceKeypairRead(d, meta)
}

func resourceKeypairRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	keypairClient := config.SandwichClient.Keypair(d.Get("project_name").(string))
	keypair, err := keypairClient.Get(d.Id())
	if err != nil {
		if apiError, ok := err.(api.APIErrorInterface); ok {
			if apiError.IsNotFound() {
				d.SetId("")
				return nil
			}
		}
		return err
	}

	d.Set("public_key", keypair.PublicKey)

	return nil
}

func resourceKeypairDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	keypairClient := config.SandwichClient.Keypair(d.Get("project_name").(string))

	err := keypairClient.Delete(d.Id())
	if err != nil {
		if apiError, ok := err.(api.APIErrorInterface); ok {
			if apiError.IsNotFound() {
				d.SetId("")
				return nil
			}
		}
		return err
	}

	d.SetId("")
	return nil
}
