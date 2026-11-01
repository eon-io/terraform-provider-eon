package provider

import (
	"fmt"
	"regexp"
	"testing"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const testAccRestoreTargetConsumer = "terraform_data.consumer"

// testAccCaptureID stores the ID of a resource so later steps can assert it was kept or replaced.
func testAccCaptureID(name string, id *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("resource not found: %s", name)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("empty id for %s", name)
		}
		*id = rs.Primary.ID
		return nil
	}
}

func testAccCheckIDUnchanged(name string, id *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("resource not found: %s", name)
		}
		if rs.Primary.ID != *id {
			return fmt.Errorf("expected in-place update to keep id %q, got %q", *id, rs.Primary.ID)
		}
		return nil
	}
}

func testAccRestoreAccountAwsS3BucketsConfig(server *fakeEonServer, accountID string) string {
	return server.providerConfig() + fmt.Sprintf(`
data "eon_restore_account_aws_s3_buckets" "test" {
  restore_account_id = %q
}

resource "terraform_data" "consumer" {
  input            = data.eon_restore_account_aws_s3_buckets.test.buckets
  triggers_replace = one([for b in data.eon_restore_account_aws_s3_buckets.test.buckets : b.bucket_name if b.bucket_name == "restore-primary"])
}
`, accountID)
}

