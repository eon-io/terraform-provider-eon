package provider

import (
	"testing"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRestoreAccountAwsS3Buckets(t *testing.T) {
	testAccPreCheck(t)

	server := newFakeEonServer(t)
	server.SetS3Buckets("acct-1", []externalEonSdkAPI.S3Bucket{
		{BucketName: "bucket-one", RegionName: "us-east-1"},
		{BucketName: "bucket-two", RegionName: "eu-west-1"},
	})

	dataSourceName := "data.eon_restore_account_aws_s3_buckets.demo"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: server.providerConfig() + `
data "eon_restore_account_aws_s3_buckets" "demo" {
  account_id = "acct-1"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "account_id", "acct-1"),
					resource.TestCheckResourceAttr(dataSourceName, "buckets.#", "2"),
					resource.TestCheckResourceAttr(dataSourceName, "buckets.0.bucket_name", "bucket-one"),
					resource.TestCheckResourceAttr(dataSourceName, "buckets.0.region_name", "us-east-1"),
					resource.TestCheckResourceAttr(dataSourceName, "buckets.1.bucket_name", "bucket-two"),
					resource.TestCheckResourceAttr(dataSourceName, "buckets.1.region_name", "eu-west-1"),
				),
			},
			{
				PreConfig: func() {
					server.SetS3Buckets("acct-1", nil)
				},
				Config: server.providerConfig() + `
data "eon_restore_account_aws_s3_buckets" "demo" {
  account_id = "acct-1"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "buckets.#", "0"),
				),
			},
		},
	})
}

func TestAccRestoreAccountRegionAwsInstanceTypes(t *testing.T) {
	testAccPreCheck(t)

	server := newFakeEonServer(t)
	resp := externalEonSdkAPI.NewListInstanceTypesResponse()
	resp.SetInstanceTypes([]string{"m5.large", "m5.xlarge"})
	resp.SetInstantRestoreMapping(map[string]string{"m5.large": "m5.xlarge"})
	server.SetAwsInstanceTypes("acct-1", "us-east-1", resp)

	dataSourceName := "data.eon_restore_account_region_aws_instance_types.demo"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: server.providerConfig() + `
data "eon_restore_account_region_aws_instance_types" "demo" {
  account_id = "acct-1"
  region     = "us-east-1"
  subnet_id  = "subnet-1"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "account_id", "acct-1"),
					resource.TestCheckResourceAttr(dataSourceName, "region", "us-east-1"),
					resource.TestCheckResourceAttr(dataSourceName, "subnet_id", "subnet-1"),
					resource.TestCheckResourceAttr(dataSourceName, "instance_types.#", "2"),
					resource.TestCheckResourceAttr(dataSourceName, "instance_types.0", "m5.large"),
					resource.TestCheckResourceAttr(dataSourceName, "instance_types.1", "m5.xlarge"),
					resource.TestCheckResourceAttr(dataSourceName, "instant_restore_mapping.m5.large", "m5.xlarge"),
				),
			},
			{
				Config: server.providerConfig() + `
data "eon_restore_account_region_aws_instance_types" "demo" {
  account_id          = "acct-1"
  region              = "us-east-1"
  subnet_id           = "subnet-1"
  volume_count        = 3
  total_volume_bytes  = 1073741824
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "volume_count", "3"),
					resource.TestCheckResourceAttr(dataSourceName, "total_volume_bytes", "1073741824"),
					resource.TestCheckResourceAttr(dataSourceName, "instance_types.#", "2"),
				),
			},
		},
	})
}

