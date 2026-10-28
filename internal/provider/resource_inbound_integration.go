package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                   = &InboundIntegrationResource{}
	_ resource.ResourceWithConfigure      = &InboundIntegrationResource{}
	_ resource.ResourceWithImportState    = &InboundIntegrationResource{}
	_ resource.ResourceWithValidateConfig = &InboundIntegrationResource{}
)

func NewInboundIntegrationResource() resource.Resource {
	return &InboundIntegrationResource{}
}

type InboundIntegrationResource struct {
	client *Client
}

type inboundIntegrationModel struct {
	ID                     types.String `tfsdk:"id"`
	InboundIntegrationName types.String `tfsdk:"inbound_integration_name"`
	Type                   types.String `tfsdk:"type"`
	Sequence               types.Int64  `tfsdk:"sequence"`
	Enabled                types.Bool   `tfsdk:"enabled"`
	EscalationPolicy       types.String `tfsdk:"escalation_policy"`
	ResponsePlay           types.String `tfsdk:"response_play"`
	ResponsePlayName       types.String `tfsdk:"response_play_name"`
	RecipientGroups        types.List   `tfsdk:"recipient_groups"`
	RecipientUsers         types.List   `tfsdk:"recipient_users"`
	InboundTemplateID      types.Int64  `tfsdk:"inbound_template_id"`
	APIURL                 types.String `tfsdk:"api_url"`
	ExtraAttributes        types.Map    `tfsdk:"extra_attributes"`
}

// defaultAPISettings mirrors what the AlertOps UI sends for a "Custom" API integration.
// The API rejects API-type integrations without a URL mapping source.
func defaultAPISettings() map[string]interface{} {
	return map[string]interface{}{
		"url_mapping": map[string]interface{}{
			"method":       "POST",
			"content":      "JSON",
			"source":       "Custom",
			"source_name":  "Custom",
			"static":       true,
			"source_value": "Custom",
			"open_alert_when": map[string]interface{}{
				"field_name": "Body",
				"type":       "Contains Any ",
				"values":     []string{""},
			},
		},
	}
}

func (r *InboundIntegrationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_inbound_integration"
}

func (r *InboundIntegrationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an AlertOps inbound integration. Any additional fields supported by the AlertOps API can be passed via `extra_attributes`. Complex nested fields (e.g., `api_settings`, `email_settings`, `chat_settings`) can be passed as JSON strings in `extra_attributes`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique identifier of the inbound integration.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"inbound_integration_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the inbound integration.",
			},
			"type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The type of inbound integration (e.g., `API`, `Email`, `Heartbeat`). For `API` integrations created without `api_settings` in `extra_attributes`, the provider sends the same default URL mapping as the AlertOps \"Custom\" template.",
			},
			"sequence": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(1),
				MarkdownDescription: "The sequence order of the integration.",
			},
			"enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Whether the integration is enabled.",
			},
			"escalation_policy": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The name of the escalation policy to use for this integration. Conflicts with `response_play`.",
			},
			"response_play": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The ID of the response play to use for this integration (e.g. `alertops_response_play.example.id`). Conflicts with `escalation_policy`.",
			},
			"response_play_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The name of the response play used by this integration.",
			},
			"recipient_groups": schema.ListAttribute{
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "List of group names to receive alerts from this integration.",
			},
			"recipient_users": schema.ListAttribute{
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "List of usernames to receive alerts from this integration.",
			},
			"inbound_template_id": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
				MarkdownDescription: "The ID of the inbound template to use for this integration.",
			},
			"api_url": schema.StringAttribute{
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "The generated endpoint URL that monitoring tools post alerts to. Contains a secret token, so it is marked sensitive.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"extra_attributes": schema.MapAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Additional fields to pass directly to the AlertOps API. If the API rejects a field or value, the raw API error is returned. Use this for complex nested fields like `api_settings`, `email_settings`, etc.",
			},
		},
	}
}

func (r *InboundIntegrationResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config inboundIntegrationModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if isKnown(config.EscalationPolicy) && config.EscalationPolicy.ValueString() != "" &&
		isKnown(config.ResponsePlay) && config.ResponsePlay.ValueString() != "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("response_play"),
			"Conflicting attributes",
			"An inbound integration can use either an escalation_policy or a response_play, not both.",
		)
	}
}

