package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &AlertOpsProvider{}

type AlertOpsProvider struct {
	version string
}

type AlertOpsProviderModel struct {
	APIKey types.String `tfsdk:"api_key"`
}

func (p *AlertOpsProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "alertops"
	resp.Version = p.version
}

func (p *AlertOpsProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The AlertOps provider allows you to manage AlertOps resources using Terraform.",
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				MarkdownDescription: "AlertOps API Key. Can also be set via the `ALERTOPS_API_KEY` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
		},
	}
}

func (p *AlertOpsProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data AlertOpsProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	apiKey := data.APIKey.ValueString()
	if apiKey == "" {
		apiKey = os.Getenv("ALERTOPS_API_KEY")
	}

	if apiKey == "" {
		resp.Diagnostics.AddError(
			"Missing API Key",
			"The provider cannot be configured without an API key. "+
				"Set the api_key provider argument or the ALERTOPS_API_KEY environment variable.",
		)
		return
	}

	client := NewClient(apiKey)

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *AlertOpsProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewUserResource,
		NewGroupResource,
		NewScheduleResource,
		NewServiceResource,
		NewInboundIntegrationResource,
		NewOutboundIntegrationResource,
		NewEscalationPolicyResource,
		NewMaintenanceWindowResource,
	}
}

func (p *AlertOpsProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &AlertOpsProvider{
			version: version,
		}
	}
}
