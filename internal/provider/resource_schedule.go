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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &ScheduleResource{}
	_ resource.ResourceWithConfigure   = &ScheduleResource{}
	_ resource.ResourceWithImportState = &ScheduleResource{}
)

func NewScheduleResource() resource.Resource {
	return &ScheduleResource{}
}

type ScheduleResource struct {
	client *Client
}

type scheduleModel struct {
	ID                     types.String `tfsdk:"id"`
	GroupID                types.String `tfsdk:"group_id"`
	Group                  types.String `tfsdk:"group"`
	ScheduleName           types.String `tfsdk:"schedule_name"`
	ScheduleType           types.String `tfsdk:"schedule_type"`
	Continuous             types.Bool   `tfsdk:"continuous"`
	IncludeAllUsersInGroup types.Bool   `tfsdk:"include_all_users_in_group"`
	Enabled                types.Bool   `tfsdk:"enabled"`
	TimeZone               types.String `tfsdk:"time_zone"`
	Color                  types.String `tfsdk:"color"`
	StartWeekday           types.String `tfsdk:"start_weekday"`
	EndWeekday             types.String `tfsdk:"end_weekday"`
	WeekdaySun             types.Bool   `tfsdk:"weekday_sun"`
	WeekdayMon             types.Bool   `tfsdk:"weekday_mon"`
	WeekdayTue             types.Bool   `tfsdk:"weekday_tue"`
	WeekdayWed             types.Bool   `tfsdk:"weekday_wed"`
	WeekdayThu             types.Bool   `tfsdk:"weekday_thu"`
	WeekdayFri             types.Bool   `tfsdk:"weekday_fri"`
	WeekdaySat             types.Bool   `tfsdk:"weekday_sat"`
	ExtraAttributes        types.Map    `tfsdk:"extra_attributes"`
}

func (r *ScheduleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_schedule"
}

func (r *ScheduleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an AlertOps on-call schedule for a group. The group must have at least one member for the schedule to be retrievable via the API. Any additional fields can be passed via `extra_attributes` and will be merged into the request payload — if the API rejects a value, the raw API error is returned.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The composite identifier of the schedule in the format `group_id/schedule_id`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"group_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The numeric ID of the group this schedule belongs to (populated after creation).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"group": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the group this schedule belongs to.",
			},
			"schedule_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the schedule.",
			},
			"schedule_type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The type of schedule. Valid values include `Fixed` and `Rotating`.",
			},
			"continuous": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Whether the schedule runs continuously without a defined start/end date.",
			},
			"include_all_users_in_group": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Whether to include all users in the group in this schedule.",
			},
			"enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Whether the schedule is active.",
			},
			"time_zone": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The time zone for the schedule (e.g., `(UTC-06:00) Central Time (US & Canada)`).",
			},
			"color": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The display color for the schedule in hex format (e.g., `#FF0000`).",
			},
			"start_weekday": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The day of the week the schedule starts (e.g., `Mon`, `Tue`, `Wed`, `Thu`, `Fri`, `Sat`, `Sun`).",
			},
			"end_weekday": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The day of the week the schedule ends (e.g., `Mon`, `Tue`, `Wed`, `Thu`, `Fri`, `Sat`, `Sun`).",
			},
			"weekday_sun": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Whether Sunday is an active day in the schedule.",
			},
			"weekday_mon": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Whether Monday is an active day in the schedule.",
			},
			"weekday_tue": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Whether Tuesday is an active day in the schedule.",
			},
			"weekday_wed": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Whether Wednesday is an active day in the schedule.",
			},
			"weekday_thu": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Whether Thursday is an active day in the schedule.",
			},
			"weekday_fri": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Whether Friday is an active day in the schedule.",
			},
			"weekday_sat": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Whether Saturday is an active day in the schedule.",
			},
			"extra_attributes": schema.MapAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Additional fields to pass directly to the AlertOps API. Values are strings; the provider merges them into the request payload. If the API rejects a field or value, the raw API error is returned.",
			},
		},
	}
}

func (r *ScheduleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ScheduleResource) buildPayload(ctx context.Context, plan *scheduleModel) map[string]interface{} {
	payload := map[string]interface{}{
		"group":                      plan.Group.ValueString(),
		"schedule_name":              plan.ScheduleName.ValueString(),
		"schedule_type":              plan.ScheduleType.ValueString(),
		"continuous":                 plan.Continuous.ValueBool(),
		"include_all_users_in_group": plan.IncludeAllUsersInGroup.ValueBool(),
		"enabled":                    plan.Enabled.ValueBool(),
		"time_zone":                  plan.TimeZone.ValueString(),
		"schedule_weekdays": map[string]interface{}{
			"sun": plan.WeekdaySun.ValueBool(),
			"mon": plan.WeekdayMon.ValueBool(),
			"tue": plan.WeekdayTue.ValueBool(),
			"wed": plan.WeekdayWed.ValueBool(),
			"thu": plan.WeekdayThu.ValueBool(),
			"fri": plan.WeekdayFri.ValueBool(),
			"sat": plan.WeekdaySat.ValueBool(),
		},
	}
	if !plan.Color.IsNull() && !plan.Color.IsUnknown() && plan.Color.ValueString() != "" {
		payload["color"] = plan.Color.ValueString()
	}
	if !plan.StartWeekday.IsNull() && !plan.StartWeekday.IsUnknown() && plan.StartWeekday.ValueString() != "" {
		payload["start_weekday"] = plan.StartWeekday.ValueString()
	}
	if !plan.EndWeekday.IsNull() && !plan.EndWeekday.IsUnknown() && plan.EndWeekday.ValueString() != "" {
		payload["end_weekday"] = plan.EndWeekday.ValueString()
	}
	mergeExtraAttributes(ctx, plan.ExtraAttributes, payload)
	return payload
}

