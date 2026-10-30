package provider

import (
	"testing"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRestoreAccountAwsS3Buckets(t *testing.T) {
	testAccPreCheck(t)

	server := newFakeEonServer(t)
	accountID := "restore-acct-1"
	server.AddS3Bucket(accountID, *externalEonSdkAPI.NewS3Bucket("bucket-a", "us-east-1"))
	server.AddS3Bucket(accountID, *externalEonSdkAPI.NewS3Bucket("bucket-b", "us-west-2"))

	dataSourceName := "data.eon_restore_account_aws_s3_buckets.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: server.providerConfig() + `
data "eon_restore_account_aws_s3_buckets" "test" {
  restore_account_id = "restore-acct-1"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "restore_account_id", accountID),
					resource.TestCheckResourceAttr(dataSourceName, "buckets.#", "2"),
					resource.TestCheckResourceAttr(dataSourceName, "buckets.0.bucket_name", "bucket-a"),
					resource.TestCheckResourceAttr(dataSourceName, "buckets.0.region_name", "us-east-1"),
					resource.TestCheckResourceAttr(dataSourceName, "buckets.1.bucket_name", "bucket-b"),
					resource.TestCheckResourceAttr(dataSourceName, "buckets.1.region_name", "us-west-2"),
				),
			},
			{
				PreConfig: func() {
					server.AddS3Bucket(accountID, *externalEonSdkAPI.NewS3Bucket("bucket-c", "eu-west-1"))
				},
				Config: server.providerConfig() + `
data "eon_restore_account_aws_s3_buckets" "test" {
  restore_account_id = "restore-acct-1"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "buckets.#", "3"),
				),
			},
		},
	})
}

func TestAccRestoreAccountRegionAwsInstanceTypes(t *testing.T) {
	testAccPreCheck(t)

	server := newFakeEonServer(t)
	accountID := "restore-acct-1"
	server.AddInstanceTypes(accountID, "us-east-1", "subnet-1",
		[]string{"m5.large", "m5.xlarge"},
		map[string]string{"m5.large": "m5.large-instant"},
	)
	server.AddInstanceTypes(accountID, "us-east-1", "subnet-2",
		[]string{"m5.2xlarge"},
		nil,
	)

	dataSourceName := "data.eon_restore_account_region_aws_instance_types.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: server.providerConfig() + `
data "eon_restore_account_region_aws_instance_types" "test" {
  restore_account_id = "restore-acct-1"
  region             = "us-east-1"
  subnet_id          = "subnet-1"
  volume_count       = 2
  total_volume_bytes = 1073741824
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "restore_account_id", accountID),
					resource.TestCheckResourceAttr(dataSourceName, "region", "us-east-1"),
					resource.TestCheckResourceAttr(dataSourceName, "subnet_id", "subnet-1"),
					resource.TestCheckResourceAttr(dataSourceName, "instance_types.#", "2"),
					resource.TestCheckResourceAttr(dataSourceName, "instance_types.0", "m5.large"),
					resource.TestCheckResourceAttr(dataSourceName, "instance_types.1", "m5.xlarge"),
					resource.TestCheckResourceAttr(dataSourceName, "instant_restore_mapping.m5.large", "m5.large-instant"),
				),
			},
			{
				Config: server.providerConfig() + `
data "eon_restore_account_region_aws_instance_types" "test" {
  restore_account_id = "restore-acct-1"
  region             = "us-east-1"
  subnet_id          = "subnet-2"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "subnet_id", "subnet-2"),
					resource.TestCheckResourceAttr(dataSourceName, "instance_types.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "instance_types.0", "m5.2xlarge"),
					resource.TestCheckResourceAttr(dataSourceName, "instant_restore_mapping.%", "0"),
				),
			},
		},
	})
}

func TestAccRestoreAccountRegionAwsSecurityGroups(t *testing.T) {
	testAccPreCheck(t)

	server := newFakeEonServer(t)
	accountID := "restore-acct-1"
	server.AddSecurityGroup(accountID, "us-east-1", "sg-1", "default", "vpc-1", "subnet-1")
	server.AddSecurityGroup(accountID, "us-east-1", "sg-2", "web", "vpc-2", "subnet-2")

	dataSourceName := "data.eon_restore_account_region_aws_security_groups.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: server.providerConfig() + `
data "eon_restore_account_region_aws_security_groups" "test" {
  restore_account_id = "restore-acct-1"
  region             = "us-east-1"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "restore_account_id", accountID),
					resource.TestCheckResourceAttr(dataSourceName, "region", "us-east-1"),
					resource.TestCheckResourceAttr(dataSourceName, "security_groups.#", "2"),
					resource.TestCheckResourceAttr(dataSourceName, "security_groups.0.id", "sg-1"),
					resource.TestCheckResourceAttr(dataSourceName, "security_groups.0.name", "default"),
					resource.TestCheckResourceAttr(dataSourceName, "security_groups.0.vpc", "vpc-1"),
				),
			},
			{
				Config: server.providerConfig() + `
data "eon_restore_account_region_aws_security_groups" "test" {
  restore_account_id = "restore-acct-1"
  region             = "us-east-1"
  vpc_id             = "vpc-2"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "security_groups.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "security_groups.0.id", "sg-2"),
					resource.TestCheckResourceAttr(dataSourceName, "security_groups.0.name", "web"),
					resource.TestCheckResourceAttr(dataSourceName, "security_groups.0.vpc", "vpc-2"),
				),
			},
			{
				Config: server.providerConfig() + `
data "eon_restore_account_region_aws_security_groups" "test" {
  restore_account_id = "restore-acct-1"
  region             = "us-east-1"
  subnet_id          = "subnet-1"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "security_groups.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "security_groups.0.id", "sg-1"),
				),
			},
		},
	})
}
