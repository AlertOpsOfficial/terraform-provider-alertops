package provider

import (
	"context"
	"os"
	"strconv"

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
	APIKey            types.String `tfsdk:"api_key"`
	BaseURL           types.String `tfsdk:"base_url"`
	RequestsPerSecond types.Int64  `tfsdk:"requests_per_second"`
	RequestsPerMinute types.Int64  `tfsdk:"requests_per_minute"`
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
			"base_url": schema.StringAttribute{
				MarkdownDescription: "Base URL of the AlertOps v2 API. Defaults to `https://api.alertops.com/api/v2`. Can also be set via the `ALERTOPS_BASE_URL` environment variable.",
				Optional:            true,
			},
			"requests_per_second": schema.Int64Attribute{
				MarkdownDescription: "Maximum API requests per second made by the provider. Defaults to `5` (the AlertOps API limit). Can also be set via `ALERTOPS_REQUESTS_PER_SECOND`.",
				Optional:            true,
			},
			"requests_per_minute": schema.Int64Attribute{
				MarkdownDescription: "Maximum API requests per minute made by the provider. Defaults to `15` (the AlertOps API limit). Raise this only if your account has a higher limit. Can also be set via `ALERTOPS_REQUESTS_PER_MINUTE`.",
				Optional:            true,
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

	baseURL := data.BaseURL.ValueString()
	if baseURL == "" {
		baseURL = os.Getenv("ALERTOPS_BASE_URL")
	}
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	perSecond := intSetting(data.RequestsPerSecond, "ALERTOPS_REQUESTS_PER_SECOND", defaultRequestsPerSecond)
	perMinute := intSetting(data.RequestsPerMinute, "ALERTOPS_REQUESTS_PER_MINUTE", defaultRequestsPerMinute)

	client := NewClientWithOptions(apiKey, baseURL, perSecond, perMinute)

	resp.DataSourceData = client
	resp.ResourceData = client
}

// intSetting resolves an integer provider setting from config, then the environment, then a default.
func intSetting(v types.Int64, envVar string, def int) int {
	if !v.IsNull() && !v.IsUnknown() {
		return int(v.ValueInt64())
	}
	if s := os.Getenv(envVar); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			return n
		}
	}
	return def
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
		NewTopicResource,
		NewResponsePlayResource,
		NewWorkflowResource,
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
