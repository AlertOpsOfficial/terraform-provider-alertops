package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &MaintenanceWindowResource{}
	_ resource.ResourceWithConfigure   = &MaintenanceWindowResource{}
	_ resource.ResourceWithImportState = &MaintenanceWindowResource{}
)

func NewMaintenanceWindowResource() resource.Resource {
	return &MaintenanceWindowResource{}
}

type MaintenanceWindowResource struct {
	client *Client
}

type maintenanceWindowModel struct {
	ID              types.String `tfsdk:"id"`
	Description     types.String `tfsdk:"description"`
	Recurrence      types.String `tfsdk:"recurrence"`
	Integration     types.List   `tfsdk:"integration"`
	StartDate       types.String `tfsdk:"start_date"`
	EndDate         types.String `tfsdk:"end_date"`
	StartTime       types.String `tfsdk:"start_time"`
	EndTime         types.String `tfsdk:"end_time"`
	Sunday          types.Bool   `tfsdk:"sunday"`
	Monday          types.Bool   `tfsdk:"monday"`
	Tuesday         types.Bool   `tfsdk:"tuesday"`
	Wednesday       types.Bool   `tfsdk:"wednesday"`
	Thursday        types.Bool   `tfsdk:"thursday"`
	Friday          types.Bool   `tfsdk:"friday"`
	Saturday        types.Bool   `tfsdk:"saturday"`
	ExtraAttributes types.Map    `tfsdk:"extra_attributes"`
}

func (r *MaintenanceWindowResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_maintenance_window"
}

func (r *MaintenanceWindowResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an AlertOps maintenance window. Any additional fields supported by the AlertOps API can be passed via `extra_attributes`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique identifier of the maintenance window.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"description": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "A description of the maintenance window.",
			},
			"recurrence": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The recurrence pattern for the maintenance window (e.g., `None`, `Daily`, `Weekly`, `Monthly`).",
			},
			"integration": schema.ListAttribute{
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "List of integration names or IDs to include in the maintenance window.",
			},
			"start_date": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The start date of the maintenance window (e.g., `2024-01-01`).",
			},
			"end_date": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The end date of the maintenance window (e.g., `2024-01-02`).",
			},
			"start_time": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The start time of the maintenance window (e.g., `08:00`).",
			},
			"end_time": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The end time of the maintenance window (e.g., `17:00`).",
			},
			"sunday": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Whether Sunday is included in the maintenance window.",
			},
			"monday": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Whether Monday is included in the maintenance window.",
			},
			"tuesday": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Whether Tuesday is included in the maintenance window.",
			},
			"wednesday": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Whether Wednesday is included in the maintenance window.",
			},
			"thursday": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Whether Thursday is included in the maintenance window.",
			},
			"friday": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Whether Friday is included in the maintenance window.",
			},
			"saturday": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Whether Saturday is included in the maintenance window.",
			},
			"extra_attributes": schema.MapAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Additional fields to pass directly to the AlertOps API. If the API rejects a field or value, the raw API error is returned.",
			},
		},
	}
}

func (r *MaintenanceWindowResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func buildMWPayload(ctx context.Context, plan *maintenanceWindowModel) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"sunday":    plan.Sunday.ValueBool(),
		"monday":    plan.Monday.ValueBool(),
		"tuesday":   plan.Tuesday.ValueBool(),
		"wednesday": plan.Wednesday.ValueBool(),
		"thursday":  plan.Thursday.ValueBool(),
		"friday":    plan.Friday.ValueBool(),
		"saturday":  plan.Saturday.ValueBool(),
	}
	if v := plan.Description.ValueString(); v != "" {
		payload["description"] = v
	}
	if v := plan.Recurrence.ValueString(); v != "" {
		payload["recurrence"] = v
	}
	if v := plan.StartDate.ValueString(); v != "" {
		payload["start_date"] = v
	}
	if v := plan.EndDate.ValueString(); v != "" {
		payload["end_date"] = v
	}
	if v := plan.StartTime.ValueString(); v != "" {
		payload["start_time"] = v
	}
	if v := plan.EndTime.ValueString(); v != "" {
		payload["end_time"] = v
	}
	if !plan.Integration.IsNull() && !plan.Integration.IsUnknown() {
		var integrations []string
		if diags := plan.Integration.ElementsAs(ctx, &integrations, false); diags.HasError() {
			return nil, fmt.Errorf("error reading integration list")
		}
		payload["integration"] = integrations
	}
	mergeExtraAttributes(ctx, plan.ExtraAttributes, payload)
	return payload, nil
}

