package provider

import (
	"context"
	"os"

	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ provider.Provider = &EonProvider{}

// EonProvider defines the provider implementation.
type EonProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// EonProviderModel describes the provider data model.
type EonProviderModel struct {
	Endpoint     types.String `tfsdk:"endpoint"`
	ClientId     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
	ProjectId    types.String `tfsdk:"project_id"`
}

// New creates a new provider instance.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &EonProvider{
			version: version,
		}
	}
}

func (p *EonProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "eon"
	resp.Version = p.version
}

func (p *EonProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The Eon provider allows you to manage your Eon cloud backup and restore infrastructure using Terraform. Configure your cloud accounts, manage backup policies, and orchestrate disaster recovery workflows.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "Eon API base URL in the format `https://<your-domain>.console.eon.io` (no trailing slash). Can also be set with the `EON_ENDPOINT` environment variable.",
				Optional:            true,
			},
			"client_id": schema.StringAttribute{
				MarkdownDescription: "Eon API client ID for authentication. Can also be set with the `EON_CLIENT_ID` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"client_secret": schema.StringAttribute{
				MarkdownDescription: "Eon API client secret for authentication. Can also be set with the `EON_CLIENT_SECRET` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "Eon project ID. Can also be set with the `EON_PROJECT_ID` environment variable.",
				Optional:            true,
			},
		},
	}
}

func (p *EonProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data EonProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := os.Getenv("EON_ENDPOINT")
	clientId := os.Getenv("EON_CLIENT_ID")
	clientSecret := os.Getenv("EON_CLIENT_SECRET")
	projectId := os.Getenv("EON_PROJECT_ID")

	if !data.Endpoint.IsNull() {
		endpoint = data.Endpoint.ValueString()
	}

	if !data.ClientId.IsNull() {
		clientId = data.ClientId.ValueString()
	}

	if !data.ClientSecret.IsNull() {
		clientSecret = data.ClientSecret.ValueString()
	}

	if !data.ProjectId.IsNull() {
		projectId = data.ProjectId.ValueString()
	}

	// Validate required fields
	if endpoint == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("endpoint"),
			"Missing Eon API Endpoint",
			"The provider requires an endpoint URL. Set the endpoint value in the configuration or use the `EON_ENDPOINT` environment variable.",
		)
	}

	if clientId == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("client_id"),
			"Missing Eon Client ID",
			"The provider requires a client ID. Set the client_id value in the configuration or use the `EON_CLIENT_ID` environment variable.",
		)
	}

	if clientSecret == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("client_secret"),
			"Missing Eon Client Secret",
			"The provider requires a client secret. Set the client_secret value in the configuration or use the `EON_CLIENT_SECRET` environment variable.",
		)
	}

	if projectId == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("project_id"),
			"Missing Eon Project ID",
			"The provider requires a project ID. Set the project_id value in the configuration or use the `EON_PROJECT_ID` environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Create Eon client
	eonClient, err := client.NewEonClient(endpoint, clientId, clientSecret, projectId)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Eon API Client",
			"An unexpected error occurred when creating the Eon API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Eon Client Error: "+err.Error(),
		)
		return
	}

	resp.DataSourceData = eonClient
	resp.ResourceData = eonClient
}

func (p *EonProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewSourceAccountResource,
		NewRestoreAccountResource,
		NewRestoreJobResource,
		NewBackupPolicyResource,
	}
}

func (p *EonProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewSourceAccountsDataSource,
		NewRestoreAccountsDataSource,
		NewSnapshotDataSource,
		NewBackupPoliciesDataSource,
	}
}
