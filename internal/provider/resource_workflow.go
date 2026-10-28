package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &WorkflowResource{}
	_ resource.ResourceWithConfigure   = &WorkflowResource{}
	_ resource.ResourceWithImportState = &WorkflowResource{}
)

func NewWorkflowResource() resource.Resource {
	return &WorkflowResource{}
}

type WorkflowResource struct {
	client *Client
}

var wfConditionAttrTypes = map[string]attr.Type{
	"type":     types.StringType,
	"match":    types.StringType,
	"name":     types.StringType,
	"operator": types.StringType,
	"value":    types.StringType,
	"list_id":  types.StringType,
}

var wfActionAttrTypes = map[string]attr.Type{
	"name":                        types.StringType,
	"value":                       types.StringType,
	"webhook_url":                 types.StringType,
	"send_to_original_recipients": types.BoolType,
	"send_to_sender":              types.BoolType,
	"send_to_owner":               types.BoolType,
	"launch_new_thread":           types.BoolType,
	"subject":                     types.StringType,
	"message_text":                types.StringType,
	"users":                       types.ListType{ElemType: types.StringType},
	"groups":                      types.ListType{ElemType: types.StringType},
	"is_escalation_policy":        types.BoolType,
}

type workflowModel struct {
	ID                 types.String `tfsdk:"id"`
	WorkflowName       types.String `tfsdk:"workflow_name"`
	WorkflowType       types.String `tfsdk:"workflow_type"`
	AlertType          types.String `tfsdk:"alert_type"`
	Enabled            types.Bool   `tfsdk:"enabled"`
	Scheduled          types.Bool   `tfsdk:"scheduled"`
	RecurrenceInterval types.Int64  `tfsdk:"recurrence_interval"`
	Conditions         types.List   `tfsdk:"conditions"`
	Actions            types.List   `tfsdk:"actions"`
}

func (r *WorkflowResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workflow"
}

func (r *WorkflowResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	emptyStringList := listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{}))
	optionalString := func(description string) schema.StringAttribute {
		return schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString(""), MarkdownDescription: description}
	}
	optionalBool := func(description string) schema.BoolAttribute {
		return schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false), MarkdownDescription: description}
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an AlertOps workflow: start conditions plus actions such as `SendMessage`, run when alerts change (e.g. on acknowledgement or closure). Attach it to an escalation policy or response play through their `workflows` attribute.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique identifier of the workflow.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"workflow_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the workflow.",
			},
			"workflow_type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The workflow type: `Alert`, `Message` or `Notification`. Changing it forces a new workflow.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"alert_type": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The alert type name the workflow applies to (e.g. `Standard Alert`).",
			},
			"enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Whether the workflow is enabled. Only enabled workflows can be attached to escalation policies and response plays.",
			},
			"scheduled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Whether the workflow runs on a schedule.",
			},
			"recurrence_interval": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
				MarkdownDescription: "Recurrence interval for scheduled workflows.",
			},
			"conditions": schema.ListNestedAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Conditions that start the workflow. Use `match = \"all\"` for conditions that must all be met and `match = \"any\"` for conditions of which one must be met.",
				Default:             listdefault.StaticValue(types.ListValueMust(types.ObjectType{AttrTypes: wfConditionAttrTypes}, []attr.Value{})),
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Optional:            true,
							Computed:            true,
							Default:             stringdefault.StaticString("start"),
							MarkdownDescription: "The condition type. Defaults to `start`.",
						},
						"match": schema.StringAttribute{
							Optional:            true,
							Computed:            true,
							Default:             stringdefault.StaticString("all"),
							MarkdownDescription: "`all` or `any`. Defaults to `all`.",
							Validators:          []validator.String{oneOfStringValidator{values: []string{"all", "any"}}},
						},
						"name": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The field the condition tests, e.g. `AlertStatus`, `Topic`, `Owner`.",
						},
						"operator": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The comparison operator, e.g. `is`, `is not`.",
						},
						"value":   optionalString("The value to compare with, e.g. `Assigned` or `Closed` for `AlertStatus`."),
						"list_id": optionalString("Optional list identifier used by list-based conditions."),
					},
				},
			},
			"actions": schema.ListNestedAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Actions run when the workflow starts.",
				Default:             listdefault.StaticValue(types.ListValueMust(types.ObjectType{AttrTypes: wfActionAttrTypes}, []attr.Value{})),
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The action name, e.g. `SendMessage`, `Send Webhook`, `Update Status`, `Update Priority`, `CancelDeliveryThread`.",
						},
						"value":                       optionalString("The action value. For `SendMessage`, the name of the escalation policy or response play used to deliver the message."),
						"webhook_url":                 optionalString("Webhook URL for webhook actions."),
						"send_to_original_recipients": optionalBool("Send the message to the alert's original recipients."),
						"send_to_sender":              optionalBool("Send the message to the sender."),
						"send_to_owner":               optionalBool("Send the message to the alert owner."),
						"launch_new_thread":           optionalBool("Launch a new message thread."),
						"subject":                     optionalString("Message subject."),
						"message_text":                optionalString("Message body. HTML is allowed."),
						"users": schema.ListAttribute{
							Optional:            true,
							Computed:            true,
							ElementType:         types.StringType,
							Default:             emptyStringList,
							MarkdownDescription: "Usernames to notify.",
						},
						"groups": schema.ListAttribute{
							Optional:            true,
							Computed:            true,
							ElementType:         types.StringType,
							Default:             emptyStringList,
							MarkdownDescription: "Group names to notify.",
						},
						"is_escalation_policy": schema.BoolAttribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "For `SendMessage`: `true` if `value` names an escalation policy, `false` if it names a response play. Determined by AlertOps when omitted.",
						},
					},
				},
			},
		},
	}
}

