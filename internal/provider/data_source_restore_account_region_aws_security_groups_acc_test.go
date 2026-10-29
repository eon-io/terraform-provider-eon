package provider

import (
	"testing"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRestoreAccountRegionAwsSecurityGroups(t *testing.T) {
	testAccPreCheck(t)

	server := newFakeEonServer(t)
	server.SetAwsSecurityGroups("racc-1", "us-east-1", externalEonSdkAPI.NewRestoreSecurityGroups(
		"us-east-1",
		[]externalEonSdkAPI.RestoreSecurityGroupsSecurityGroupsInner{
			*externalEonSdkAPI.NewRestoreSecurityGroupsSecurityGroupsInner("default", "sg-1", "vpc-1"),
			*externalEonSdkAPI.NewRestoreSecurityGroupsSecurityGroupsInner("other", "sg-2", "vpc-2"),
		},
	))

	dataSourceName := "data.eon_restore_account_region_aws_security_groups.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: server.providerConfig() + `
data "eon_restore_account_region_aws_security_groups" "test" {
  restore_account_id = "racc-1"
  region              = "us-east-1"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "restore_account_id", "racc-1"),
					resource.TestCheckResourceAttr(dataSourceName, "region", "us-east-1"),
					resource.TestCheckResourceAttr(dataSourceName, "security_groups.#", "2"),
					resource.TestCheckResourceAttr(dataSourceName, "security_groups.0.id", "sg-1"),
					resource.TestCheckResourceAttr(dataSourceName, "security_groups.0.name", "default"),
					resource.TestCheckResourceAttr(dataSourceName, "security_groups.0.vpc", "vpc-1"),
					resource.TestCheckResourceAttr(dataSourceName, "security_groups.1.id", "sg-2"),
				),
			},
			{
				Config: server.providerConfig() + `
data "eon_restore_account_region_aws_security_groups" "test" {
  restore_account_id = "racc-1"
  region              = "us-east-1"
  vpc_id              = "vpc-1"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "vpc_id", "vpc-1"),
					resource.TestCheckResourceAttr(dataSourceName, "security_groups.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "security_groups.0.id", "sg-1"),
				),
			},
		},
	})
}
