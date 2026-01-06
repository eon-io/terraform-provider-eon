package provider

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// CloudProvider represents a cloud provider type (AWS, Azure, GCP, etc.)
type CloudProvider string

const (
	CloudProviderAWS   CloudProvider = "AWS"
	CloudProviderAzure CloudProvider = "AZURE"
	CloudProviderGCP   CloudProvider = "GCP"
)

// BlockName returns the lowercase Terraform schema block name (e.g., "aws", "azure")
func (c CloudProvider) BlockName() string {
	return strings.ToLower(string(c))
}

// String implements Stringer for SDK compatibility
func (c CloudProvider) String() string {
	return string(c)
}

// AwsAccountConfigModel represents AWS-specific configuration for cloud accounts.
// Shared between source and restore accounts.
type AwsAccountConfigModel struct {
	RoleArn types.String `tfsdk:"role_arn"`
}

// AzureAccountConfigModel represents Azure-specific configuration for cloud accounts.
// Shared between source and restore accounts.
type AzureAccountConfigModel struct {
	TenantId          types.String `tfsdk:"tenant_id"`
	SubscriptionId    types.String `tfsdk:"subscription_id"`
	ResourceGroupName types.String `tfsdk:"resource_group_name"`
}

// GcpAccountConfigModel represents GCP-specific configuration for cloud accounts.
// Shared between source and restore accounts.
type GcpAccountConfigModel struct {
	ProjectId      types.String `tfsdk:"project_id"`
	ServiceAccount types.String `tfsdk:"service_account"`
}

// awsSchemaBlock returns the schema block for AWS-specific configuration.
func awsSchemaBlock() schema.SingleNestedBlock {
	return schema.SingleNestedBlock{
		MarkdownDescription: "AWS-specific configuration. Required when `cloud_provider` is `AWS`.",
		Attributes: map[string]schema.Attribute{
			"role_arn": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "ARN of the IAM role Eon assumes to access the account. Required when using the aws block.",
			},
		},
	}
}

// azureSchemaBlock returns the schema block for Azure-specific configuration.
// The resourceGroupDescription parameter allows customization for source vs restore context.
func azureSchemaBlock(resourceGroupDescription string) schema.SingleNestedBlock {
	return schema.SingleNestedBlock{
		MarkdownDescription: "Azure-specific configuration. Required when `cloud_provider` is `AZURE`.",
		Attributes: map[string]schema.Attribute{
			"tenant_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Azure Active Directory tenant ID. Required when using the azure block.",
			},
			"subscription_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Azure subscription ID. Required when using the azure block.",
			},
			"resource_group_name": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: resourceGroupDescription,
			},
		},
	}
}

// gcpSchemaBlock returns the schema block for GCP-specific configuration.
func gcpSchemaBlock() schema.SingleNestedBlock {
	return schema.SingleNestedBlock{
		MarkdownDescription: "GCP-specific configuration. Required when `cloud_provider` is `GCP`.",
		Attributes: map[string]schema.Attribute{
			"project_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "GCP project ID. Required when using the gcp block.",
			},
			"service_account": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Email of the GCP service account Eon uses to access the project. Required when using the gcp block.",
			},
		},
	}
}

// conflictErrorMessage generates a user-friendly error message for 409 Conflict errors.
// It provides the terraform import command to help users adopt existing resources.
func conflictErrorMessage(resourceType, existingID string) (title, detail string) {
	title = fmt.Sprintf("%s Already Exists", resourceType)

	if existingID != "" {
		detail = fmt.Sprintf("A %s with this configuration already exists.\n\n"+
			"To manage this existing resource with Terraform, import it using:\n\n"+
			"  terraform import eon_%s.YOUR_RESOURCE_NAME %s\n\n"+
			"Replace YOUR_RESOURCE_NAME with the name from your Terraform configuration.",
			resourceType, resourceTypeTFName(resourceType), existingID)
	} else {
		detail = fmt.Sprintf("A %s with this configuration already exists.\n\n"+
			"To manage this existing resource with Terraform, import it using:\n\n"+
			"  terraform import eon_%s.YOUR_RESOURCE_NAME ACCOUNT_ID\n\n"+
			"Use the eon_%ss data source to find the account ID.",
			resourceType, resourceTypeTFName(resourceType), resourceTypeTFName(resourceType))
	}
	return title, detail
}

// resourceTypeTFName converts a human-readable resource type to Terraform resource name format.
func resourceTypeTFName(resourceType string) string {
	switch resourceType {
	case "Source Account":
		return "source_account"
	case "Restore Account":
		return "restore_account"
	default:
		return "account"
	}
}
