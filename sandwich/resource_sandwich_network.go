package sandwich

import (
	"fmt"
	"net"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/sandwichcloud/deli-cli/api"
	"github.com/sandwichcloud/deli-cli/api/client"
)

func resourceNetwork() *schema.Resource {
	return &schema.Resource{
		Create: resourceNetworkCreate,
		Read:   resourceNetworkRead,
		Delete: resourceNetworkDelete,

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
			"region_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"port_group": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"cidr": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"gateway": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"pool_start": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"pool_end": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"dns_servers": {
				Type:     schema.TypeList,
				Required: true,
				ForceNew: true,
				MinItems: 1,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func resourceNetworkCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkClient := config.SandwichClient.Network()

	name := d.Get("name").(string)
	regionName := d.Get("region_name").(string)
	portGroup := d.Get("port_group").(string)
	cidr := d.Get("cidr").(string)
	gateway := net.ParseIP(d.Get("gateway").(string))
	poolStart := net.ParseIP(d.Get("pool_start").(string))
	poolEnd := net.ParseIP(d.Get("pool_end").(string))
	var dnsServers []net.IP

	for _, dnsServer := range d.Get("dns_servers").([]interface{}) {
		dnsServers = append(dnsServers, net.ParseIP(dnsServer.(string)))
	}

	network, err := networkClient.Create(name, regionName, portGroup, cidr, gateway, poolStart, poolEnd, dnsServers)
	if err != nil {
		return err
	}

	d.Partial(true) // Things can still be created but error during a state change

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"ToCreate", "Creating"},
		Target:     []string{"Created"},
		Refresh:    NetworkRefreshFunc(networkClient, network.Name),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}
	d.SetId(network.Name)
	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for network (%s) to become ready: %s", network.Name, err)
	}
	d.Partial(false) // There was no error during a state change so we should be safe
	return resourceNetworkRead(d, meta)
}

func resourceNetworkRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkClient := config.SandwichClient.Network()

	network, err := networkClient.Get(d.Id())
	if err != nil {
		if apiError, ok := err.(api.APIErrorInterface); ok {
			if apiError.IsNotFound() {
				d.SetId("")
				return nil
			}
		}
		return err
	}

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

func resourceNetworkDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkClient := config.SandwichClient.Network()

	err := networkClient.Delete(d.Id())
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
		Refresh:    NetworkRefreshFunc(networkClient, d.Id()),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}
	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for network (%s) to delete: %s", d.Id(), err)
	}

	d.SetId("")
	return nil
}

func NetworkRefreshFunc(networkClient client.NetworkClientInterface, networkName string) func() (result interface{}, state string, err error) {
	return func() (result interface{}, state string, err error) {
		network, err := networkClient.Get(networkName)
		if err != nil {
			if apiError, ok := err.(api.APIErrorInterface); ok {
				if apiError.IsNotFound() {
					return network, "Deleted", nil
				}
			}
			return nil, "", err
		}
		return network, network.State, nil
	}
}