func TestAccRestoreAccountAwsS3Buckets(t *testing.T) {
	testAccPreCheck(t)

	server := newFakeEonServer(t)
	server.AddS3Bucket("ra-1", externalEonSdkAPI.NewS3Bucket("restore-primary", "us-east-1"))
	server.AddS3Bucket("ra-1", externalEonSdkAPI.NewS3Bucket("restore-logs", "us-east-1"))
	server.AddS3Bucket("ra-2", externalEonSdkAPI.NewS3Bucket("restore-primary", "eu-west-1"))
	server.AddS3Bucket("ra-2", externalEonSdkAPI.NewS3Bucket("restore-archive", "eu-west-1"))
	server.AddS3Bucket("ra-2", externalEonSdkAPI.NewS3Bucket("restore-cold", "eu-west-1"))

	dataSourceName := "data.eon_restore_account_aws_s3_buckets.test"
	var consumerID string

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: server.providerConfig() + `
data "eon_restore_account_aws_s3_buckets" "missing" {
  restore_account_id = "ra-missing"
}
`,
				ExpectError: regexp.MustCompile(`Unable to read restore account S3 buckets`),
			},
			{
				Config: testAccRestoreAccountAwsS3BucketsConfig(server, "ra-1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "restore_account_id", "ra-1"),
					resource.TestCheckResourceAttr(dataSourceName, "buckets.#", "2"),
					resource.TestCheckResourceAttr(dataSourceName, "buckets.0.bucket_name", "restore-primary"),
					resource.TestCheckResourceAttr(dataSourceName, "buckets.0.region", "us-east-1"),
					resource.TestCheckResourceAttr(dataSourceName, "buckets.1.bucket_name", "restore-logs"),
					resource.TestCheckResourceAttr(dataSourceName, "buckets.1.region", "us-east-1"),
					resource.TestCheckResourceAttr(testAccRestoreTargetConsumer, "input.#", "2"),
					resource.TestCheckResourceAttr(testAccRestoreTargetConsumer, "triggers_replace", "restore-primary"),
					testAccCaptureID(testAccRestoreTargetConsumer, &consumerID),
				),
			},
			{
				Config: testAccRestoreAccountAwsS3BucketsConfig(server, "ra-2"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(testAccRestoreTargetConsumer, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "restore_account_id", "ra-2"),
					resource.TestCheckResourceAttr(dataSourceName, "buckets.#", "3"),
					resource.TestCheckResourceAttr(dataSourceName, "buckets.0.bucket_name", "restore-primary"),
					resource.TestCheckResourceAttr(dataSourceName, "buckets.0.region", "eu-west-1"),
					resource.TestCheckResourceAttr(dataSourceName, "buckets.2.bucket_name", "restore-cold"),
					resource.TestCheckResourceAttr(testAccRestoreTargetConsumer, "input.#", "3"),
					testAccCheckIDUnchanged(testAccRestoreTargetConsumer, &consumerID),
				),
			},
			{
				Config: testAccRestoreAccountAwsS3BucketsConfig(server, "ra-2") + `
data "eon_restore_account_aws_s3_buckets" "other" {
  restore_account_id = "ra-1"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.eon_restore_account_aws_s3_buckets.other", "buckets.#", "2"),
					resource.TestCheckResourceAttr("data.eon_restore_account_aws_s3_buckets.other", "buckets.1.bucket_name", "restore-logs"),
					testAccCheckIDUnchanged(testAccRestoreTargetConsumer, &consumerID),
				),
			},
			{
				PreConfig: func() {
					server.DeleteS3Bucket("ra-2", "restore-primary")
				},
				Config:             testAccRestoreAccountAwsS3BucketsConfig(server, "ra-2"),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPreRefresh: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(testAccRestoreTargetConsumer, plancheck.ResourceActionReplace),
					},
				},
			},
		},
	})
}

func testAccRestoreAccountRegionAwsInstanceTypesConfig(server *fakeEonServer, sizing string) string {
	return server.providerConfig() + fmt.Sprintf(`
data "eon_restore_account_region_aws_instance_types" "test" {
  restore_account_id = "ra-1"
  region             = "us-east-1"
  subnet_id          = "subnet-1"
%s
}

resource "terraform_data" "consumer" {
  input            = data.eon_restore_account_region_aws_instance_types.test.instance_types
  triggers_replace = one([for t in data.eon_restore_account_region_aws_instance_types.test.instance_types : t if t == "m5.large"])
}
`, sizing)
}

func newTestInstanceTypes(instanceTypes []string, mapping map[string]string) *externalEonSdkAPI.ListInstanceTypesResponse {
	resp := externalEonSdkAPI.NewListInstanceTypesResponse()
	resp.SetInstanceTypes(instanceTypes)
	if mapping != nil {
		resp.SetInstantRestoreMapping(mapping)
	}
	return resp
}

func TestAccRestoreAccountRegionAwsInstanceTypes(t *testing.T) {
	testAccPreCheck(t)

	server := newFakeEonServer(t)
	server.SetInstanceTypes("ra-1", "us-east-1", "subnet-1", newTestInstanceTypes(
		[]string{"t3.micro", "m5.large"},
		map[string]string{"m5.large": "m5.xlarge"},
	))
	server.SetInstanceTypes("ra-1", "us-east-1", "subnet-2", newTestInstanceTypes([]string{"r5.large"}, nil))

	dataSourceName := "data.eon_restore_account_region_aws_instance_types.test"
	var consumerID string

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: server.providerConfig() + `
data "eon_restore_account_region_aws_instance_types" "missing" {
  restore_account_id = "ra-1"
  region             = "us-east-1"
  subnet_id          = "subnet-missing"
}
`,
				ExpectError: regexp.MustCompile(`Unable to read restore account AWS instance types`),
			},
			{
				Config: testAccRestoreAccountRegionAwsInstanceTypesConfig(server, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "restore_account_id", "ra-1"),
					resource.TestCheckResourceAttr(dataSourceName, "region", "us-east-1"),
					resource.TestCheckResourceAttr(dataSourceName, "subnet_id", "subnet-1"),
					resource.TestCheckNoResourceAttr(dataSourceName, "volume_count"),
					resource.TestCheckNoResourceAttr(dataSourceName, "total_volume_bytes"),
					resource.TestCheckResourceAttr(dataSourceName, "instance_types.#", "2"),
					resource.TestCheckResourceAttr(dataSourceName, "instance_types.0", "t3.micro"),
					resource.TestCheckResourceAttr(dataSourceName, "instance_types.1", "m5.large"),
					resource.TestCheckResourceAttr(dataSourceName, "instant_restore_mapping.%", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "instant_restore_mapping.m5.large", "m5.xlarge"),
					resource.TestCheckResourceAttr(testAccRestoreTargetConsumer, "triggers_replace", "m5.large"),
					testAccCaptureID(testAccRestoreTargetConsumer, &consumerID),
					func(s *terraform.State) error {
						query := server.LastInstanceTypesQuery()
						if len(query) != 1 || query["subnetId"] != "subnet-1" {
							return fmt.Errorf("expected only subnetId=subnet-1 in query, got %v", query)
						}
						return nil
					},
				),
			},
			{
				Config: testAccRestoreAccountRegionAwsInstanceTypesConfig(server, `
  volume_count       = 3
  total_volume_bytes = 107374182400
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "volume_count", "3"),
					resource.TestCheckResourceAttr(dataSourceName, "total_volume_bytes", "107374182400"),
					resource.TestCheckResourceAttr(dataSourceName, "instance_types.#", "2"),
					testAccCheckIDUnchanged(testAccRestoreTargetConsumer, &consumerID),
					func(s *terraform.State) error {
						query := server.LastInstanceTypesQuery()
						if query["subnetId"] != "subnet-1" || query["volumeCount"] != "3" || query["totalVolumeBytes"] != "107374182400" {
							return fmt.Errorf("expected sizing hints in query, got %v", query)
						}
						return nil
					},
				),
			},
			{
				Config: testAccRestoreAccountRegionAwsInstanceTypesConfig(server, "") + `
data "eon_restore_account_region_aws_instance_types" "other" {
  restore_account_id = "ra-1"
  region             = "us-east-1"
  subnet_id          = "subnet-2"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.eon_restore_account_region_aws_instance_types.other", "instance_types.#", "1"),
					resource.TestCheckResourceAttr("data.eon_restore_account_region_aws_instance_types.other", "instance_types.0", "r5.large"),
					resource.TestCheckResourceAttr("data.eon_restore_account_region_aws_instance_types.other", "instant_restore_mapping.%", "0"),
					testAccCheckIDUnchanged(testAccRestoreTargetConsumer, &consumerID),
				),
			},
			{
				PreConfig: func() {
					server.SetInstanceTypes("ra-1", "us-east-1", "subnet-1", newTestInstanceTypes([]string{"t3.micro"}, nil))
				},
				Config:             testAccRestoreAccountRegionAwsInstanceTypesConfig(server, ""),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPreRefresh: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(testAccRestoreTargetConsumer, plancheck.ResourceActionReplace),
					},
				},
			},
		},
	})
}

func testAccRestoreAccountRegionAwsSecurityGroupsConfig(server *fakeEonServer, filter string) string {
	return server.providerConfig() + fmt.Sprintf(`
data "eon_restore_account_region_aws_security_groups" "test" {
  restore_account_id = "ra-1"
  region             = "us-east-1"
%s
}

resource "terraform_data" "consumer" {
  input            = data.eon_restore_account_region_aws_security_groups.test.security_groups
  triggers_replace = one([for sg in data.eon_restore_account_region_aws_security_groups.test.security_groups : sg.id if sg.name == "web"])
}
`, filter)
}

func TestAccRestoreAccountRegionAwsSecurityGroups(t *testing.T) {
	testAccPreCheck(t)

	server := newFakeEonServer(t)
	server.AddSubnet("subnet-2", "vpc-2")
	server.AddSecurityGroup("ra-1", "us-east-1", externalEonSdkAPI.NewRestoreSecurityGroupsSecurityGroupsInner("default", "sg-1", "vpc-1"))
	server.AddSecurityGroup("ra-1", "us-east-1", externalEonSdkAPI.NewRestoreSecurityGroupsSecurityGroupsInner("web", "sg-2", "vpc-1"))
	server.AddSecurityGroup("ra-1", "us-east-1", externalEonSdkAPI.NewRestoreSecurityGroupsSecurityGroupsInner("db", "sg-3", "vpc-2"))

	dataSourceName := "data.eon_restore_account_region_aws_security_groups.test"
	var consumerID string

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: server.providerConfig() + `
data "eon_restore_account_region_aws_security_groups" "missing" {
  restore_account_id = "ra-1"
  region             = "eu-central-1"
}
`,
				ExpectError: regexp.MustCompile(`Unable to read restore account AWS security groups`),
			},
			{
				Config: testAccRestoreAccountRegionAwsSecurityGroupsConfig(server, ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "restore_account_id", "ra-1"),
					resource.TestCheckResourceAttr(dataSourceName, "region", "us-east-1"),
					resource.TestCheckNoResourceAttr(dataSourceName, "vpc_id"),
					resource.TestCheckNoResourceAttr(dataSourceName, "subnet_id"),
					resource.TestCheckResourceAttr(dataSourceName, "security_groups.#", "3"),
					resource.TestCheckResourceAttr(dataSourceName, "security_groups.0.id", "sg-1"),
					resource.TestCheckResourceAttr(dataSourceName, "security_groups.0.name", "default"),
					resource.TestCheckResourceAttr(dataSourceName, "security_groups.0.vpc_id", "vpc-1"),
					resource.TestCheckResourceAttr(dataSourceName, "security_groups.2.id", "sg-3"),
					resource.TestCheckResourceAttr(dataSourceName, "security_groups.2.vpc_id", "vpc-2"),
					resource.TestCheckResourceAttr(testAccRestoreTargetConsumer, "input.#", "3"),
					resource.TestCheckResourceAttr(testAccRestoreTargetConsumer, "triggers_replace", "sg-2"),
					testAccCaptureID(testAccRestoreTargetConsumer, &consumerID),
				),
			},
			{
				Config: testAccRestoreAccountRegionAwsSecurityGroupsConfig(server, `  vpc_id = "vpc-1"`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(testAccRestoreTargetConsumer, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "vpc_id", "vpc-1"),
					resource.TestCheckResourceAttr(dataSourceName, "security_groups.#", "2"),
					resource.TestCheckResourceAttr(dataSourceName, "security_groups.1.id", "sg-2"),
					resource.TestCheckResourceAttr(testAccRestoreTargetConsumer, "input.#", "2"),
					testAccCheckIDUnchanged(testAccRestoreTargetConsumer, &consumerID),
				),
			},
			{
				Config: testAccRestoreAccountRegionAwsSecurityGroupsConfig(server, `  vpc_id = "vpc-1"`) + `
data "eon_restore_account_region_aws_security_groups" "by_subnet" {
  restore_account_id = "ra-1"
  region             = "us-east-1"
  subnet_id          = "subnet-2"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.eon_restore_account_region_aws_security_groups.by_subnet", "subnet_id", "subnet-2"),
					resource.TestCheckResourceAttr("data.eon_restore_account_region_aws_security_groups.by_subnet", "security_groups.#", "1"),
					resource.TestCheckResourceAttr("data.eon_restore_account_region_aws_security_groups.by_subnet", "security_groups.0.id", "sg-3"),
					resource.TestCheckResourceAttr("data.eon_restore_account_region_aws_security_groups.by_subnet", "security_groups.0.name", "db"),
					testAccCheckIDUnchanged(testAccRestoreTargetConsumer, &consumerID),
				),
			},
			{
				PreConfig: func() {
					server.DeleteSecurityGroup("ra-1", "us-east-1", "sg-2")
				},
				Config:             testAccRestoreAccountRegionAwsSecurityGroupsConfig(server, `  vpc_id = "vpc-1"`),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPreRefresh: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(testAccRestoreTargetConsumer, plancheck.ResourceActionReplace),
					},
				},
			},
		},
	})
}