func parseMWFromAPI(ctx context.Context, result map[string]interface{}, state *maintenanceWindowModel) error {
	if v, ok := result["description"].(string); ok {
		state.Description = types.StringValue(v)
	} else {
		state.Description = types.StringValue("")
	}
	if v, ok := result["recurrence"].(string); ok {
		state.Recurrence = types.StringValue(v)
	} else {
		state.Recurrence = types.StringValue("")
	}
	if v, ok := result["start_date"].(string); ok {
		state.StartDate = types.StringValue(v)
	} else {
		state.StartDate = types.StringValue("")
	}
	if v, ok := result["end_date"].(string); ok {
		state.EndDate = types.StringValue(v)
	} else {
		state.EndDate = types.StringValue("")
	}
	if v, ok := result["start_time"].(string); ok {
		state.StartTime = types.StringValue(v)
	} else {
		state.StartTime = types.StringValue("")
	}
	if v, ok := result["end_time"].(string); ok {
		state.EndTime = types.StringValue(v)
	} else {
		state.EndTime = types.StringValue("")
	}
	getBool := func(key string) bool {
		if v, ok := result[key].(bool); ok {
			return v
		}
		return false
	}
	state.Sunday = types.BoolValue(getBool("sunday"))
	state.Monday = types.BoolValue(getBool("monday"))
	state.Tuesday = types.BoolValue(getBool("tuesday"))
	state.Wednesday = types.BoolValue(getBool("wednesday"))
	state.Thursday = types.BoolValue(getBool("thursday"))
	state.Friday = types.BoolValue(getBool("friday"))
	state.Saturday = types.BoolValue(getBool("saturday"))

	if rawInts, ok := result["integration"].([]interface{}); ok {
		var integrations []string
		for _, i := range rawInts {
			switch v := i.(type) {
			case string:
				integrations = append(integrations, v)
			case map[string]interface{}:
				// API returns integration as {"id": 123, "name": "..."}
				if id, ok := v["id"].(float64); ok {
					integrations = append(integrations, fmt.Sprintf("%d", int(id)))
				} else if idStr, ok := v["id"].(string); ok {
					integrations = append(integrations, idStr)
				}
			}
		}
		list, diags := types.ListValueFrom(ctx, types.StringType, integrations)
		if diags.HasError() {
			return fmt.Errorf("error building integration list")
		}
		state.Integration = list
	} else {
		state.Integration = types.ListValueMust(types.StringType, nil)
	}
	return nil
}

func (r *MaintenanceWindowResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan maintenanceWindowModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload, err := buildMWPayload(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Error building maintenance window payload", err.Error())
		return
	}

	respBody, err := r.client.Post(ctx, "/maintenancewindows", payload)
	if err != nil {
		resp.Diagnostics.AddError("Error creating maintenance window", err.Error())
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing maintenance window response", err.Error())
		return
	}

	if id, ok := result["maintenance_window_id"].(float64); ok {
		plan.ID = types.StringValue(fmt.Sprintf("%d", int(id)))
	} else if idStr, ok := result["maintenance_window_id"].(string); ok {
		plan.ID = types.StringValue(idStr)
	} else {
		resp.Diagnostics.AddError("Error creating maintenance window",
			fmt.Sprintf("Could not get maintenance_window_id from response: %s", string(respBody)))
		return
	}

	if err := parseMWFromAPI(ctx, result, &plan); err != nil {
		resp.Diagnostics.AddError("Error processing maintenance window response", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *MaintenanceWindowResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state maintenanceWindowModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	respBody, err := r.client.Get(ctx, "/maintenancewindows/"+state.ID.ValueString())
	if err != nil {
		if IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading maintenance window", err.Error())
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing maintenance window response", err.Error())
		return
	}

	if err := parseMWFromAPI(ctx, result, &state); err != nil {
		resp.Diagnostics.AddError("Error processing maintenance window response", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *MaintenanceWindowResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan maintenanceWindowModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state maintenanceWindowModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload, err := buildMWPayload(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Error building maintenance window payload", err.Error())
		return
	}

	if _, err := r.client.Put(ctx, "/maintenancewindows/"+state.ID.ValueString(), payload); err != nil {
		resp.Diagnostics.AddError("Error updating maintenance window", err.Error())
		return
	}

	// The PUT returns 204 No Content, so read the window back.
	respBody, err := r.client.Get(ctx, "/maintenancewindows/"+state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading maintenance window after update", err.Error())
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing maintenance window response", err.Error())
		return
	}

	plan.ID = state.ID
	if err := parseMWFromAPI(ctx, result, &plan); err != nil {
		resp.Diagnostics.AddError("Error processing maintenance window response", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *MaintenanceWindowResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state maintenanceWindowModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.Delete(ctx, "/maintenancewindows/"+state.ID.ValueString())
	if err != nil {
		if IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting maintenance window", err.Error())
		return
	}
}

func (r *MaintenanceWindowResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
