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
