package infoblox

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	ibclient "github.com/infobloxopen/infoblox-go-client/v2"
	"github.com/infobloxopen/infoblox-go-client/v2/utils"
)

func testAccCheckNetworkViewRecordDestroy(s *terraform.State) error {
	meta := testAccProvider.Meta()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "infoblox_network_view" {
			continue
		}
		connector := meta.(ibclient.IBConnector)
		objMgr := ibclient.NewObjectManager(connector, "terraform_test", "test")
		rec, _ := objMgr.GetNetworkViewByRef(rs.Primary.ID)
		if rec != nil {
			return fmt.Errorf("record not found")
		}

	}
	return nil
}

func testAccNetworkViewCompare(t *testing.T, resPath string, expectedRec *ibclient.NetworkView) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		res, found := s.RootModule().Resources[resPath]
		if !found {
			return fmt.Errorf("not found: %s", resPath)
		}
		if res.Primary.ID == "" {
			return fmt.Errorf("ID is not set")
		}
		meta := testAccProvider.Meta()
		connector := meta.(ibclient.IBConnector)
		objMgr := ibclient.NewObjectManager(connector, "terraform_test", "test")

		netView, _ := objMgr.GetNetworkViewByRef(res.Primary.ID)
		if netView == nil {
			return fmt.Errorf("network view not found")
		}

		if !reflect.DeepEqual(netView.Name, expectedRec.Name) {
			return fmt.Errorf(
				"'name' does not match: got '%s', expected '%s'",
				safePtrValue(netView.Name), safePtrValue(expectedRec.Name))
		}
		if !reflect.DeepEqual(netView.Comment, expectedRec.Comment) {
			return fmt.Errorf(
				"'comment' does not match: got '%s', expected '%s'",
				safePtrValue(netView.Comment), safePtrValue(expectedRec.Comment))
		}
		return validateEAs(netView.Ea, expectedRec.Ea)
	}
}

func TestAccResourceNetworkView(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNetworkViewRecordDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "infoblox_network_view" "foo"{
						name = "testNetworkView"
						comment = "test comment 1"
						ext_attrs = jsonencode({
							"Tenant ID"="terraform_test_tenant"
							"Location"="Test loc"
							"Site"="Test site"
							"TestEA1"=["text1","text2"]
						})
					}`),
				Check: resource.ComposeTestCheckFunc(
					testAccNetworkViewCompare(t, "infoblox_network_view.foo", &ibclient.NetworkView{
						Name:    utils.StringPtr("testNetworkView"),
						Comment: utils.StringPtr("test comment 1"),
						Ea: ibclient.EA{
							"Tenant ID": "terraform_test_tenant",
							"Location":  "Test loc",
							"Site":      "Test site",
							"TestEA1":   []string{"text1", "text2"},
						},
					}),
				),
			},
		},
	})
}
