package pgrmongodb

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &appServicesAppResource{}
	_ resource.ResourceWithConfigure   = &appServicesAppResource{}
	_ resource.ResourceWithImportState = &appServicesAppResource{}
)

func NewAppServicesAppResource() resource.Resource {
	return &appServicesAppResource{}
}

type appServicesAppResource struct {
	bearer_token string
}

type appServicesAppResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	ProjectID          types.String `tfsdk:"project_id"`
	ClusterName        types.String `tfsdk:"cluster_name"`
	AppServicesAppName types.String `tfsdk:"appservices_app_name"`
	LinkedDatasourceID types.String `tfsdk:"linked_datasource_id"`
}

func (r *appServicesAppResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_appservicesapp"
}

func (r *appServicesAppResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a MongoDB Atlas App Services App",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "identifier for resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"project_id": schema.StringAttribute{
				Description: "MongoDB Atlas project identifier. Sometime referred to as group id.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(24),
					stringvalidator.LengthAtMost(24),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^([a-f0-9]{24})$`),
						"must be a valid 12 byte hexadecimal project_id",
					),
				},
			},
			"cluster_name": schema.StringAttribute{
				Description: "Name of the MongoDB Atlas cluster deployed to project.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtMost(64),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]*$`),
						"must be a valid cluster name",
					),
				},
			},
			"appservices_app_name": schema.StringAttribute{
				Description: "Name of MongoDB Atlas App Services app name to be managed.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("TerraformApp"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`),
						"must be a valid app services app name",
					),
				},
			},
			"linked_datasource_id": schema.StringAttribute{
				Description: "Identifier for linked datasource associated to this App Services app.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *appServicesAppResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.bearer_token = req.ProviderData.(providerData).bearer_token
}

func (r *appServicesAppResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan appServicesAppResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := plan.ProjectID.ValueString()
	clusterName := plan.ClusterName.ValueString()
	appName := plan.AppServicesAppName.ValueString()

	tflog.Info(ctx, "creating mongodb atlas app services app")
	response, err := createAppServicesApp(r.bearer_token, projectID, clusterName, appName)
	tflog.Debug(ctx, fmt.Sprintf("%v", response))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating App Services App",
			"Could not create MongoDB Atlas App Services App. Received error: "+err.Error(),
		)
		return
	}
	appservices_app_id := response["_id"].(string)
	linked_datasource_id, err := getAppServicesLinkedDatasourceByAppID(r.bearer_token, projectID, appservices_app_id, clusterName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating App Services App",
			"Could not create MongoDB Atlas App Services App. Received error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(appservices_app_id)
	plan.LinkedDatasourceID = types.StringValue(linked_datasource_id)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *appServicesAppResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state appServicesAppResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := state.ProjectID.ValueString()
	clusterName := state.ClusterName.ValueString()
	appName := state.AppServicesAppName.ValueString()

	tflog.Info(ctx, "reading mongodb atlas app services app")
	appservices_app_id, linked_datasource_id, err := getAppServicesAppByName(r.bearer_token, projectID, appName, clusterName, false)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading App Services App",
			"Could not read MongoDB Atlas App Services App. Received error: "+err.Error()+","+req.State.Raw.String(),
		)
		return
	}

	state.ID = types.StringValue(appservices_app_id)
	state.LinkedDatasourceID = types.StringValue(linked_datasource_id)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *appServicesAppResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state appServicesAppResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var plan appServicesAppResourceModel
	diags = req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasChange := false
	if state.ClusterName != plan.ClusterName || state.ProjectID != plan.ProjectID || state.AppServicesAppName != plan.AppServicesAppName {
		hasChange = true
	}

	tflog.Info(ctx, "checking app services app for deltas")
	if hasChange {
		projectID := plan.ProjectID.ValueString()
		clusterName := plan.ClusterName.ValueString()
		appName := plan.AppServicesAppName.ValueString()
		tflog.Info(ctx, "creating mongodb atlas app services app for update plan")
		response, err := createAppServicesApp(r.bearer_token, projectID, clusterName, appName)
		tflog.Debug(ctx, fmt.Sprintf("%v", response))
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Creating App Services App",
				"Could not create MongoDB Atlas App Services App. Received error: "+err.Error(),
			)
			return
		}
		appservices_app_id := response["_id"].(string)
		linked_datasource_id, err := getAppServicesLinkedDatasourceByAppID(r.bearer_token, projectID, appservices_app_id, clusterName)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Creating App Services App",
				"Could not create MongoDB Atlas App Services App. Received error: "+err.Error(),
			)
			return
		}

		tflog.Info(ctx, fmt.Sprintf("deleting old app services app %s for update", state.AppServicesAppName.ValueString()))
		err = deleteAppServicesApp(r.bearer_token, state.ProjectID.ValueString(), state.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Deleting App Services App",
				"Could not delete MongoDB Atlas App Services App. Received error: "+err.Error(),
			)
			return
		}

		state.ID = types.StringValue(appservices_app_id)
		state.AppServicesAppName = plan.AppServicesAppName
		state.LinkedDatasourceID = types.StringValue(linked_datasource_id)
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *appServicesAppResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state appServicesAppResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := state.ProjectID.ValueString()
	appID := state.ID.ValueString()

	tflog.Info(ctx, fmt.Sprintf("deleting app services app %s", state.AppServicesAppName.ValueString()))
	err := deleteAppServicesApp(r.bearer_token, projectID, appID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting App Services App",
			"Could not delete MongoDB Atlas App Services App. Received error: "+err.Error(),
		)
		return
	}
}

func (r *appServicesAppResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, ",")

	if len(idParts) != 4 {
		resp.Diagnostics.AddError(
			"Error Importing App Services App",
			"Could not import MongoDB Atlas App Services App.\nPlease ensure you run terraform import with project_id,atlasClusterName,appServicesAppName,linkedDatasourceID",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_name"), idParts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("appservices_app_name"), idParts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("linked_datasource_id"), idParts[3])...)
}
