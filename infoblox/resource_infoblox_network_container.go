package infoblox

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ibclient "github.com/infobloxopen/infoblox-go-client/v2"
)

var (
	netContainerIPv4Regexp = regexp.MustCompile("^networkcontainer/.+")
	netContainerIPv6Regexp = regexp.MustCompile("^ipv6networkcontainer/.+")
)

func resourceNetworkContainer() *schema.Resource {
	return &schema.Resource{
		Importer: &schema.ResourceImporter{},

		Schema: map[string]*schema.Schema{
			"network_view": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     defaultNetView,
				Description: "The name of network view for the network container.",
			},
			"cidr": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The network container's address, in CIDR format.",
			},
			"comment": {
				Type:        schema.TypeString,
				Default:     "",
				Optional:    true,
				Description: "A description of the network container.",
			},
			"ext_attrs": {
				Type:        schema.TypeString,
				Default:     "",
				Optional:    true,
				Description: "The Extensible attributes of the network container to be added/updated, as a map in JSON format",
			},
		},
	}
}

func resourceNetworkContainerCreate(d *schema.ResourceData, m interface{}, isIPv6 bool) error {
	nvName := d.Get("network_view").(string)
	cidr := d.Get("cidr").(string)
	comment := d.Get("comment").(string)

	extAttrJSON := d.Get("ext_attrs").(string)
	extAttrs := make(map[string]interface{})
	if extAttrJSON != "" {
		if err := json.Unmarshal([]byte(extAttrJSON), &extAttrs); err != nil {
			return fmt.Errorf("cannot process 'ext_attrs' field: %s", err.Error())
		}
	}
	var tenantID string
	for attrName, attrValueInf := range extAttrs {
		attrValue, _ := attrValueInf.(string)
		if attrName == eaNameForTenantId {
			tenantID = attrValue
			break
		}
	}

	if cidr == "" || nvName == "" {
		return fmt.Errorf(
			"Tenant ID, network view's name and CIDR are required to create a network container")
	}

	connector := m.(ibclient.IBConnector)
	objMgr := ibclient.NewObjectManager(connector, "Terraform", tenantID)
	nc, err := objMgr.CreateNetworkContainer(nvName, cidr, isIPv6, comment, extAttrs)
	if err != nil {
		return fmt.Errorf(
			"creation of IPv6 network container block failed in network view '%s': %s",
			nvName, err.Error())
	}
	d.SetId(nc.Ref)

	return nil
}

func resourceNetworkContainerRead(d *schema.ResourceData, m interface{}) error {
	extAttrJSON := d.Get("ext_attrs").(string)
	extAttrs := make(map[string]interface{})
	if extAttrJSON != "" {
		if err := json.Unmarshal([]byte(extAttrJSON), &extAttrs); err != nil {
			return fmt.Errorf("cannot process 'ext_attrs' field: %s", err.Error())
		}
	}
	var tenantID string
	tempVal, found := extAttrs[eaNameForTenantId]
	if found {
		tenantID = tempVal.(string)
	}

	connector := m.(ibclient.IBConnector)
	objMgr := ibclient.NewObjectManager(connector, "Terraform", tenantID)

	obj, err := objMgr.GetNetworkContainerByRef(d.Id())
	if err != nil {
		return fmt.Errorf("failed to retrieve network container: %s", err.Error())
	}

	if obj.Ea != nil && len(obj.Ea) > 0 {
		// TODO: temporary scaffold, need to rework marshalling/unmarshalling of EAs
		//       (avoiding additional layer of keys ("value" key)
		eaMap := (map[string]interface{})(obj.Ea)
		ea, err := json.Marshal(eaMap)
		if err != nil {
			return err
		}
		if err = d.Set("ext_attrs", string(ea)); err != nil {
			return err
		}
	}

	if err = d.Set("comment", obj.Comment); err != nil {
		return err
	}

	if err = d.Set("network_view", obj.NetviewName); err != nil {
		return err
	}

	if err = d.Set("cidr", obj.Cidr); err != nil {
		return err
	}

	d.SetId(obj.Ref)

	return nil
}

