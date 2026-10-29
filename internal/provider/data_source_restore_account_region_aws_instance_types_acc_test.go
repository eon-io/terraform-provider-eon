package provider

import (
	"testing"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRestoreAccountRegionAwsInstanceTypes(t *testing.T) {
	testAccPreCheck(t)

	server := newFakeEonServer(t)
	instanceTypesResp := externalEonSdkAPI.NewListInstanceTypesResponse()
	instanceTypesResp.SetInstanceTypes([]string{"m5.large", "m5.xlarge"})
	instanceTypesResp.SetInstantRestoreMapping(map[string]string{"m5.large": "i3.large"})
	server.SetAwsInstanceTypes("racc-1", "us-east-1", instanceTypesResp)

	dataSourceName := "data.eon_restore_account_region_aws_instance_types.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: server.providerConfig() + `
data "eon_restore_account_region_aws_instance_types" "test" {
  restore_account_id = "racc-1"
  region              = "us-east-1"
  subnet_id           = "subnet-123"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "restore_account_id", "racc-1"),
					resource.TestCheckResourceAttr(dataSourceName, "region", "us-east-1"),
					resource.TestCheckResourceAttr(dataSourceName, "subnet_id", "subnet-123"),
					resource.TestCheckResourceAttr(dataSourceName, "instance_types.#", "2"),
					resource.TestCheckResourceAttr(dataSourceName, "instance_types.0", "m5.large"),
					resource.TestCheckResourceAttr(dataSourceName, "instance_types.1", "m5.xlarge"),
					resource.TestCheckResourceAttr(dataSourceName, "instant_restore_mapping.m5.large", "i3.large"),
				),
			},
			{
				Config: server.providerConfig() + `
data "eon_restore_account_region_aws_instance_types" "test" {
  restore_account_id = "racc-1"
  region              = "us-east-1"
  subnet_id           = "subnet-123"
  volume_count        = 2
  total_volume_bytes  = 1073741824
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "volume_count", "2"),
					resource.TestCheckResourceAttr(dataSourceName, "total_volume_bytes", "1073741824"),
					resource.TestCheckResourceAttr(dataSourceName, "instance_types.#", "2"),
				),
			},
		},
	})
}
