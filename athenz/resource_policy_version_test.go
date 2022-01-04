package athenz

import (
	"fmt"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/AthenZ/terraform-provider-athenz/client"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccGroupPolicyVersionBasic(t *testing.T) {
	t.Skip("Skipping testing until docker image will be updated")
	if v := os.Getenv("TF_ACC"); v != "1" && v != "true" {
		log.Print("TF_ACC must be set for acceptance tests")
		return
	}
	if v := os.Getenv("DOMAIN"); v == "" {
		t.Fatal("DOMAIN must be set for acceptance tests")
	}
	resName := "athenz_policy_version.policy_version_test"
	rInt := acctest.RandInt()
	domainName := os.Getenv("DOMAIN")
	name := fmt.Sprintf("test%d", rInt)
	version1 := "test_version_1"
	version2 := "test_version_2"
	version3 := "test_version_3"
	role1 := "acctest_role1"
	role2 := "acctest_role2"
	role3 := "acctest_role3"
	resourceRole1 := fmt.Sprintf(`resource "athenz_role" "%s" {
  			name = "%s"
  			domain = "%s"
		}`, role1, role1, domainName)
	resourceRole2 := fmt.Sprintf(`resource "athenz_role" "%s" {
  			name = "%s"
  			domain = "%s"
		}`, role2, role2, domainName)
	resourceRole3 := fmt.Sprintf(`resource "athenz_role" "%s" {
  			name = "%s"
  			domain = "%s"
		}`, role3, role3, domainName)
	t.Cleanup(func() {
		cleanAllAccTestPoliciesVersion(domainName, []string{name}, []string{role1, role2, role3})
	})
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccCheckGroupPolicyVersionsDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGroupPolicyVersionConfigBasic(name, domainName, version1, version1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupPolicyVersionsExists(resName, []string{version1}),
					resource.TestCheckResourceAttr(resName, "name", name),
					resource.TestCheckResourceAttr(resName, "active_version", version1),
					resource.TestCheckResourceAttr(resName, "versions.#", "1"),
				),
			},
			{
				Config: testAccGroupPolicyVersionConfigAddAssertion(resourceRole1, name, domainName, version1, version1, role1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupPolicyVersionsExists(resName, []string{version1}),
					resource.TestCheckResourceAttr(resName, "name", name),
					resource.TestCheckResourceAttr(resName, "active_version", version1),
					resource.TestCheckResourceAttr(resName, "versions.#", "1"),
				),
			},
			{
				Config: testAccGroupPolicyVersionConfigAddNonActiveVersion(resourceRole1, resourceRole2, name, domainName, version1, version1, version2, role1, role2),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupPolicyVersionsExists(resName, []string{version1, version2}),
					resource.TestCheckResourceAttr(resName, "name", name),
					resource.TestCheckResourceAttr(resName, "active_version", version1),
					resource.TestCheckResourceAttr(resName, "versions.#", "2"),
				),
			},
			{
				Config: testAccGroupPolicyVersionConfigChangeActiveVersion(resourceRole1, resourceRole2, name, domainName, version2, version1, version2, role1, role2),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupPolicyVersionsExists(resName, []string{version1, version2}),
					resource.TestCheckResourceAttr(resName, "name", name),
					resource.TestCheckResourceAttr(resName, "active_version", version2),
					resource.TestCheckResourceAttr(resName, "versions.#", "2"),
				),
			},
			{
				Config: testAccGroupPolicyVersionConfigAddActiveVersion(resourceRole1, resourceRole2, resourceRole3, name, domainName, version3, version1, version2, version3, role1, role2, role3),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupPolicyVersionsExists(resName, []string{version1, version2, version3}),
					resource.TestCheckResourceAttr(resName, "name", name),
					resource.TestCheckResourceAttr(resName, "active_version", version3),
					resource.TestCheckResourceAttr(resName, "versions.#", "3"),
					resource.TestCheckResourceAttr(resName, "audit_ref", AUDIT_REF),
				),
			},
			{
				Config: testAccGroupPolicyVersionConfigRemoveNonActiveVersion(resourceRole1, resourceRole3, name, domainName, version3, version1, version3, role1, role3),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupPolicyVersionsExists(resName, []string{version1, version3}),
					resource.TestCheckResourceAttr(resName, "name", name),
					resource.TestCheckResourceAttr(resName, "active_version", version3),
					resource.TestCheckResourceAttr(resName, "versions.#", "2"),
				),
			},
			{
				Config: testAccGroupPolicyVersionConfigRemovePreviousActiveVersion(resourceRole1, name, domainName, version1, version1, role1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupPolicyVersionsExists(resName, []string{version1}),
					resource.TestCheckResourceAttr(resName, "name", name),
					resource.TestCheckResourceAttr(resName, "active_version", version1),
					resource.TestCheckResourceAttr(resName, "versions.#", "1"),
					resource.TestCheckResourceAttr(resName, "audit_ref", AUDIT_REF),
				),
			},
		},
	})
}

