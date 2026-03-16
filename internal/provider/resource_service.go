package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &ServiceResource{}
	_ resource.ResourceWithConfigure   = &ServiceResource{}
	_ resource.ResourceWithImportState = &ServiceResource{}
)

func NewServiceResource() resource.Resource {
	return &ServiceResource{}
}

type ServiceResource struct {
	client *Client
}

var serviceComponentAttrTypes = map[string]attr.Type{
	"component_name": types.StringType,
	"external_id":    types.StringType,
}

var serviceTemplateAttrTypes = map[string]attr.Type{
	"template_name": types.StringType,
	"message":       types.StringType,
	"subject":       types.StringType,
}

type serviceModel struct {
	ID              types.String `tfsdk:"id"`
	ServiceName     types.String `tfsdk:"service_name"`
	ShowUptime      types.Bool   `tfsdk:"show_uptime"`
	Components      types.List   `tfsdk:"components"`
	Templates       types.List   `tfsdk:"templates"`
	ExtraAttributes types.Map    `tfsdk:"extra_attributes"`
}

func (r *ServiceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

func (r *ServiceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an AlertOps service. Any additional fields supported by the AlertOps API can be passed via `extra_attributes`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique identifier of the service.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"service_name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The name of the service.",
			},
			"show_uptime": schema.BoolAttribute{
				Required:            true,
				MarkdownDescription: "Whether to show uptime for the service.",
			},
			"components": schema.ListNestedAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "List of components associated with the service.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"component_name": schema.StringAttribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "The name of the component.",
						},
						"external_id": schema.StringAttribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "An external identifier for the component.",
						},
					},
				},
			},
			"templates": schema.ListNestedAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "List of message templates for the service.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"template_name": schema.StringAttribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "The name of the template.",
						},
						"message": schema.StringAttribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "The message body of the template.",
						},
						"subject": schema.StringAttribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "The subject line of the template.",
						},
					},
				},
			},
			"extra_attributes": schema.MapAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Additional fields to pass directly to the AlertOps API. If the API rejects a field or value, the raw API error is returned.",
			},
		},
	}
}

func (r *ServiceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func buildServicePayload(ctx context.Context, plan *serviceModel) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"show_uptime": plan.ShowUptime.ValueBool(),
	}
	if v := plan.ServiceName.ValueString(); v != "" {
		payload["service_name"] = v
	}

	if !plan.Components.IsNull() && !plan.Components.IsUnknown() {
		var comps []map[string]interface{}
		for _, elem := range plan.Components.Elements() {
			obj := elem.(types.Object)
			attrs := obj.Attributes()
			comp := map[string]interface{}{}
			if v := attrs["component_name"].(types.String).ValueString(); v != "" {
				comp["component_name"] = v
			}
			if v := attrs["external_id"].(types.String).ValueString(); v != "" {
				comp["external_id"] = v
			}
			comps = append(comps, comp)
		}
		payload["components"] = comps
	}

	if !plan.Templates.IsNull() && !plan.Templates.IsUnknown() {
		var tmpls []map[string]interface{}
		for _, elem := range plan.Templates.Elements() {
			obj := elem.(types.Object)
			attrs := obj.Attributes()
			tmpl := map[string]interface{}{}
			if v := attrs["template_name"].(types.String).ValueString(); v != "" {
				tmpl["template_name"] = v
			}
			if v := attrs["message"].(types.String).ValueString(); v != "" {
				tmpl["message"] = v
			}
			if v := attrs["subject"].(types.String).ValueString(); v != "" {
				tmpl["subject"] = v
			}
			tmpls = append(tmpls, tmpl)
		}
		payload["templates"] = tmpls
	}

	mergeExtraAttributes(ctx, plan.ExtraAttributes, payload)
	return payload, nil
}

