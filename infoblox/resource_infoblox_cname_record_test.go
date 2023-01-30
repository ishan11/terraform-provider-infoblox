package infoblox

import (
	"fmt"
	"github.com/infobloxopen/infoblox-go-client/v2/utils"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	ibclient "github.com/infobloxopen/infoblox-go-client/v2"
)

var testAccresourceCNAMERecordCreate = fmt.Sprintf(`
resource "infoblox_cname_record" "foo"{
	dns_view="default"
	canonical="test-canonicalName.test.com"
	alias="test-aliasname.test.com"
	comment="CNAME record created"
	ext_attrs = jsonencode({
		"Tenant ID" = "terraform_test_tenant"
		"Location" = "Test loc"
		"Site" = "Test site"
		"TestEA1"=["text1","text2"]
	  })

}`)

var testAccresourceCNAMERecordUpdate = fmt.Sprintf(`
resource "infoblox_cname_record" "foo"{
	dns_view="default"
	canonical="test-canonicalName.test.com"
	alias="test-aliasname.test.com"
	comment="CNAME record updated"
	ext_attrs = jsonencode({
		"Tenant ID" = "terraform_test_tenant"
		"Location" = "Test loc 2"
		"Site" = "Test site 2"
		"TestEA1"="text3"
	  })

}`)

func validateRecordCNAME(
	resourceName string,
	expectedValue *ibclient.RecordCNAME) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		res, found := s.RootModule().Resources[resourceName]
		if !found {
			return fmt.Errorf("not found: %s", resourceName)
		}

		id := res.Primary.ID
		if id == "" {
			return fmt.Errorf("ID is not set")
		}

		connector := testAccProvider.Meta().(ibclient.IBConnector)
		objMgr := ibclient.NewObjectManager(
			connector,
			"terraform_test",
			"terraform_test_tenant")
		recCNAME, err := objMgr.GetCNAMERecordByRef(id)
		if err != nil {
			if isNotFoundError(err) {
				if expectedValue == nil {
					return nil
				}
				return fmt.Errorf("object with ID '%s' not found, but expected to exist", id)
			}
		}

		expView := expectedValue.View
		if !reflect.DeepEqual(recCNAME.View, expView) {
			return fmt.Errorf(
				"the value of 'dns_view' field is '%s', but expected '%s'",
				safePtrValue(recCNAME.View), safePtrValue(expView))
		}

		expCanonical := expectedValue.Canonical
		if !reflect.DeepEqual(recCNAME.Canonical, expCanonical) {
			return fmt.Errorf(
				"the value of 'canonical' field is '%s', but expected '%s'",
				safePtrValue(recCNAME.Canonical), safePtrValue(expCanonical))
		}

		expName := expectedValue.Name
		if !reflect.DeepEqual(recCNAME.Name, expName) {
			return fmt.Errorf(
				"the value of 'alias Name' field is '%s', but expected '%s'",
				safePtrValue(recCNAME.Name), safePtrValue(expName))
		}

		if !reflect.DeepEqual(recCNAME.UseTtl, expectedValue.UseTtl) {
			return fmt.Errorf(
				"the value of 'use_ttl' field does not match: got '%s', expected '%s'",
				safePtrValue(recCNAME.UseTtl), safePtrValue(expectedValue.UseTtl))
		} else if recCNAME.UseTtl != nil && *recCNAME.UseTtl {
			if !reflect.DeepEqual(recCNAME.Ttl, expectedValue.Ttl) {
				return fmt.Errorf(
					"the value of 'ttl' field does not match: got '%s', expected '%s'",
					safePtrValue(recCNAME.Ttl), safePtrValue(expectedValue.Ttl))
			}
		}

		expComment := expectedValue.Comment
		if !reflect.DeepEqual(recCNAME.Comment, expComment) {
			return fmt.Errorf(
				"the value of 'comment' field is '%s', but expected '%s'",
				safePtrValue(recCNAME.Comment), safePtrValue(expComment))
		}

		// the rest is about extensible attributes
		expectedEAs := expectedValue.Ea
		if expectedEAs == nil && recCNAME.Ea != nil {
			return fmt.Errorf(
				"the object with ID '%s' has 'ext_attrs' field, but it is not expected to exist", id)
		}
		if expectedEAs != nil && recCNAME.Ea == nil {
			return fmt.Errorf(
				"the object with ID '%s' has no 'ext_attrs' field, but it is expected to exist", id)
		}
		if expectedEAs == nil {
			return nil
		}

		return validateEAs(recCNAME.Ea, expectedEAs)
	}
}

func TestAccResourceCNAMERecord(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCNAMERecordDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccresourceCNAMERecordCreate,
				Check: validateRecordCNAME(
					"infoblox_cname_record.foo",
					&ibclient.RecordCNAME{
						View:      utils.StringPtr("default"),
						Canonical: utils.StringPtr("test-canonicalName.test.com"),
						Name:      utils.StringPtr("test-aliasname.test.com"),
						Zone:      "test.com",
						Comment:   utils.StringPtr("CNAME record created"),
						UseTtl:    utils.BoolPtr(false),
						Ea: ibclient.EA{
							"Tenant ID": "terraform_test_tenant",
							"Location":  "Test loc",
							"Site":      "Test site",
							"TestEA1":   []string{"text1", "text2"},
						},
					},
				),
			},
			{
				Config: testAccresourceCNAMERecordUpdate,
				Check: validateRecordCNAME(
					"infoblox_cname_record.foo",
					&ibclient.RecordCNAME{
						View:      utils.StringPtr("default"),
						Canonical: utils.StringPtr("test-canonicalName.test.com"),
						Name:      utils.StringPtr("test-aliasname.test.com"),
						Zone:      "test.com",
						Comment:   utils.StringPtr("CNAME record updated"),
						UseTtl:    utils.BoolPtr(false),
						Ea: ibclient.EA{
							"Tenant ID": "terraform_test_tenant",
							"Location":  "Test loc 2",
							"Site":      "Test site 2",
							"TestEA1":   "text3",
						},
					},
				),
			},
		},
	})
}

func testAccCheckCNAMERecordDestroy(s *terraform.State) error {
	connector := testAccProvider.Meta().(ibclient.IBConnector)
	objMgr := ibclient.NewObjectManager(
		connector,
		"terraform_test",
		"terraform_test_tenant")
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "infoblox_cname_record" {
			continue
		}
		res, err := objMgr.GetCNAMERecordByRef(rs.Primary.ID)
		if err != nil {
			if isNotFoundError(err) {
				continue
			}
			return err
		}
		if res != nil {
			return fmt.Errorf("object with ID '%s' remains", rs.Primary.ID)
		}
	}
	return nil
}