func (r *InboundIntegrationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *Client, got: %T", req.ProviderData))
		return
	}
	r.client = client
}

// buildInboundBasicPayload builds the fields accepted by both the create and `/basic` update endpoints.
func buildInboundBasicPayload(ctx context.Context, plan *inboundIntegrationModel) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"inbound_integration_name": plan.InboundIntegrationName.ValueString(),
		"type":                     plan.Type.ValueString(),
		"sequence":                 plan.Sequence.ValueInt64(),
		"enabled":                  plan.Enabled.ValueBool(),
		"inbound_template_id":      plan.InboundTemplateID.ValueInt64(),
	}
	if isKnown(plan.ResponsePlay) && plan.ResponsePlay.ValueString() != "" {
		payload["response_play"] = plan.ResponsePlay.ValueString()
	} else if v := plan.EscalationPolicy.ValueString(); isKnown(plan.EscalationPolicy) && v != "" {
		payload["escalation_policy"] = v
	}
	if !plan.RecipientGroups.IsNull() && !plan.RecipientGroups.IsUnknown() {
		var groups []string
		if diags := plan.RecipientGroups.ElementsAs(ctx, &groups, false); diags.HasError() {
			return nil, fmt.Errorf("error reading recipient_groups")
		}
		payload["recipient_groups"] = groups
	}
	if !plan.RecipientUsers.IsNull() && !plan.RecipientUsers.IsUnknown() {
		var users []string
		if diags := plan.RecipientUsers.ElementsAs(ctx, &users, false); diags.HasError() {
			return nil, fmt.Errorf("error reading recipient_users")
		}
		payload["recipient_users"] = users
	}
	return payload, nil
}

func buildInboundPayload(ctx context.Context, plan *inboundIntegrationModel) (map[string]interface{}, error) {
	payload, err := buildInboundBasicPayload(ctx, plan)
	if err != nil {
		return nil, err
	}
	mergeExtraAttributes(ctx, plan.ExtraAttributes, payload)
	if _, ok := payload["api_settings"]; !ok && strings.EqualFold(plan.Type.ValueString(), "API") {
		payload["api_settings"] = defaultAPISettings()
	}
	return payload, nil
}

func hasExtraAttributes(m types.Map) bool {
	return !m.IsNull() && !m.IsUnknown() && len(m.Elements()) > 0
}

func parseInboundFromAPI(ctx context.Context, result map[string]interface{}, state *inboundIntegrationModel) error {
	if v, ok := result["inbound_integration_name"].(string); ok {
		state.InboundIntegrationName = types.StringValue(v)
	}
	if v, ok := result["type"].(string); ok {
		state.Type = types.StringValue(v)
	}
	if v, ok := result["sequence"].(float64); ok {
		state.Sequence = types.Int64Value(int64(v))
	}
	if v, ok := result["enabled"].(bool); ok {
		state.Enabled = types.BoolValue(v)
	}
	if v, ok := result["escalation_policy_name"].(string); ok {
		state.EscalationPolicy = types.StringValue(v)
	} else if v, ok := result["escalation_policy"].(string); ok {
		state.EscalationPolicy = types.StringValue(v)
	} else {
		state.EscalationPolicy = types.StringValue("")
	}
	if id := mapInt64(result, "response_play_id"); id > 0 {
		state.ResponsePlay = types.StringValue(fmt.Sprintf("%d", id))
	} else {
		state.ResponsePlay = types.StringValue("")
	}
	state.ResponsePlayName = types.StringValue(mapString(result, "response_play_name"))
	if v, ok := result["inbound_template_id"].(float64); ok {
		state.InboundTemplateID = types.Int64Value(int64(v))
	}
	state.APIURL = types.StringValue(mapString(result, "api_url"))

	if rawGroups, ok := result["recipient_groups"].([]interface{}); ok {
		var groups []string
		for _, g := range rawGroups {
			if s, ok := g.(string); ok {
				groups = append(groups, s)
			}
		}
		list, diags := types.ListValueFrom(ctx, types.StringType, groups)
		if diags.HasError() {
			return fmt.Errorf("error building recipient_groups list")
		}
		state.RecipientGroups = list
	} else {
		state.RecipientGroups = types.ListValueMust(types.StringType, nil)
	}

	if rawUsers, ok := result["recipient_users"].([]interface{}); ok {
		var users []string
		for _, u := range rawUsers {
			if s, ok := u.(string); ok {
				users = append(users, s)
			}
		}
		list, diags := types.ListValueFrom(ctx, types.StringType, users)
		if diags.HasError() {
			return fmt.Errorf("error building recipient_users list")
		}
		state.RecipientUsers = list
	} else {
		state.RecipientUsers = types.ListValueMust(types.StringType, nil)
	}

	return nil
}