func cleanAllAccTestPoliciesVersion(domain string, policies []string, roles []string) {
	zmsClient := testAccProvider.Meta().(client.ZmsClient)
	for _, policyName := range policies {
		_, err := zmsClient.GetPolicy(domain, policyName)
		if err == nil {
			if err = zmsClient.DeletePolicy(domain, policyName, AUDIT_REF); err != nil {
				log.Printf("error deleting Policy %s: %s", policyName, err)
			}
		}
	}
	for _, roleName := range roles {
		_, err := zmsClient.GetRole(domain, roleName)
		if err == nil {
			if err = zmsClient.DeleteRole(domain, roleName, AUDIT_REF); err != nil {
				log.Printf("error deleting Role %s: %s", roleName, err)
			}
		}
	}
}

func testAccCheckGroupPolicyVersionsExists(resourceName string, policyVersionNames []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no Athenz Policy ID is set")
		}

		fullResourceName := strings.Split(rs.Primary.ID, POLICY_SEPARATOR)
		dn, pn := fullResourceName[0], fullResourceName[1]

		zmsClient := testAccProvider.Meta().(client.ZmsClient)
		for _, versionName := range policyVersionNames {
			_, err := zmsClient.GetPolicyVersion(dn, pn, versionName)
			if err != nil {
				return err
			}
		}
		return nil
	}
}

func testAccCheckGroupPolicyVersionsDestroy(s *terraform.State) error {
	zmsClient := testAccProvider.Meta().(client.ZmsClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "athenz_policy_version" {
			continue
		}

		fullResourceName := strings.Split(rs.Primary.ID, POLICY_SEPARATOR)
		dn, pn := fullResourceName[0], fullResourceName[1]

		_, err := zmsClient.GetPolicy(dn, pn)

		if err == nil {
			return fmt.Errorf("athenz Policy still exists")
		}
	}

	return nil
}

func testAccGroupPolicyVersionConfigBasic(name, domain, activeVersion, version1 string) string {
	return fmt.Sprintf(`
resource "athenz_policy_version" "policy_version_test" {
name = "%s"
domain = "%s"
active_version = "%s"
versions{
	version_name = "%s"
	}
}`, name, domain, activeVersion, version1)
}
func testAccGroupPolicyVersionConfigAddAssertion(role1, name, domain, activeVersion, version1, resource1Name string) string {
	return fmt.Sprintf(`
%s
resource "athenz_policy_version" "policy_version_test" {
name = "%s"
domain = "%s"
active_version = "%s"
versions {
  version_name = "%s"
  assertion = [{
    effect = "ALLOW"
    action = "*"
    role = "${athenz_role.%s.name}"
    resource = "mendi_resource1"
  }]
}
}`, role1, name, domain, activeVersion, version1, resource1Name)
}
func testAccGroupPolicyVersionConfigAddNonActiveVersion(role1, role2, name, domain, activeVersion, version1, version2, resource1Name, resource2Name string) string {
	return fmt.Sprintf(`
%s
%s
resource "athenz_policy_version" "policy_version_test" {
name = "%s"
domain = "%s"
active_version = "%s"
versions {
  version_name = "%s"
  assertion = [{
    effect = "ALLOW"
    action = "*"
    role="${athenz_role.%s.name}"
    resource = "mendi_resource1"
  }]
}

versions {
  version_name = "%s"
  assertion  = [{
    effect = "ALLOW"
    action = "*"
	role="${athenz_role.%s.name}"
    resource = "mendi_resource2"
  },
	{
	role="${athenz_role.%s.name}"
    effect = "DENY"
    action = "play"
    resource = "mendi_resource2"
  }]
}
}
`, role1, role2, name, domain, activeVersion, version1, resource1Name, version2, resource2Name, resource2Name)
}