func resourceNetworkContainerUpdate(d *schema.ResourceData, m interface{}) error {
	var updateSuccessful bool
	defer func() {
		// Reverting the state back, in case of a failure,
		// otherwise Terraform will keep the values, which leaded to the failure,
		// in the state file.
		if !updateSuccessful {
			prevNetView, _ := d.GetChange("network_view")
			prevCIDR, _ := d.GetChange("cidr")
			prevComment, _ := d.GetChange("comment")
			prevEa, _ := d.GetChange("ext_attrs")

			_ = d.Set("network_view", prevNetView.(string))
			_ = d.Set("cidr", prevCIDR.(string))
			_ = d.Set("comment", prevComment.(string))
			_ = d.Set("ext_attrs", prevEa.(string))
		}
	}()

	nvName := d.Get("network_view").(string)
	if d.HasChange("network_view") {
		return fmt.Errorf("changing the value of 'network_view' field is not allowed")
	}
	cidr := d.Get("cidr").(string)
	extAttrJSON := d.Get("ext_attrs").(string)
	extAttrs := make(map[string]interface{})
	if extAttrJSON != "" {
		if err := json.Unmarshal([]byte(extAttrJSON), &extAttrs); err != nil {
			return fmt.Errorf("cannot process 'ext_attrs' field: %s", err.Error())
		}
	}

	var tenantID string
	tempVal, found := extAttrs[eaNameForTenantId]
	if found {
		tenantID = tempVal.(string)
	}

	if cidr == "" || nvName == "" {
		return fmt.Errorf(
			"Tenant ID, network view's name and CIDR are required to update a network container")
	}

	connector := m.(ibclient.IBConnector)
	objMgr := ibclient.NewObjectManager(connector, "Terraform", tenantID)

	comment := ""
	commentText, commentFieldFound := d.GetOk("comment")
	if commentFieldFound {
		comment = commentText.(string)
	}

	nc, err := objMgr.UpdateNetworkContainer(d.Id(), extAttrs, comment)
	if err != nil {
		return fmt.Errorf(
			"failed to update the network container in network view '%s': %s",
			nvName, err.Error())
	}
	updateSuccessful = true
	d.SetId(nc.Ref)

	return nil
}

func resourceNetworkContainerDelete(d *schema.ResourceData, m interface{}) error {
	extAttrJSON := d.Get("ext_attrs").(string)
	extAttrs := make(map[string]interface{})
	if extAttrJSON != "" {
		if err := json.Unmarshal([]byte(extAttrJSON), &extAttrs); err != nil {
			return fmt.Errorf("cannot process 'ext_attrs' field: %s", err.Error())
		}
	}
	var tenantID string
	for attrName, attrValueInf := range extAttrs {
		attrValue, _ := attrValueInf.(string)
		if attrName == eaNameForTenantId {
			tenantID = attrValue
			break
		}
	}
	connector := m.(ibclient.IBConnector)
	objMgr := ibclient.NewObjectManager(connector, "Terraform", tenantID)

	if _, err := objMgr.DeleteNetworkContainer(d.Id()); err != nil {
		return fmt.Errorf(
			"deletion of the network container failed: %s", err.Error())
	}

	return nil
}

// TODO: implement this after infoblox-go-client refactoring
//func resourceNetworkContainerExists(d *schema.ResourceData, m interface{}, isIPv6 bool) (bool, error) {
//	return false, nil
//}

func resourceIPv4NetworkContainerCreate(d *schema.ResourceData, m interface{}) error {
	return resourceNetworkContainerCreate(d, m, false)
}

func resourceIPv4NetworkContainerRead(d *schema.ResourceData, m interface{}) error {
	ref := d.Id()
	if !netContainerIPv4Regexp.MatchString(ref) {
		return fmt.Errorf("reference '%s' for 'networkcontainer' object has an invalid format", ref)
	}

	return resourceNetworkContainerRead(d, m)
}

func resourceIPv4NetworkContainerUpdate(d *schema.ResourceData, m interface{}) error {
	return resourceNetworkContainerUpdate(d, m)
}

func resourceIPv4NetworkContainerDelete(d *schema.ResourceData, m interface{}) error {
	return resourceNetworkContainerDelete(d, m)
}

//func resourceIPv4NetworkContainerExists(d *schema.ResourceData, m interface{}) (bool, error) {
//	return resourceNetworkContainerExists(d, m, false)
//}

func resourceIPv4NetworkContainer() *schema.Resource {
	nc := resourceNetworkContainer()
	nc.Create = resourceIPv4NetworkContainerCreate
	nc.Read = resourceIPv4NetworkContainerRead
	nc.Update = resourceIPv4NetworkContainerUpdate
	nc.Delete = resourceIPv4NetworkContainerDelete
	//nc.Exists = resourceIPv4NetworkContainerExists

	return nc
}

func resourceIPv6NetworkContainerCreate(d *schema.ResourceData, m interface{}) error {
	return resourceNetworkContainerCreate(d, m, true)
}

func resourceIPv6NetworkContainerRead(d *schema.ResourceData, m interface{}) error {
	ref := d.Id()
	if !netContainerIPv6Regexp.MatchString(ref) {
		return fmt.Errorf("reference '%s' for 'ipv6networkcontainer' object has an invalid format", ref)
	}

	return resourceNetworkContainerRead(d, m)
}

func resourceIPv6NetworkContainerUpdate(d *schema.ResourceData, m interface{}) error {
	return resourceNetworkContainerUpdate(d, m)
}

func resourceIPv6NetworkContainerDelete(d *schema.ResourceData, m interface{}) error {
	return resourceNetworkContainerDelete(d, m)
}

//func resourceIPv6NetworkContainerExists(d *schema.ResourceData, m interface{}) (bool, error) {
//	return resourceNetworkContainerExists(d, m, true)
//}

func resourceIPv6NetworkContainer() *schema.Resource {
	nc := resourceNetworkContainer()
	nc.Create = resourceIPv6NetworkContainerCreate
	nc.Read = resourceIPv6NetworkContainerRead
	nc.Update = resourceIPv6NetworkContainerUpdate
	nc.Delete = resourceIPv6NetworkContainerDelete
	//nc.Exists = resourceIPv6NetworkContainerExists

	return nc
}