func parseServiceFromAPI(ctx context.Context, result map[string]interface{}, state *serviceModel) error {
	if v, ok := result["service_name"].(string); ok {
		state.ServiceName = types.StringValue(v)
	} else {
		state.ServiceName = types.StringValue("")
	}
	if v, ok := result["show_uptime"].(bool); ok {
		state.ShowUptime = types.BoolValue(v)
	}

	// Components
	if rawComps, ok := result["components"].([]interface{}); ok {
		var compObjs []attr.Value
		for _, item := range rawComps {
			c, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			getString := func(key string) string {
				if v, ok := c[key].(string); ok {
					return v
				}
				return ""
			}
			obj, diags := types.ObjectValue(serviceComponentAttrTypes, map[string]attr.Value{
				"component_name": types.StringValue(getString("component_name")),
				"external_id":    types.StringValue(getString("external_id")),
			})
			if diags.HasError() {
				return fmt.Errorf("error building component object")
			}
			compObjs = append(compObjs, obj)
		}
		list, diags := types.ListValue(types.ObjectType{AttrTypes: serviceComponentAttrTypes}, compObjs)
		if diags.HasError() {
			return fmt.Errorf("error building components list")
		}
		state.Components = list
	} else {
		state.Components = types.ListValueMust(types.ObjectType{AttrTypes: serviceComponentAttrTypes}, []attr.Value{})
	}

	// Templates
	if rawTmpls, ok := result["templates"].([]interface{}); ok {
		var tmplObjs []attr.Value
		for _, item := range rawTmpls {
			t, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			getString := func(key string) string {
				if v, ok := t[key].(string); ok {
					return v
				}
				return ""
			}
			obj, diags := types.ObjectValue(serviceTemplateAttrTypes, map[string]attr.Value{
				"template_name": types.StringValue(getString("template_name")),
				"message":       types.StringValue(getString("message")),
				"subject":       types.StringValue(getString("subject")),
			})
			if diags.HasError() {
				return fmt.Errorf("error building template object")
			}
			tmplObjs = append(tmplObjs, obj)
		}
		list, diags := types.ListValue(types.ObjectType{AttrTypes: serviceTemplateAttrTypes}, tmplObjs)
		if diags.HasError() {
			return fmt.Errorf("error building templates list")
		}
		state.Templates = list
	} else {
		state.Templates = types.ListValueMust(types.ObjectType{AttrTypes: serviceTemplateAttrTypes}, []attr.Value{})
	}

	return nil
}

func (r *ServiceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan serviceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload, err := buildServicePayload(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Error building service payload", err.Error())
		return
	}

	respBody, err := r.client.Post(ctx, "/services", payload)
	if err != nil {
		resp.Diagnostics.AddError("Error creating service", err.Error())
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing service response", err.Error())
		return
	}

	if id, ok := result["service_id"].(float64); ok {
		plan.ID = types.StringValue(fmt.Sprintf("%d", int(id)))
	} else if idStr, ok := result["service_id"].(string); ok {
		plan.ID = types.StringValue(idStr)
	} else {
		resp.Diagnostics.AddError("Error creating service",
			fmt.Sprintf("Could not get service_id from response: %s", string(respBody)))
		return
	}

	if err := parseServiceFromAPI(ctx, result, &plan); err != nil {
		resp.Diagnostics.AddError("Error processing service response", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ServiceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state serviceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	respBody, err := r.client.Get(ctx, "/services/"+state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading service", err.Error())
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing service response", err.Error())
		return
	}

	if err := parseServiceFromAPI(ctx, result, &state); err != nil {
		resp.Diagnostics.AddError("Error processing service response", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ServiceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan serviceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state serviceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload, err := buildServicePayload(ctx, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Error building service payload", err.Error())
		return
	}

	// PUT returns 204 No Content, so we must GET after update
	_, err = r.client.Put(ctx, "/services/"+state.ID.ValueString(), payload)
	if err != nil {
		resp.Diagnostics.AddError("Error updating service", err.Error())
		return
	}

	// Refresh state via GET
	respBody, err := r.client.Get(ctx, "/services/"+state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading service after update", err.Error())
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		resp.Diagnostics.AddError("Error parsing service response", err.Error())
		return
	}

	plan.ID = state.ID
	if err := parseServiceFromAPI(ctx, result, &plan); err != nil {
		resp.Diagnostics.AddError("Error processing service response", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ServiceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state serviceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.Delete(ctx, "/services/"+state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting service", err.Error())
		return
	}
}

func (r *ServiceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
