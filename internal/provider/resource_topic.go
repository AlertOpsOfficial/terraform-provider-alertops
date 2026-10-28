package provider

import (
	"context"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &TopicResource{}
	_ resource.ResourceWithConfigure   = &TopicResource{}
	_ resource.ResourceWithImportState = &TopicResource{}
)

func NewTopicResource() resource.Resource {
	return &TopicResource{}
}

type TopicResource struct {
	client *Client
}

type topicModel struct {
	ID        types.String `tfsdk:"id"`
	TopicName types.String `tfsdk:"topic_name"`
	Priority  types.String `tfsdk:"priority"`
	Template  types.String `tfsdk:"template"`
	Groups    types.Set    `tfsdk:"groups"`
}

func (r *TopicResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_topic"
}

func (r *TopicResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an AlertOps topic. Topics are used to route alerts to the groups subscribed to them.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique identifier of the topic.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"topic_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the topic.",
			},
			"priority": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The priority of the topic (e.g., `No Priority`, `Low`, `Medium`, `High`, `Critical`).",
			},
			"template": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				MarkdownDescription: "The topic template.",
			},
			"groups": schema.SetAttribute{
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Names of the groups subscribed to this topic. When set, the provider adds and removes groups so that exactly these are subscribed. When omitted, subscriptions are left unmanaged. Do not also manage the same subscriptions through `alertops_group.topics`.",
			},
		},
	}
}

func (r *TopicResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func buildTopicPayload(plan *topicModel) map[string]interface{} {
	return map[string]interface{}{
		"topic_name": plan.TopicName.ValueString(),
		"priority":   plan.Priority.ValueString(),
		"template":   plan.Template.ValueString(),
	}
}

func parseTopicFromAPI(result map[string]interface{}, state *topicModel) {
	state.TopicName = types.StringValue(mapString(result, "topic_name"))
	state.Priority = types.StringValue(mapString(result, "priority"))
	state.Template = types.StringValue(mapString(result, "template"))
}

// readTopicGroups returns the subscribed groups as name -> group_id.
func (r *TopicResource) readTopicGroups(ctx context.Context, id string) (map[string]string, error) {
	body, err := r.client.Get(ctx, "/topics/"+id+"/groups")
	if err != nil {
		return nil, err
	}
	raw, err := decodeArray(body)
	if err != nil {
		return nil, err
	}
	groups := map[string]string{}
	for _, item := range raw {
		if m, ok := item.(map[string]interface{}); ok {
			groupID, _ := idFromResult(m, "group_id")
			groups[mapString(m, "group_name")] = groupID
		}
	}
	return groups, nil
}

func groupNamesSet(groups map[string]string) types.Set {
	names := make([]string, 0, len(groups))
	for name := range groups {
		names = append(names, name)
	}
	sort.Strings(names)
	return types.SetValueMust(types.StringType, stringValues(names))
}

// syncTopicGroups makes the topic's subscribed groups match the desired set when it is configured.
func (r *TopicResource) syncTopicGroups(ctx context.Context, id string, desired types.Set) error {
	if !isKnown(desired) {
		return nil
	}
	current, err := r.readTopicGroups(ctx, id)
	if err != nil {
		return fmt.Errorf("error reading topic groups: %w", err)
	}
	want := map[string]bool{}
	var add []string
	for _, name := range stringsFromElements(desired.Elements()) {
		want[name] = true
		if _, ok := current[name]; !ok {
			add = append(add, name)
		}
	}
	for name, groupID := range current {
		if !want[name] {
			if _, err := r.client.Delete(ctx, "/topics/"+id+"/groups/"+groupID); err != nil && !IsNotFound(err) {
				return fmt.Errorf("error removing group %q from topic: %w", name, err)
			}
		}
	}
	if len(add) > 0 {
		if _, err := r.client.Post(ctx, "/topics/"+id+"/groups", map[string]interface{}{"groups": add}); err != nil {
			return fmt.Errorf("error adding groups to topic: %w", err)
		}
	}
	return nil
}

// refresh reads the topic and its groups into state.
func (r *TopicResource) refresh(ctx context.Context, state *topicModel) error {
	body, err := r.client.Get(ctx, "/topics/"+state.ID.ValueString())
	if err != nil {
		return err
	}
	result, err := decodeObject(body)
	if err != nil {
		return err
	}
	parseTopicFromAPI(result, state)
	groups, err := r.readTopicGroups(ctx, state.ID.ValueString())
	if err != nil {
		return err
	}
	state.Groups = groupNamesSet(groups)
	return nil
}

func (r *TopicResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan topicModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	respBody, err := r.client.Post(ctx, "/topics", buildTopicPayload(&plan))
	if err != nil {
		resp.Diagnostics.AddError("Error creating topic", err.Error())
		return
	}
	result, err := decodeObject(respBody)
	if err != nil {
		resp.Diagnostics.AddError("Error parsing topic response", err.Error())
		return
	}
	id, ok := idFromResult(result, "topic_id")
	if !ok {
		resp.Diagnostics.AddError("Error creating topic", fmt.Sprintf("Could not get topic_id from response: %s", string(respBody)))
		return
	}
	plan.ID = types.StringValue(id)
	// Save the ID immediately so a failure below does not orphan the topic.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), plan.ID)...)

	if err := r.syncTopicGroups(ctx, id, plan.Groups); err != nil {
		resp.Diagnostics.AddError("Error setting topic groups", err.Error())
		return
	}
	if err := r.refresh(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading topic after create", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TopicResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state topicModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.refresh(ctx, &state); err != nil {
		if IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading topic", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *TopicResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state topicModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = state.ID
	id := state.ID.ValueString()

	if _, err := r.client.Put(ctx, "/topics/"+id, buildTopicPayload(&plan)); err != nil {
		resp.Diagnostics.AddError("Error updating topic", err.Error())
		return
	}
	if err := r.syncTopicGroups(ctx, id, plan.Groups); err != nil {
		resp.Diagnostics.AddError("Error updating topic groups", err.Error())
		return
	}
	if err := r.refresh(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading topic after update", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TopicResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state topicModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := r.client.Delete(ctx, "/topics/"+state.ID.ValueString()); err != nil && !IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting topic", err.Error())
	}
}

func (r *TopicResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
