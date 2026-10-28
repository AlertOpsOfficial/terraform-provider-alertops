package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
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
	_ resource.Resource                = &UserResource{}
	_ resource.ResourceWithConfigure   = &UserResource{}
	_ resource.ResourceWithImportState = &UserResource{}
)

func NewUserResource() resource.Resource {
	return &UserResource{}
}

type UserResource struct {
	client *Client
}

// contactMethodAttrTypes defines the Terraform attribute types for a contact method.
var contactMethodAttrTypes = map[string]attr.Type{
	"contact_method_name":   types.StringType,
	"wait_time_in_mins":     types.Int64Type,
	"repeat":                types.BoolType,
	"repeat_times":          types.Int64Type,
	"repeat_minutes":        types.Int64Type,
	"notification_time24x7": types.BoolType,
	"enabled":               types.BoolType,
	"sequence":              types.Int64Type,
	"email_address":         types.StringType,
	"country_code":          types.StringType,
	"phone_number":          types.StringType,
	"phone_extension":       types.StringType,
}

func (r *UserResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (r *UserResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an AlertOps user. Any additional fields supported by the AlertOps API can be passed via `extra_attributes` and will be merged into the request payload. If the API rejects a value, the raw API error is returned.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique identifier of the user.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"user_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The username of the user.",
			},
			"first_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The first name of the user.",
			},
			"last_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The last name of the user.",
			},
			"locale": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The locale of the user (e.g., `en-US`).",
			},
			"time_zone": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The time zone of the user (e.g., `(UTC-06:00) Central Time (US & Canada)`).",
			},
			"type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The type of user (e.g., `Standard`, `Stakeholder`).",
			},
			"external_id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "An optional external identifier for the user.",
			},
			"roles": schema.ListAttribute{
				Required:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "List of roles assigned to the user (e.g., `[\"App Admin\"]`).",
			},
			"contact_methods": schema.ListNestedAttribute{
				Required:            true,
				MarkdownDescription: "List of contact methods. At least one `Email-Official` contact method is required.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"contact_method_name": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The name of the contact method (e.g., `Email-Official`, `Phone-Work`, `SMS-Work`).",
						},
						"wait_time_in_mins": schema.Int64Attribute{
							Optional:            true,
							Computed:            true,
							Default:             int64default.StaticInt64(0),
							MarkdownDescription: "Wait time in minutes before this contact method is used.",
						},
						"repeat": schema.BoolAttribute{
							Optional:            true,
							Computed:            true,
							Default:             booldefault.StaticBool(false),
							MarkdownDescription: "Whether to repeat notifications.",
						},
						"repeat_times": schema.Int64Attribute{
							Optional:            true,
							Computed:            true,
							Default:             int64default.StaticInt64(0),
							MarkdownDescription: "Number of times to repeat.",
						},
						"repeat_minutes": schema.Int64Attribute{
							Optional:            true,
							Computed:            true,
							Default:             int64default.StaticInt64(0),
							MarkdownDescription: "Minutes between repeats.",
						},
						"notification_time24x7": schema.BoolAttribute{
							Optional:            true,
							Computed:            true,
							Default:             booldefault.StaticBool(true),
							MarkdownDescription: "Whether to notify 24x7.",
						},
						"enabled": schema.BoolAttribute{
							Optional:            true,
							Computed:            true,
							Default:             booldefault.StaticBool(true),
							MarkdownDescription: "Whether this contact method is enabled.",
						},
						"sequence": schema.Int64Attribute{
							Optional:            true,
							Computed:            true,
							Default:             int64default.StaticInt64(1),
							MarkdownDescription: "The sequence order of this contact method.",
						},
						"email_address": schema.StringAttribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "Email address (required for email contact methods).",
						},
						"country_code": schema.StringAttribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "Country code for phone/SMS contact methods.",
						},
						"phone_number": schema.StringAttribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "Phone number for phone/SMS contact methods.",
						},
						"phone_extension": schema.StringAttribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "Phone extension for phone contact methods.",
						},
					},
				},
			},
			"extra_attributes": schema.MapAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Additional fields to pass directly to the AlertOps API. Values are strings; the provider merges them into the request payload. If the API rejects a field or value, the raw API error is returned.",
			},
		},
	}
}