func testAccGroupPolicyVersionConfigChangeActiveVersion(role1, role2, name, domain, activeVersion, version1, version2, resource1Name, resource2Name string) string {
	return fmt.Sprintf(`
%s
%s
resource "athenz_policy_version" "policy_version_test" {
name = "%s"
domain = "%s"
active_version = "%s"
versions {
 version_name = "%s"
 assertion = [{
	effect = "ALLOW"
	action = "*"
	role="${athenz_role.%s.name}"
	resource = "mendi_resource1"
 }]
}

versions {
 version_name = "%s"
 assertion  = [{
	effect = "ALLOW"
	action = "*"
	role="${athenz_role.%s.name}"
	resource = "mendi_resource2"
 },
 {
	role="${athenz_role.%s.name}"
	effect = "DENY"
	action = "play"
	resource = "mendi_resource2"
 }]
}
}
`, role1, role2, name, domain, activeVersion, version1, resource1Name, version2, resource2Name, resource2Name)
}

func testAccGroupPolicyVersionConfigAddActiveVersion(role1, role2, role3, name, domain, activeVersion, version1, version2, version3, resource1Name, resource2Name, resource3Name string) string {
	return fmt.Sprintf(`
%s
%s
%s
resource "athenz_policy_version" "policy_version_test" {
name = "%s"
domain = "%s"
active_version = "%s"
versions = [
  {
    version_name = "%s"
    assertion = [
      {
        effect = "ALLOW"
        action = "*"
	    role="${athenz_role.%s.name}"
        resource = "mendi_resource1"
      }]
  },
  {
    version_name = "%s"
    assertion = [
      {
        effect = "ALLOW"
        action = "*"
	    role="${athenz_role.%s.name}"
        resource = "mendi_resource2"
      },
      {
	     role="${athenz_role.%s.name}"
		 effect = "DENY"
		 action = "play"
		 resource = "mendi_resource2"
      }
	]
  },
  {
    version_name = "%s"
    assertion = [
      {
        effect = "ALLOW"
        action = "*"
	    role="${athenz_role.%s.name}"
        resource = "mendi_resource3"
      },
      {
	    role="${athenz_role.%s.name}"
		effect = "DENY"
        action = "play"
        resource = "mendi_resource3"
      }]
  }
]
}
`, role1, role2, role3, name, domain, activeVersion, version1, resource1Name, version2, resource2Name, resource2Name, version3, resource3Name, resource3Name)
}

func testAccGroupPolicyVersionConfigRemoveNonActiveVersion(role1, role3, name, domain, activeVersion, version1, version3, resource1Name, resource3Name string) string {
	return fmt.Sprintf(`
%s
%s
resource "athenz_policy_version" "policy_version_test" {
name = "%s"
domain = "%s"
active_version = "%s"
versions = [
  {
    version_name = "%s"
    assertion = [
      {
        effect = "ALLOW"
        action = "*"
	    role="${athenz_role.%s.name}"
        resource = "mendi_resource1"
      }]
  },
  {
    version_name = "%s"
    assertion = [
      {
        effect = "ALLOW"
        action = "*"
	    role="${athenz_role.%s.name}"
        resource = "mendi_resource3"
      },
      {
	    role="${athenz_role.%s.name}"
        effect = "DENY"
        action = "play"
        resource = "mendi_resource3"
      }]
  }
]
}
`, role1, role3, name, domain, activeVersion, version1, resource1Name, version3, resource3Name, resource3Name)
}

func testAccGroupPolicyVersionConfigRemovePreviousActiveVersion(role1, name, domain, activeVersion, version1, resource1Name string) string {
	return fmt.Sprintf(`
%s
resource "athenz_policy_version" "policy_version_test" {
name = "%s"
domain = "%s"
active_version = "%s"
versions = [
  {
    version_name = "%s"
    assertion = [
      {
        effect = "ALLOW"
        action = "*"
	    role="${athenz_role.%s.name}"
        resource = "mendi_resource1"
      }]
  }
]
}
`, role1, name, domain, activeVersion, version1, resource1Name)
}