func (r *ScheduleResource) applyAPIResponse(result map[string]interface{}, state *scheduleModel) {
	if v, ok := result["group"].(string); ok {
		state.GroupID = types.StringValue(v)
	}
	if v, ok := result["schedule_name"].(string); ok {
		state.ScheduleName = types.StringValue(v)
	}
	if v, ok := result["schedule_type"].(string); ok {
		state.ScheduleType = types.StringValue(v)
	}
	if v, ok := result["continuous"].(bool); ok {
		state.Continuous = types.BoolValue(v)
	}
	if v, ok := result["include_all_users_in_group"].(bool); ok {
		state.IncludeAllUsersInGroup = types.BoolValue(v)
	}
	if v, ok := result["enabled"].(bool); ok {
		state.Enabled = types.BoolValue(v)
	}
	if v, ok := result["time_zone"].(string); ok {
		state.TimeZone = types.StringValue(v)
	}
	if v, ok := result["color"].(string); ok {
		state.Color = types.StringValue(v)
	}
	if v, ok := result["start_weekday"].(string); ok {
		state.StartWeekday = types.StringValue(v)
	}
	if v, ok := result["end_weekday"].(string); ok {
		state.EndWeekday = types.StringValue(v)
	}
	if wd, ok := result["schedule_weekdays"].(map[string]interface{}); ok {
		getBool := func(k string) bool {
			if v, ok := wd[k].(bool); ok {
				return v
			}
			return false
		}
		state.WeekdaySun = types.BoolValue(getBool("sun"))
		state.WeekdayMon = types.BoolValue(getBool("mon"))
		state.WeekdayTue = types.BoolValue(getBool("tue"))
		state.WeekdayWed = types.BoolValue(getBool("wed"))
		state.WeekdayThu = types.BoolValue(getBool("thu"))
		state.WeekdayFri = types.BoolValue(getBool("fri"))
		state.WeekdaySat = types.BoolValue(getBool("sat"))
	}
}

func (r *ScheduleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan scheduleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload := r.buildPayload(ctx, &plan)

	respBody, err := r.client.Post(ctx, "/schedules", payload)
	if err != nil {
		resp.Diagnostics.AddError("Error creating schedule", err.Error())
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing schedule response", err.Error())
		return
	}

	scheduleID, ok := result["schedule_id"].(float64)
	if !ok {
		resp.Diagnostics.AddError("Error creating schedule", fmt.Sprintf("Could not get schedule_id from response: %s", string(respBody)))
		return
	}

	groupID, ok := result["group"].(string)
	if !ok {
		resp.Diagnostics.AddError("Error creating schedule", fmt.Sprintf("Could not get group from response: %s", string(respBody)))
		return
	}

	plan.GroupID = types.StringValue(groupID)
	plan.ID = types.StringValue(fmt.Sprintf("%s/%d", groupID, int(scheduleID)))

	r.applyAPIResponse(result, &plan)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ScheduleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state scheduleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	parts := strings.SplitN(state.ID.ValueString(), "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid schedule ID", fmt.Sprintf("Expected format group_id/schedule_id, got: %s", state.ID.ValueString()))
		return
	}
	groupID, scheduleID := parts[0], parts[1]

	respBody, err := r.client.Get(ctx, fmt.Sprintf("/schedules/%s/%s", groupID, scheduleID))
	if err != nil {
		if IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading schedule", err.Error())
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing schedule response", err.Error())
		return
	}

	r.applyAPIResponse(result, &state)

	// After import only the ID is known; the API returns the group ID, so look up its name.
	if state.Group.IsNull() || state.Group.ValueString() == "" {
		groupBody, err := r.client.Get(ctx, "/groups/"+groupID)
		if err != nil {
			resp.Diagnostics.AddError("Error reading schedule group", err.Error())
			return
		}
		group, err := decodeObject(groupBody)
		if err != nil {
			resp.Diagnostics.AddError("Error parsing schedule group", err.Error())
			return
		}
		state.Group = types.StringValue(mapString(group, "group_name"))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ScheduleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan scheduleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state scheduleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	parts := strings.SplitN(state.ID.ValueString(), "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid schedule ID", fmt.Sprintf("Expected format group_id/schedule_id, got: %s", state.ID.ValueString()))
		return
	}
	groupID, scheduleID := parts[0], parts[1]

	payload := r.buildPayload(ctx, &plan)

	if _, err := r.client.Put(ctx, fmt.Sprintf("/schedules/%s/%s", groupID, scheduleID), payload); err != nil {
		resp.Diagnostics.AddError("Error updating schedule", err.Error())
		return
	}

	// The PUT returns no body, and AlertOps derives some fields (e.g. weekdays from the
	// start/end weekday range), so read the schedule back.
	respBody, err := r.client.Get(ctx, fmt.Sprintf("/schedules/%s/%s", groupID, scheduleID))
	if err != nil {
		resp.Diagnostics.AddError("Error reading schedule after update", err.Error())
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing schedule response", err.Error())
		return
	}

	plan.ID = state.ID
	plan.GroupID = state.GroupID
	r.applyAPIResponse(result, &plan)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ScheduleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state scheduleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	parts := strings.SplitN(state.ID.ValueString(), "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid schedule ID", fmt.Sprintf("Expected format group_id/schedule_id, got: %s", state.ID.ValueString()))
		return
	}
	groupID, scheduleID := parts[0], parts[1]

	_, err := r.client.Delete(ctx, fmt.Sprintf("/schedules/%s/%s", groupID, scheduleID))
	if err != nil {
		if IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting schedule", err.Error())
		return
	}
}

func (r *ScheduleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