func (r *UserResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// buildContactMethodsPayload converts Terraform list to API payload slice.
func buildContactMethodsPayload(ctx context.Context, list types.List) ([]map[string]interface{}, error) {
	var result []map[string]interface{}
	elements := list.Elements()
	for _, elem := range elements {
		obj, ok := elem.(types.Object)
		if !ok {
			return nil, fmt.Errorf("unexpected element type in contact_methods")
		}
		attrs := obj.Attributes()

		cm := map[string]interface{}{
			"contact_method_name":   attrs["contact_method_name"].(types.String).ValueString(),
			"wait_time_in_mins":     attrs["wait_time_in_mins"].(types.Int64).ValueInt64(),
			"repeat":                attrs["repeat"].(types.Bool).ValueBool(),
			"repeat_times":          attrs["repeat_times"].(types.Int64).ValueInt64(),
			"repeat_minutes":        attrs["repeat_minutes"].(types.Int64).ValueInt64(),
			"notification_time24x7": attrs["notification_time24x7"].(types.Bool).ValueBool(),
			"enabled":               attrs["enabled"].(types.Bool).ValueBool(),
			"sequence":              attrs["sequence"].(types.Int64).ValueInt64(),
		}

		emailAddr := attrs["email_address"].(types.String).ValueString()
		if emailAddr != "" {
			cm["email"] = map[string]interface{}{"email_address": emailAddr}
		}

		phoneNumber := attrs["phone_number"].(types.String).ValueString()
		countryCode := attrs["country_code"].(types.String).ValueString()
		phoneExt := attrs["phone_extension"].(types.String).ValueString()

		if phoneNumber != "" {
			name := cm["contact_method_name"].(string)
			if len(name) >= 3 && name[:3] == "SMS" {
				cm["sms"] = map[string]interface{}{
					"country_code": countryCode,
					"phone_number": phoneNumber,
				}
			} else {
				cm["phone"] = map[string]interface{}{
					"country_code": countryCode,
					"phone_number": phoneNumber,
					"extension":    phoneExt,
				}
			}
		}

		result = append(result, cm)
	}
	return result, nil
}

// parseContactMethodsFromAPI converts API response contact methods to Terraform list.
func parseContactMethodsFromAPI(raw []interface{}) (types.List, error) {
	var objs []attr.Value
	for _, item := range raw {
		cm, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		emailAddr := ""
		if emailObj, ok := cm["email"].(map[string]interface{}); ok && emailObj != nil {
			if ea, ok := emailObj["email_address"].(string); ok {
				emailAddr = ea
			}
		}
		countryCode, phoneNumber, phoneExt := "", "", ""
		if phoneObj, ok := cm["phone"].(map[string]interface{}); ok && phoneObj != nil {
			if v, ok := phoneObj["country_code"].(string); ok {
				countryCode = v
			}
			if v, ok := phoneObj["phone_number"].(string); ok {
				phoneNumber = v
			}
			if v, ok := phoneObj["extension"].(string); ok {
				phoneExt = v
			}
		}
		if smsObj, ok := cm["sms"].(map[string]interface{}); ok && smsObj != nil {
			if v, ok := smsObj["country_code"].(string); ok {
				countryCode = v
			}
			if v, ok := smsObj["phone_number"].(string); ok {
				phoneNumber = v
			}
		}

		getInt64 := func(key string) int64 {
			if v, ok := cm[key].(float64); ok {
				return int64(v)
			}
			return 0
		}
		getBool := func(key string) bool {
			if v, ok := cm[key].(bool); ok {
				return v
			}
			return false
		}
		getString := func(key string) string {
			if v, ok := cm[key].(string); ok {
				return v
			}
			return ""
		}

		obj, diags := types.ObjectValue(contactMethodAttrTypes, map[string]attr.Value{
			"contact_method_name":   types.StringValue(getString("contact_method_name")),
			"wait_time_in_mins":     types.Int64Value(getInt64("wait_time_in_mins")),
			"repeat":                types.BoolValue(getBool("repeat")),
			"repeat_times":          types.Int64Value(getInt64("repeat_times")),
			"repeat_minutes":        types.Int64Value(getInt64("repeat_minutes")),
			"notification_time24x7": types.BoolValue(getBool("notification_time24x7")),
			"enabled":               types.BoolValue(getBool("enabled")),
			"sequence":              types.Int64Value(getInt64("sequence")),
			"email_address":         types.StringValue(emailAddr),
			"country_code":          types.StringValue(countryCode),
			"phone_number":          types.StringValue(phoneNumber),
			"phone_extension":       types.StringValue(phoneExt),
		})
		if diags.HasError() {
			return types.ListNull(types.ObjectType{AttrTypes: contactMethodAttrTypes}), fmt.Errorf("error building contact method object")
		}
		objs = append(objs, obj)
	}

	if len(objs) == 0 {
		return types.ListValueMust(types.ObjectType{AttrTypes: contactMethodAttrTypes}, []attr.Value{}), nil
	}
	list, diags := types.ListValue(types.ObjectType{AttrTypes: contactMethodAttrTypes}, objs)
	if diags.HasError() {
		return types.ListNull(types.ObjectType{AttrTypes: contactMethodAttrTypes}), fmt.Errorf("error building contact methods list")
	}
	return list, nil
}

// mergeExtraAttributes merges extra_attributes (map[string]string) into the payload map.
func mergeExtraAttributes(ctx context.Context, extra types.Map, payload map[string]interface{}) {
	if extra.IsNull() || extra.IsUnknown() {
		return
	}
	for k, v := range extra.Elements() {
		if sv, ok := v.(types.String); ok {
			strVal := sv.ValueString()
			// If the value looks like a JSON object or array, parse it
			if len(strVal) > 0 && (strVal[0] == '{' || strVal[0] == '[') {
				var parsed interface{}
				if err := json.Unmarshal([]byte(strVal), &parsed); err == nil {
					payload[k] = parsed
					continue
				}
			}
			payload[k] = strVal
		}
	}
}

type userModel struct {
	ID              types.String `tfsdk:"id"`
	UserName        types.String `tfsdk:"user_name"`
	FirstName       types.String `tfsdk:"first_name"`
	LastName        types.String `tfsdk:"last_name"`
	Locale          types.String `tfsdk:"locale"`
	TimeZone        types.String `tfsdk:"time_zone"`
	Type            types.String `tfsdk:"type"`
	ExternalID      types.String `tfsdk:"external_id"`
	Roles           types.List   `tfsdk:"roles"`
	ContactMethods  types.List   `tfsdk:"contact_methods"`
	ExtraAttributes types.Map    `tfsdk:"extra_attributes"`
}

func (r *UserResource) buildPayload(ctx context.Context, plan *userModel) (map[string]interface{}, error) {
	var roles []string
	if diags := plan.Roles.ElementsAs(ctx, &roles, false); diags.HasError() {
		return nil, fmt.Errorf("error reading roles")
	}

	cms, err := buildContactMethodsPayload(ctx, plan.ContactMethods)
	if err != nil {
		return nil, err
	}

	payload := map[string]interface{}{
		"user_name":       plan.UserName.ValueString(),
		"first_name":      plan.FirstName.ValueString(),
		"last_name":       plan.LastName.ValueString(),
		"locale":          plan.Locale.ValueString(),
		"time_zone":       plan.TimeZone.ValueString(),
		"type":            plan.Type.ValueString(),
		"roles":           roles,
		"contact_methods": cms,
	}
	if plan.ExternalID.ValueString() != "" {
		payload["external_id"] = plan.ExternalID.ValueString()
	}
	mergeExtraAttributes(ctx, plan.ExtraAttributes, payload)
	return payload, nil
}

func (r *UserResource) applyAPIResponse(ctx context.Context, result map[string]interface{}, plan *userModel) error {
	if extID, ok := result["external_id"].(string); ok {
		plan.ExternalID = types.StringValue(extID)
	} else {
		plan.ExternalID = types.StringValue("")
	}

	if rawRoles, ok := result["roles"].([]interface{}); ok {
		var roles []string
		for _, rv := range rawRoles {
			if s, ok := rv.(string); ok {
				roles = append(roles, s)
			}
		}
		rolesList, diags := types.ListValueFrom(ctx, types.StringType, roles)
		if diags.HasError() {
			return fmt.Errorf("error building roles list")
		}
		plan.Roles = rolesList
	}

	if rawCMs, ok := result["contact_methods"].([]interface{}); ok {
		cmList, err := parseContactMethodsFromAPI(rawCMs)
		if err != nil {
			return err
		}
		plan.ContactMethods = cmList
	}
	return nil
}

func (r *UserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan userModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload, err := r.buildPayload(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Error building user payload", err.Error())
		return
	}

	respBody, err := r.client.Post(ctx, "/users", payload)
	if err != nil {
		resp.Diagnostics.AddError("Error creating user", err.Error())
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing user response", err.Error())
		return
	}

	userID, ok := result["user_id"].(float64)
	if !ok {
		resp.Diagnostics.AddError("Error creating user", fmt.Sprintf("Could not get user_id from response: %s", string(respBody)))
		return
	}
	plan.ID = types.StringValue(fmt.Sprintf("%d", int(userID)))
	// Save the ID immediately so a failure below does not orphan the user.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), plan.ID)...)

	// The API ignores time_zone and locale on create (the account defaults are applied),
	// but honours them on update, so apply them with a follow-up PUT when they differ.
	wantTimeZone, wantLocale := plan.TimeZone.ValueString(), plan.Locale.ValueString()
	if gotTimeZone, gotLocale := mapString(result, "time_zone"), mapString(result, "locale"); gotTimeZone != wantTimeZone || gotLocale != wantLocale {
		if _, err := r.client.Put(ctx, "/users/"+plan.ID.ValueString(), payload); err != nil {
			resp.Diagnostics.AddError("Error setting user time zone and locale after create", err.Error())
			return
		}
	}

	if err := r.refresh(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading user after create", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// refresh reads the user from the API into the model. Write responses do not include every
// field (e.g. contact methods), so state is always taken from a GET.
func (r *UserResource) refresh(ctx context.Context, model *userModel) error {
	respBody, err := r.client.Get(ctx, "/users/"+model.ID.ValueString())
	if err != nil {
		return err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("error parsing user response: %w", err)
	}

	model.UserName = types.StringValue(mapString(result, "user_name"))
	model.FirstName = types.StringValue(mapString(result, "first_name"))
	model.LastName = types.StringValue(mapString(result, "last_name"))
	model.Locale = types.StringValue(mapString(result, "locale"))
	model.TimeZone = types.StringValue(mapString(result, "time_zone"))
	model.Type = types.StringValue(mapString(result, "type"))

	return r.applyAPIResponse(ctx, result, model)
}

func (r *UserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state userModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.refresh(ctx, &state); err != nil {
		if IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading user", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *UserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan userModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload, err := r.buildPayload(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Error building user payload", err.Error())
		return
	}

	if _, err := r.client.Put(ctx, "/users/"+plan.ID.ValueString(), payload); err != nil {
		resp.Diagnostics.AddError("Error updating user", err.Error())
		return
	}

	if err := r.refresh(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading user after update", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *UserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state userModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.Delete(ctx, "/users/"+state.ID.ValueString())
	if err != nil {
		if IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting user", err.Error())
		return
	}
}

func (r *UserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
