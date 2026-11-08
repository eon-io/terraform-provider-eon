package provider

import (
	"testing"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func newTestAwsSubnet(id, vpc, az, name, cidr string) externalEonSdkAPI.ListSubnetsResponseV2SubnetsInner {
	subnet := externalEonSdkAPI.NewListSubnetsResponseV2SubnetsInner(id, vpc, az)
	if name != "" {
		subnet.SetName(name)
	}
	if cidr != "" {
		subnet.SetCidrBlock(cidr)
	}
	return *subnet
}

func newTestAwsVpc(id, name string) externalEonSdkAPI.ListAwsVpcsResponseVpcsInner {
	vpc := externalEonSdkAPI.NewListAwsVpcsResponseVpcsInner(id)
	if name != "" {
		vpc.SetName(name)
	}
	return *vpc
}

func TestAccRestoreAccountRegionAwsSubnets(t *testing.T) {
	testAccPreCheck(t)

	server := newFakeEonServer(t)
	server.SetAwsSubnets("acct-1", "us-east-1", externalEonSdkAPI.NewListSubnetsResponseV2(
		[]externalEonSdkAPI.ListSubnetsResponseV2SubnetsInner{
			newTestAwsSubnet("subnet-1", "vpc-1", "us-east-1a", "private-a", "10.0.0.0/24"),
			newTestAwsSubnet("subnet-2", "vpc-1", "us-east-1b", "", ""),
		},
	))
	server.SetAwsSubnets("acct-1", "eu-west-1", externalEonSdkAPI.NewListSubnetsResponseV2(
		[]externalEonSdkAPI.ListSubnetsResponseV2SubnetsInner{
			newTestAwsSubnet("subnet-eu", "vpc-eu", "eu-west-1a", "eu-private", "10.1.0.0/24"),
		},
	))

	dataSourceName := "data.eon_restore_account_region_aws_subnets.demo"
	config := func(region string) string {
		return server.providerConfig() + `
data "eon_restore_account_region_aws_subnets" "demo" {
  account_id = "acct-1"
  region     = "` + region + `"
}

output "first_subnet_id" {
  value = try(data.eon_restore_account_region_aws_subnets.demo.subnets[0].id, null)
}
`
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config("us-east-1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "account_id", "acct-1"),
					resource.TestCheckResourceAttr(dataSourceName, "region", "us-east-1"),
					resource.TestCheckResourceAttr(dataSourceName, "subnets.#", "2"),
					resource.TestCheckResourceAttr(dataSourceName, "subnets.0.id", "subnet-1"),
					resource.TestCheckResourceAttr(dataSourceName, "subnets.0.vpc", "vpc-1"),
					resource.TestCheckResourceAttr(dataSourceName, "subnets.0.availability_zone", "us-east-1a"),
					resource.TestCheckResourceAttr(dataSourceName, "subnets.0.name", "private-a"),
					resource.TestCheckResourceAttr(dataSourceName, "subnets.0.cidr_block", "10.0.0.0/24"),
					resource.TestCheckResourceAttr(dataSourceName, "subnets.1.id", "subnet-2"),
					resource.TestCheckNoResourceAttr(dataSourceName, "subnets.1.name"),
					resource.TestCheckNoResourceAttr(dataSourceName, "subnets.1.cidr_block"),
					resource.TestCheckOutput("first_subnet_id", "subnet-1"),
				),
			},
			{
				Config: config("eu-west-1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "region", "eu-west-1"),
					resource.TestCheckResourceAttr(dataSourceName, "subnets.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "subnets.0.id", "subnet-eu"),
					resource.TestCheckResourceAttr(dataSourceName, "subnets.0.availability_zone", "eu-west-1a"),
					resource.TestCheckOutput("first_subnet_id", "subnet-eu"),
				),
			},
			{
				PreConfig: func() {
					server.DeleteAwsSubnets("acct-1", "eu-west-1")
				},
				Config:             config("eu-west-1"),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{
				Config: config("eu-west-1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "subnets.#", "0"),
					resource.TestCheckNoResourceAttr(dataSourceName, "subnets.0.id"),
				),
			},
		},
	})
}

