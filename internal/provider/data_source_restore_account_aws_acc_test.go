package provider

import (
	"testing"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRestoreAccountAwsLookups(t *testing.T) {
	testAccPreCheck(t)

	server := newFakeEonServer(t)
	const (
		accountID = "restore-acct-1"
		region    = "us-east-1"
		subnetID  = "subnet-1"
	)
	server.AddS3Bucket(accountID, externalEonSdkAPI.NewS3Bucket("restore-bucket", region))
	server.SetAwsInstanceTypes(accountID, region, subnetID, newTestInstanceTypes("m5.large", "i3.large"))
	server.SetAwsInstanceTypes(accountID, region, "subnet-2", newTestInstanceTypes("m5.xlarge", "i3.xlarge"))
	server.SetAwsSecurityGroups(accountID, region, externalEonSdkAPI.NewRestoreSecurityGroups(
		region,
		[]externalEonSdkAPI.RestoreSecurityGroupsSecurityGroupsInner{
			*externalEonSdkAPI.NewRestoreSecurityGroupsSecurityGroupsInner("restore", "sg-1", "vpc-1"),
		},
	))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: server.providerConfig() + `
data "eon_restore_account_aws_s3_buckets" "destination" {
  restore_account_id = "restore-acct-1"
}

data "eon_restore_account_region_aws_instance_types" "destination" {
  restore_account_id = "restore-acct-1"
  region             = "us-east-1"
  subnet_id          = "subnet-1"
  volume_count       = 2
  total_volume_bytes = 107374182400
}

data "eon_restore_account_region_aws_security_groups" "destination" {
  restore_account_id = "restore-acct-1"
  region             = "us-east-1"
  vpc_id             = "vpc-1"
  subnet_id          = "subnet-1"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.eon_restore_account_aws_s3_buckets.destination", "buckets.#", "1"),
					resource.TestCheckResourceAttr("data.eon_restore_account_aws_s3_buckets.destination", "buckets.0.bucket_name", "restore-bucket"),
					resource.TestCheckResourceAttr("data.eon_restore_account_region_aws_instance_types.destination", "instance_types.#", "1"),
					resource.TestCheckResourceAttr("data.eon_restore_account_region_aws_instance_types.destination", "instance_types.0", "m5.large"),
					resource.TestCheckResourceAttr("data.eon_restore_account_region_aws_instance_types.destination", "instant_restore_mapping.m5.large", "i3.large"),
					resource.TestCheckResourceAttr("data.eon_restore_account_region_aws_security_groups.destination", "security_groups.#", "1"),
					resource.TestCheckResourceAttr("data.eon_restore_account_region_aws_security_groups.destination", "security_groups.0.id", "sg-1"),
				),
			},
			{
				Config: server.providerConfig() + `
data "eon_restore_account_aws_s3_buckets" "destination" {
  restore_account_id = "restore-acct-1"
}

data "eon_restore_account_region_aws_instance_types" "destination" {
  restore_account_id = "restore-acct-1"
  region             = "us-east-1"
  subnet_id          = "subnet-2"
}

data "eon_restore_account_region_aws_security_groups" "destination" {
  restore_account_id = "restore-acct-1"
  region             = "us-east-1"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.eon_restore_account_region_aws_instance_types.destination", "instance_types.0", "m5.xlarge"),
					resource.TestCheckResourceAttr("data.eon_restore_account_region_aws_security_groups.destination", "security_groups.0.name", "restore"),
				),
			},
		},
	})
}

func newTestInstanceTypes(instanceType string, instantRestoreType string) *externalEonSdkAPI.ListInstanceTypesResponse {
	response := externalEonSdkAPI.NewListInstanceTypesResponse()
	response.SetInstanceTypes([]string{instanceType})
	response.SetInstantRestoreMapping(map[string]string{instanceType: instantRestoreType})
	return response
}
