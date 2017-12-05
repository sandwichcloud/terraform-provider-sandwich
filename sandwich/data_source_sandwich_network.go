package sandwich

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceNetwork() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceNetworkRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"region_id": {
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

	networkList, err := networkClient.List(d.Get("name").(string), d.Get("region_id").(string), 2, "")
	if err != nil {
		return err
	}

	if len(networkList.Networks) == 0 {
		return fmt.Errorf("Returned no results")
	}

	if len(networkList.Networks) > 1 {
		return fmt.Errorf("Returned more than one result")
	}

	network := networkList.Networks[0]

	d.SetId(network.ID.String())
	d.Set("region_id", network.RegionID.String())
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
