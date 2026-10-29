package provider

import (
	"testing"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRestoreAccountAwsS3Buckets(t *testing.T) {
	testAccPreCheck(t)

	server := newFakeEonServer(t)
	server.SetS3Buckets("racc-1", []externalEonSdkAPI.S3Bucket{
		*externalEonSdkAPI.NewS3Bucket("bucket-a", "us-east-1"),
		*externalEonSdkAPI.NewS3Bucket("bucket-b", "us-west-2"),
	})

	dataSourceName := "data.eon_restore_account_aws_s3_buckets.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: server.providerConfig() + `
data "eon_restore_account_aws_s3_buckets" "test" {
  restore_account_id = "racc-1"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "restore_account_id", "racc-1"),
					resource.TestCheckResourceAttr(dataSourceName, "buckets.#", "2"),
					resource.TestCheckResourceAttr(dataSourceName, "buckets.0.bucket_name", "bucket-a"),
					resource.TestCheckResourceAttr(dataSourceName, "buckets.0.region_name", "us-east-1"),
					resource.TestCheckResourceAttr(dataSourceName, "buckets.1.bucket_name", "bucket-b"),
					resource.TestCheckResourceAttr(dataSourceName, "buckets.1.region_name", "us-west-2"),
				),
			},
		},
	})
}