func TestAccRestoreAccountRegionAwsSecurityGroups(t *testing.T) {
	testAccPreCheck(t)

	server := newFakeEonServer(t)
	server.SetAwsSecurityGroups("acct-1", "us-east-1", externalEonSdkAPI.NewRestoreSecurityGroups(
		"us-east-1",
		[]externalEonSdkAPI.RestoreSecurityGroupsSecurityGroupsInner{
			*externalEonSdkAPI.NewRestoreSecurityGroupsSecurityGroupsInner("default", "sg-1", "vpc-1"),
			*externalEonSdkAPI.NewRestoreSecurityGroupsSecurityGroupsInner("web", "sg-2", "vpc-1"),
		},
	))

	dataSourceName := "data.eon_restore_account_region_aws_security_groups.demo"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: server.providerConfig() + `
data "eon_restore_account_region_aws_security_groups" "demo" {
  account_id = "acct-1"
  region     = "us-east-1"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "account_id", "acct-1"),
					resource.TestCheckResourceAttr(dataSourceName, "region", "us-east-1"),
					resource.TestCheckResourceAttr(dataSourceName, "security_groups.#", "2"),
					resource.TestCheckResourceAttr(dataSourceName, "security_groups.0.id", "sg-1"),
					resource.TestCheckResourceAttr(dataSourceName, "security_groups.0.name", "default"),
					resource.TestCheckResourceAttr(dataSourceName, "security_groups.0.vpc", "vpc-1"),
				),
			},
			{
				PreConfig: func() {
					server.SetAwsSecurityGroups("acct-1", "us-east-1", externalEonSdkAPI.NewRestoreSecurityGroups(
						"us-east-1",
						[]externalEonSdkAPI.RestoreSecurityGroupsSecurityGroupsInner{
							*externalEonSdkAPI.NewRestoreSecurityGroupsSecurityGroupsInner("default", "sg-1", "vpc-1"),
						},
					))
				},
				Config: server.providerConfig() + `
data "eon_restore_account_region_aws_security_groups" "demo" {
  account_id = "acct-1"
  region     = "us-east-1"
  vpc_id     = "vpc-1"
  subnet_id  = "subnet-1"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "vpc_id", "vpc-1"),
					resource.TestCheckResourceAttr(dataSourceName, "subnet_id", "subnet-1"),
					resource.TestCheckResourceAttr(dataSourceName, "security_groups.#", "1"),
				),
			},
		},
	})
}