func (r *InboundIntegrationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan inboundIntegrationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload, err := buildInboundPayload(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Error building inbound integration payload", err.Error())
		return
	}

	respBody, err := r.client.Post(ctx, "/integrations/inbound", payload)
	if err != nil {
		resp.Diagnostics.AddError("Error creating inbound integration", err.Error())
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing inbound integration response", err.Error())
		return
	}

	if id, ok := result["inbound_integration_id"].(float64); ok {
		plan.ID = types.StringValue(fmt.Sprintf("%d", int(id)))
	} else if idStr, ok := result["inbound_integration_id"].(string); ok {
		plan.ID = types.StringValue(idStr)
	} else {
		resp.Diagnostics.AddError("Error creating inbound integration",
			fmt.Sprintf("Could not get inbound_integration_id from response: %s", string(respBody)))
		return
	}

	if err := parseInboundFromAPI(ctx, result, &plan); err != nil {
		resp.Diagnostics.AddError("Error processing inbound integration response", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *InboundIntegrationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state inboundIntegrationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	respBody, err := r.client.Get(ctx, "/integrations/inbound/"+state.ID.ValueString())
	if err != nil {
		if IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading inbound integration", err.Error())
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing inbound integration response", err.Error())
		return
	}

	if err := parseInboundFromAPI(ctx, result, &state); err != nil {
		resp.Diagnostics.AddError("Error processing inbound integration response", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *InboundIntegrationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan inboundIntegrationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state inboundIntegrationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	basePath := "/integrations/inbound/" + state.ID.ValueString()

	if hasExtraAttributes(plan.ExtraAttributes) {
		// Full update: the API requires the complete payload (including api_settings for
		// API integrations), so keep the current settings unless they are overridden.
		payload, err := buildInboundPayload(ctx, &plan)
		if err != nil {
			resp.Diagnostics.AddError("Error building inbound integration payload", err.Error())
			return
		}
		if extra := plan.ExtraAttributes.Elements(); extra["api_settings"] == nil && strings.EqualFold(plan.Type.ValueString(), "API") {
			currentBody, err := r.client.Get(ctx, basePath)
			if err != nil {
				resp.Diagnostics.AddError("Error reading inbound integration before update", err.Error())
				return
			}
			current, err := decodeObject(currentBody)
			if err != nil {
				resp.Diagnostics.AddError("Error parsing inbound integration response", err.Error())
				return
			}
			if settings, ok := current["api_settings"].(map[string]interface{}); ok {
				payload["api_settings"] = settings
			}
		}
		if _, err := r.client.Put(ctx, basePath, payload); err != nil {
			resp.Diagnostics.AddError("Error updating inbound integration", err.Error())
			return
		}
	} else {
		payload, err := buildInboundBasicPayload(ctx, &plan)
		if err != nil {
			resp.Diagnostics.AddError("Error building inbound integration payload", err.Error())
			return
		}
		if _, err := r.client.Put(ctx, basePath+"/basic", payload); err != nil {
			resp.Diagnostics.AddError("Error updating inbound integration", err.Error())
			return
		}
	}

	respBody, err := r.client.Get(ctx, basePath)
	if err != nil {
		resp.Diagnostics.AddError("Error reading inbound integration after update", err.Error())
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing inbound integration response", err.Error())
		return
	}

	plan.ID = state.ID
	if err := parseInboundFromAPI(ctx, result, &plan); err != nil {
		resp.Diagnostics.AddError("Error processing inbound integration response", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *InboundIntegrationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state inboundIntegrationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.Delete(ctx, "/integrations/inbound/"+state.ID.ValueString())
	if err != nil {
		if IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting inbound integration", err.Error())
		return
	}
}

func (r *InboundIntegrationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