func TestAccRestoreAccountRegionAwsVpcs(t *testing.T) {
	testAccPreCheck(t)

	server := newFakeEonServer(t)
	server.SetAwsVpcs("acct-1", "us-east-1", externalEonSdkAPI.NewListAwsVpcsResponse(
		[]externalEonSdkAPI.ListAwsVpcsResponseVpcsInner{
			newTestAwsVpc("vpc-1", "main"),
			newTestAwsVpc("vpc-2", ""),
		},
	))

	dataSourceName := "data.eon_restore_account_region_aws_vpcs.demo"
	config := server.providerConfig() + `
data "eon_restore_account_region_aws_vpcs" "demo" {
  account_id = "acct-1"
  region     = "us-east-1"
}

output "vpc_ids" {
  value = join(",", [for v in data.eon_restore_account_region_aws_vpcs.demo.vpcs : v.id])
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "account_id", "acct-1"),
					resource.TestCheckResourceAttr(dataSourceName, "region", "us-east-1"),
					resource.TestCheckResourceAttr(dataSourceName, "vpcs.#", "2"),
					resource.TestCheckResourceAttr(dataSourceName, "vpcs.0.id", "vpc-1"),
					resource.TestCheckResourceAttr(dataSourceName, "vpcs.0.name", "main"),
					resource.TestCheckResourceAttr(dataSourceName, "vpcs.1.id", "vpc-2"),
					resource.TestCheckNoResourceAttr(dataSourceName, "vpcs.1.name"),
					resource.TestCheckOutput("vpc_ids", "vpc-1,vpc-2"),
				),
			},
			{
				PreConfig: func() {
					server.SetAwsVpcs("acct-1", "us-east-1", externalEonSdkAPI.NewListAwsVpcsResponse(
						[]externalEonSdkAPI.ListAwsVpcsResponseVpcsInner{
							newTestAwsVpc("vpc-1", "main-renamed"),
						},
					))
				},
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "vpcs.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "vpcs.0.name", "main-renamed"),
					resource.TestCheckOutput("vpc_ids", "vpc-1"),
				),
			},
			{
				PreConfig: func() {
					server.DeleteAwsVpcs("acct-1", "us-east-1")
				},
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "vpcs.#", "0"),
					resource.TestCheckOutput("vpc_ids", ""),
				),
			},
		},
	})
}

func TestAccRestoreAccountRegionEncryptionKeys(t *testing.T) {
	testAccPreCheck(t)

	server := newFakeEonServer(t)
	server.SetEncryptionKeys("acct-1", "us-east-1", externalEonSdkAPI.NewEncryptionKeys(
		[]externalEonSdkAPI.EncryptionKey{
			*externalEonSdkAPI.NewEncryptionKey("key-1", "arn:aws:kms:us-east-1:123456789012:key/key-1", "alias/restore"),
			*externalEonSdkAPI.NewEncryptionKey("key-aws", "arn:aws:kms:us-east-1:123456789012:key/key-aws", "alias/aws/ebs"),
		},
	))

	dataSourceName := "data.eon_restore_account_region_encryption_keys.demo"
	baseConfig := server.providerConfig() + `
data "eon_restore_account_region_encryption_keys" "demo" {
  account_id = "acct-1"
  region     = "us-east-1"
}

output "key_count" {
  value = length(data.eon_restore_account_region_encryption_keys.demo.keys)
}
`
	withAwsManagedConfig := server.providerConfig() + `
data "eon_restore_account_region_encryption_keys" "demo" {
  account_id               = "acct-1"
  region                   = "us-east-1"
  include_aws_managed_keys = true
}

output "key_count" {
  value = length(data.eon_restore_account_region_encryption_keys.demo.keys)
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: baseConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "account_id", "acct-1"),
					resource.TestCheckResourceAttr(dataSourceName, "region", "us-east-1"),
					resource.TestCheckNoResourceAttr(dataSourceName, "include_aws_managed_keys"),
					resource.TestCheckResourceAttr(dataSourceName, "keys.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "keys.0.id", "key-1"),
					resource.TestCheckResourceAttr(dataSourceName, "keys.0.arn", "arn:aws:kms:us-east-1:123456789012:key/key-1"),
					resource.TestCheckResourceAttr(dataSourceName, "keys.0.alias_name", "alias/restore"),
					resource.TestCheckOutput("key_count", "1"),
				),
			},
			{
				Config: withAwsManagedConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "include_aws_managed_keys", "true"),
					resource.TestCheckResourceAttr(dataSourceName, "keys.#", "2"),
					resource.TestCheckResourceAttr(dataSourceName, "keys.1.id", "key-aws"),
					resource.TestCheckResourceAttr(dataSourceName, "keys.1.alias_name", "alias/aws/ebs"),
					resource.TestCheckOutput("key_count", "2"),
				),
			},
			{
				PreConfig: func() {
					server.DeleteEncryptionKeys("acct-1", "us-east-1")
				},
				Config:             withAwsManagedConfig,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{
				Config: withAwsManagedConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "keys.#", "0"),
					resource.TestCheckOutput("key_count", "0"),
				),
			},
		},
	})
}
