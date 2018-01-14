package sandwich

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"api_server": {
				Type:     schema.TypeString,
				Required: true,
			},
			"username": {
				Type:     schema.TypeString,
				Required: true,
			},
			"password": {
				Type:      schema.TypeString,
				Required:  true,
				Sensitive: true,
			},
			"auth_method": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"project_id": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
		},
		DataSourcesMap: map[string]*schema.Resource{
			"sandwich_region":  dataSourceRegion(),
			"sandwich_network": dataSourceNetwork(),
		},
		ResourcesMap: map[string]*schema.Resource{
			"sandwich_region":   resourceRegion(),
			"sandwich_zone":     resourceZone(),
			"sandwich_network":  resourceNetwork(),
			"sandwich_project":  resourceProject(),
			"sandwich_image":    resourceImage(),
			"sandwich_keypair":  resourceKeypair(),
			"sandwich_flavor":   resourceFlavor(),
			"sandwich_instance": resourceInstance(),
		},
		ConfigureFunc: configureProvider,
	}
}

func configureProvider(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		APIServer:  d.Get("api_server").(string),
		Username:   d.Get("username").(string),
		Password:   d.Get("password").(string),
		AuthMethod: d.Get("auth_method").(string),
		ProjectID:  d.Get("project_id").(string),
	}

	if err := config.LoadAndValidate(); err != nil {
		return nil, err
	}

	return &config, nil
}