func (r *WorkflowResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ---------------------------------------------------------------------------
// API <-> model conversion
// ---------------------------------------------------------------------------

func conditionFromAPI(m map[string]interface{}) types.Object {
	return types.ObjectValueMust(wfConditionAttrTypes, map[string]attr.Value{
		"type":     types.StringValue(mapString(m, "type")),
		"match":    types.StringValue(mapString(m, "match")),
		"name":     types.StringValue(mapString(m, "name")),
		"operator": types.StringValue(mapString(m, "operator")),
		"value":    types.StringValue(mapScalarString(m, "value")),
		"list_id":  types.StringValue(mapScalarString(m, "list_id")),
	})
}

func actionFromAPI(m map[string]interface{}) types.Object {
	return types.ObjectValueMust(wfActionAttrTypes, map[string]attr.Value{
		"name":                        types.StringValue(mapString(m, "name")),
		"value":                       types.StringValue(mapScalarString(m, "value")),
		"webhook_url":                 types.StringValue(mapString(m, "webhook_url")),
		"send_to_original_recipients": types.BoolValue(mapBool(m, "send_to_original_recipients")),
		"send_to_sender":              types.BoolValue(mapBool(m, "send_to_sender")),
		"send_to_owner":               types.BoolValue(mapBool(m, "send_to_owner")),
		"launch_new_thread":           types.BoolValue(mapBool(m, "launch_new_thread")),
		"subject":                     types.StringValue(mapString(m, "subject")),
		"message_text":                types.StringValue(mapString(m, "message_text")),
		"users":                       stringListFromAPI(mapArray(m, "users")),
		"groups":                      stringListFromAPI(mapArray(m, "groups")),
		"is_escalation_policy":        types.BoolValue(mapBool(m, "is_escalation_policy")),
	})
}

func parseWorkflowFromAPI(result map[string]interface{}, state *workflowModel) {
	state.WorkflowName = types.StringValue(mapString(result, "workflow_name"))
	if v := mapString(result, "workflow_type_name"); v != "" {
		state.WorkflowType = types.StringValue(v)
	} else {
		state.WorkflowType = types.StringValue(mapString(result, "workflow_type"))
	}
	state.AlertType = types.StringValue(mapString(result, "alert_type"))
	state.Enabled = types.BoolValue(mapBool(result, "enabled"))
	state.Scheduled = types.BoolValue(mapBool(result, "scheduled"))
	state.RecurrenceInterval = types.Int64Value(mapInt64(result, "recurrence_interval"))

	conditions := []attr.Value{}
	for _, item := range mapArray(result, "conditions") {
		if m, ok := item.(map[string]interface{}); ok {
			conditions = append(conditions, conditionFromAPI(m))
		}
	}
	state.Conditions = types.ListValueMust(types.ObjectType{AttrTypes: wfConditionAttrTypes}, conditions)

	actions := []attr.Value{}
	for _, item := range mapArray(result, "actions") {
		if m, ok := item.(map[string]interface{}); ok {
			actions = append(actions, actionFromAPI(m))
		}
	}
	state.Actions = types.ListValueMust(types.ObjectType{AttrTypes: wfActionAttrTypes}, actions)
}

func conditionPayload(obj types.Object) map[string]interface{} {
	a := obj.Attributes()
	return map[string]interface{}{
		"type":     a["type"].(types.String).ValueString(),
		"match":    a["match"].(types.String).ValueString(),
		"name":     a["name"].(types.String).ValueString(),
		"operator": a["operator"].(types.String).ValueString(),
		"value":    a["value"].(types.String).ValueString(),
		"list_id":  a["list_id"].(types.String).ValueString(),
	}
}

func actionPayload(obj types.Object) map[string]interface{} {
	a := obj.Attributes()
	payload := map[string]interface{}{
		"name":                        a["name"].(types.String).ValueString(),
		"value":                       a["value"].(types.String).ValueString(),
		"send_to_original_recipients": a["send_to_original_recipients"].(types.Bool).ValueBool(),
		"send_to_sender":              a["send_to_sender"].(types.Bool).ValueBool(),
		"send_to_owner":               a["send_to_owner"].(types.Bool).ValueBool(),
		"launch_new_thread":           a["launch_new_thread"].(types.Bool).ValueBool(),
		"subject":                     a["subject"].(types.String).ValueString(),
		"message_text":                a["message_text"].(types.String).ValueString(),
		"users":                       stringsFromElements(a["users"].(types.List).Elements()),
		"groups":                      stringsFromElements(a["groups"].(types.List).Elements()),
	}
	if v := a["webhook_url"].(types.String).ValueString(); v != "" {
		payload["webhook_url"] = v
	}
	if v := a["is_escalation_policy"].(types.Bool); isKnown(v) {
		payload["is_escalation_policy"] = v.ValueBool()
	}
	return payload
}

// itemsEqual compares planned items with the API's, ignoring attributes still unknown in the plan.
func itemsEqual(planned []attr.Value, current []types.Object) bool {
	if len(planned) != len(current) {
		return false
	}
	for i, p := range planned {
		pa := p.(types.Object).Attributes()
		ca := current[i].Attributes()
		for key, v := range pa {
			if v.IsUnknown() {
				continue
			}
			if !v.Equal(ca[key]) {
				return false
			}
		}
	}
	return true
}

// ---------------------------------------------------------------------------
// Sync helpers
// ---------------------------------------------------------------------------

func (r *WorkflowResource) get(ctx context.Context, id string) (map[string]interface{}, error) {
	body, err := r.client.Get(ctx, "/workflows/"+id)
	if err != nil {
		return nil, err
	}
	return decodeObject(body)
}

// syncItems replaces a workflow's conditions or actions ("conditions"/"actions") when they differ.
func (r *WorkflowResource) syncItems(ctx context.Context, id, kind, idKey string, plan types.List, current map[string]interface{},
	fromAPI func(map[string]interface{}) types.Object, toPayload func(types.Object) map[string]interface{}) error {
	if !isKnown(plan) {
		return nil
	}
	var existing []types.Object
	var existingIDs []string
	for _, item := range mapArray(current, kind) {
		if m, ok := item.(map[string]interface{}); ok {
			existing = append(existing, fromAPI(m))
			itemID, _ := idFromResult(m, idKey)
			existingIDs = append(existingIDs, itemID)
		}
	}
	if itemsEqual(plan.Elements(), existing) {
		return nil
	}
	for _, itemID := range existingIDs {
		if _, err := r.client.Delete(ctx, fmt.Sprintf("/workflows/%s/%s/%s", id, kind, itemID)); err != nil && !IsNotFound(err) {
			return fmt.Errorf("error removing workflow %s %s: %w", kind, itemID, err)
		}
	}
	for i, elem := range plan.Elements() {
		if _, err := r.client.Post(ctx, fmt.Sprintf("/workflows/%s/%s", id, kind), toPayload(elem.(types.Object))); err != nil {
			return fmt.Errorf("error adding workflow %s %d: %w", kind, i+1, err)
		}
	}
	return nil
}

func (r *WorkflowResource) syncConditionsAndActions(ctx context.Context, id string, plan *workflowModel, current map[string]interface{}) error {
	if err := r.syncItems(ctx, id, "conditions", "condition_id", plan.Conditions, current, conditionFromAPI, conditionPayload); err != nil {
		return err
	}
	return r.syncItems(ctx, id, "actions", "action_id", plan.Actions, current, actionFromAPI, actionPayload)
}

func basicWorkflowPayload(plan *workflowModel, alertType string) map[string]interface{} {
	return map[string]interface{}{
		"workflow_name":       plan.WorkflowName.ValueString(),
		"enabled":             plan.Enabled.ValueBool(),
		"alert_type":          alertType,
		"scheduled":           plan.Scheduled.ValueBool(),
		"recurrence_interval": plan.RecurrenceInterval.ValueInt64(),
	}
}

// ---------------------------------------------------------------------------
// CRUD
// ---------------------------------------------------------------------------

func (r *WorkflowResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan workflowModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create disabled, add conditions and actions, then enable. Adding them one by one
	// (rather than inline) preserves fields such as the action subject.
	payload := map[string]interface{}{
		"workflow_name":       plan.WorkflowName.ValueString(),
		"workflow_type":       plan.WorkflowType.ValueString(),
		"enabled":             false,
		"scheduled":           plan.Scheduled.ValueBool(),
		"recurrence_interval": plan.RecurrenceInterval.ValueInt64(),
		"conditions":          []interface{}{},
		"actions":             []interface{}{},
	}
	if isKnown(plan.AlertType) && plan.AlertType.ValueString() != "" {
		payload["alert_type"] = plan.AlertType.ValueString()
	}

	respBody, err := r.client.Post(ctx, "/workflows", payload)
	if err != nil {
		resp.Diagnostics.AddError("Error creating workflow", err.Error())
		return
	}
	current, err := decodeObject(respBody)
	if err != nil {
		resp.Diagnostics.AddError("Error parsing workflow response", err.Error())
		return
	}
	id, ok := idFromResult(current, "workflow_id")
	if !ok {
		resp.Diagnostics.AddError("Error creating workflow",
			fmt.Sprintf("Could not get workflow_id from response: %s", string(respBody)))
		return
	}
	plan.ID = types.StringValue(id)
	// Save the ID immediately so a failure below does not orphan the workflow.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), plan.ID)...)

	if err := r.syncConditionsAndActions(ctx, id, &plan, current); err != nil {
		resp.Diagnostics.AddError("Error configuring workflow", err.Error())
		return
	}

	if plan.Enabled.ValueBool() {
		if _, err := r.client.Put(ctx, "/workflows/"+id+"/status", map[string]interface{}{"enabled": true}); err != nil {
			resp.Diagnostics.AddError("Error enabling workflow", err.Error())
			return
		}
	}

	result, err := r.get(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading workflow after create", err.Error())
		return
	}
	parseWorkflowFromAPI(result, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *WorkflowResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state workflowModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.get(ctx, state.ID.ValueString())
	if err != nil {
		if IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading workflow", err.Error())
		return
	}
	parseWorkflowFromAPI(result, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *WorkflowResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state workflowModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = state.ID
	id := state.ID.ValueString()

	current, err := r.get(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading workflow before update", err.Error())
		return
	}

	alertType := mapString(current, "alert_type")
	if isKnown(plan.AlertType) && plan.AlertType.ValueString() != "" {
		alertType = plan.AlertType.ValueString()
	}
	// Use the /basic endpoint: PUT /workflows/{id} replaces conditions and actions too.
	if plan.WorkflowName.ValueString() != mapString(current, "workflow_name") ||
		alertType != mapString(current, "alert_type") ||
		plan.Scheduled.ValueBool() != mapBool(current, "scheduled") ||
		plan.RecurrenceInterval.ValueInt64() != mapInt64(current, "recurrence_interval") ||
		plan.Enabled.ValueBool() != mapBool(current, "enabled") {
		if _, err := r.client.Put(ctx, "/workflows/"+id+"/basic", basicWorkflowPayload(&plan, alertType)); err != nil {
			resp.Diagnostics.AddError("Error updating workflow", err.Error())
			return
		}
	}

	if err := r.syncConditionsAndActions(ctx, id, &plan, current); err != nil {
		resp.Diagnostics.AddError("Error updating workflow", err.Error())
		return
	}

	result, err := r.get(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading workflow after update", err.Error())
		return
	}
	parseWorkflowFromAPI(result, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *WorkflowResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state workflowModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := r.client.Delete(ctx, "/workflows/"+state.ID.ValueString()); err != nil && !IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting workflow", err.Error())
	}
}

func (r *WorkflowResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
