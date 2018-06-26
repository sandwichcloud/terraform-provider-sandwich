package sandwich

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/sandwichcloud/deli-cli/api"
)

func dataSourceNetwork() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceNetworkRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"region_name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"port_group": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"cidr": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"gateway": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"pool_start": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"pool_end": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"dns_servers": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func dataSourceNetworkRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkClient := config.SandwichClient.Network()

	networkName := d.Get("name").(string)
	network, err := networkClient.Get(networkName)
	if err != nil {
		if apiError, ok := err.(api.APIErrorInterface); ok {
			if apiError.IsNotFound() {
				return fmt.Errorf("Could not find a network with the name of %s", networkName)
			}
		}
		return err
	}

	d.SetId(network.Name)
	d.Set("region_name", network.RegionName)
	d.Set("port_group", network.PortGroup)
	d.Set("cidr", network.Cidr)
	d.Set("gateway", network.Gateway.String())
	d.Set("pool_start", network.PoolStart.String())
	d.Set("pool_end", network.PoolEnd.String())

	var dnsServers []string
	for _, dnsServer := range network.DNSServers {
		dnsServers = append(dnsServers, dnsServer.String())
	}
	d.Set("dns_servers", dnsServers)

	return nil
}
