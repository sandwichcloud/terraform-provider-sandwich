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
			"token": {
				Type:     schema.TypeString,
				Required: true,
			},
			"project_name": {
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
			"sandwich_location_region":             resourceRegion(),
			"sandwich_location_zone":               resourceZone(),
			"sandwich_compute_network":             resourceNetwork(),
			"sandwich_compute_image":               resourceImage(),
			"sandwich_compute_keypair":             resourceKeypair(),
			"sandwich_compute_flavor":              resourceFlavor(),
			"sandwich_compute_instance":            resourceInstance(),
			"sandwich_compute_volume":              resourceVolume(),
			"sandwich_iam_project":                 resourceProject(),
			"sandwich_iam_project_quota":           resourceProjectQuota(),
			"sandwich_iam_system_role":             resourceSystemRole(),
			"sandwich_iam_project_role":            resourceProjectRole(),
			"sandwich_iam_system_service_account":  resourceSystemServiceAccount(),
			"sandwich_iam_project_service_account": resourceProjectServiceAccount(),
			"sandwich_iam_system_policy":           resourceSystemPolicy(),
			"sandwich_iam_system_policy_binding":   resourceSystemPolicyBinding(),
			"sandwich_iam_system_policy_member":    resourceSystemPolicyMember(),
			"sandwich_iam_project_policy":          resourceProjectPolicy(),
			"sandwich_iam_project_policy_binding":  resourceProjectPolicyBinding(),
			"sandwich_iam_project_policy_member":   resourceProjectPolicyMember(),
		},
		ConfigureFunc: configureProvider,
	}
}

func configureProvider(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		APIServer:   d.Get("api_server").(string),
		Token:       d.Get("token").(string),
		ProjectName: d.Get("project_name").(string),
	}

	if err := config.LoadAndValidate(); err != nil {
		return nil, err
	}

	return &config, nil
}