func TestAccRestoreAccountRegionAwsSubnets(t *testing.T) {
	testAccPreCheck(t)

	server := newFakeEonServer(t)
	named := externalEonSdkAPI.NewListSubnetsResponseV2SubnetsInner("subnet-1", "vpc-1", "us-east-1a")
	named.SetName("private-a")
	named.SetCidrBlock("10.0.0.0/24")
	unnamed := externalEonSdkAPI.NewListSubnetsResponseV2SubnetsInner("subnet-2", "vpc-1", "us-east-1b")
	server.SetAwsSubnets("acct-1", "us-east-1", externalEonSdkAPI.NewListSubnetsResponseV2(
		[]externalEonSdkAPI.ListSubnetsResponseV2SubnetsInner{*named, *unnamed},
	))
	euSubnet := externalEonSdkAPI.NewListSubnetsResponseV2SubnetsInner("subnet-eu", "vpc-eu", "eu-west-1a")
	euSubnet.SetCidrBlock("10.1.0.0/24")
	server.SetAwsSubnets("acct-1", "eu-west-1", externalEonSdkAPI.NewListSubnetsResponseV2(
		[]externalEonSdkAPI.ListSubnetsResponseV2SubnetsInner{*euSubnet},
	))

	dataSourceName := "data.eon_restore_account_region_aws_subnets.demo"
	config := func(region string) string {
		return server.providerConfig() + `
data "eon_restore_account_region_aws_subnets" "demo" {
  account_id = "acct-1"
  region     = "` + region + `"
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
				),
			},
			{
				Config: config("eu-west-1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "region", "eu-west-1"),
					resource.TestCheckResourceAttr(dataSourceName, "subnets.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "subnets.0.id", "subnet-eu"),
					resource.TestCheckResourceAttr(dataSourceName, "subnets.0.availability_zone", "eu-west-1a"),
					resource.TestCheckResourceAttr(dataSourceName, "subnets.0.cidr_block", "10.1.0.0/24"),
					resource.TestCheckNoResourceAttr(dataSourceName, "subnets.0.name"),
				),
			},
			{
				PreConfig: func() {
					server.SetAwsSubnets("acct-1", "eu-west-1", nil)
				},
				Config: config("eu-west-1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "subnets.#", "0"),
				),
			},
		},
	})
}

func TestAccRestoreAccountRegionAwsVpcs(t *testing.T) {
	testAccPreCheck(t)

	server := newFakeEonServer(t)
	named := externalEonSdkAPI.NewListAwsVpcsResponseVpcsInner("vpc-1")
	named.SetName("main")
	server.SetAwsVpcs("acct-1", "us-east-1", externalEonSdkAPI.NewListAwsVpcsResponse(
		[]externalEonSdkAPI.ListAwsVpcsResponseVpcsInner{*named, *externalEonSdkAPI.NewListAwsVpcsResponseVpcsInner("vpc-2")},
	))
	server.SetAwsVpcs("acct-1", "eu-west-1", externalEonSdkAPI.NewListAwsVpcsResponse(
		[]externalEonSdkAPI.ListAwsVpcsResponseVpcsInner{*externalEonSdkAPI.NewListAwsVpcsResponseVpcsInner("vpc-eu")},
	))

	dataSourceName := "data.eon_restore_account_region_aws_vpcs.demo"
	config := func(region string) string {
		return server.providerConfig() + `
data "eon_restore_account_region_aws_vpcs" "demo" {
  account_id = "acct-1"
  region     = "` + region + `"
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
					resource.TestCheckResourceAttr(dataSourceName, "vpcs.#", "2"),
					resource.TestCheckResourceAttr(dataSourceName, "vpcs.0.id", "vpc-1"),
					resource.TestCheckResourceAttr(dataSourceName, "vpcs.0.name", "main"),
					resource.TestCheckResourceAttr(dataSourceName, "vpcs.1.id", "vpc-2"),
					resource.TestCheckNoResourceAttr(dataSourceName, "vpcs.1.name"),
				),
			},
			{
				Config: config("eu-west-1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "region", "eu-west-1"),
					resource.TestCheckResourceAttr(dataSourceName, "vpcs.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "vpcs.0.id", "vpc-eu"),
				),
			},
			{
				PreConfig: func() {
					server.SetAwsVpcs("acct-1", "eu-west-1", nil)
				},
				Config: config("eu-west-1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "vpcs.#", "0"),
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

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: server.providerConfig() + `
data "eon_restore_account_region_encryption_keys" "demo" {
  account_id = "acct-1"
  region     = "us-east-1"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "account_id", "acct-1"),
					resource.TestCheckResourceAttr(dataSourceName, "region", "us-east-1"),
					resource.TestCheckNoResourceAttr(dataSourceName, "include_aws_managed_keys"),
					resource.TestCheckResourceAttr(dataSourceName, "keys.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "keys.0.id", "key-1"),
					resource.TestCheckResourceAttr(dataSourceName, "keys.0.arn", "arn:aws:kms:us-east-1:123456789012:key/key-1"),
					resource.TestCheckResourceAttr(dataSourceName, "keys.0.alias_name", "alias/restore"),
				),
			},
			{
				Config: server.providerConfig() + `
data "eon_restore_account_region_encryption_keys" "demo" {
  account_id               = "acct-1"
  region                   = "us-east-1"
  include_aws_managed_keys = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "include_aws_managed_keys", "true"),
					resource.TestCheckResourceAttr(dataSourceName, "keys.#", "2"),
					resource.TestCheckResourceAttr(dataSourceName, "keys.1.id", "key-aws"),
					resource.TestCheckResourceAttr(dataSourceName, "keys.1.alias_name", "alias/aws/ebs"),
				),
			},
			{
				PreConfig: func() {
					server.SetEncryptionKeys("acct-1", "us-east-1", nil)
				},
				Config: server.providerConfig() + `
data "eon_restore_account_region_encryption_keys" "demo" {
  account_id               = "acct-1"
  region                   = "us-east-1"
  include_aws_managed_keys = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "keys.#", "0"),
				),
			},
		},
	})
}
