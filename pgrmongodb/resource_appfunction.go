package pgrmongodb

import (
	"context"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &appFunctionResource{}
	_ resource.ResourceWithConfigure   = &appFunctionResource{}
	_ resource.ResourceWithImportState = &appFunctionResource{}
)

func NewAppFunctionResource() resource.Resource {
	return &appFunctionResource{}
}

type appFunctionResource struct {
	bearer_token string
}

type appFunctionResourceModel struct {
	ID               types.String `tfsdk:"id"`
	ProjectID        types.String `tfsdk:"project_id"`
	AppServicesAppID types.String `tfsdk:"appservices_app_id"`
	FunctionName     types.String `tfsdk:"function_name"`
	FunctionCode     types.String `tfsdk:"function_code"`
}

func (r *appFunctionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_appfunction"
}

func (r *appFunctionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a MongoDB Atlas App Services Function",
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
			"appservices_app_id": schema.StringAttribute{
				Description: "MongoDB Atlas App Services app id to manage functions.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(24),
					stringvalidator.LengthAtMost(24),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^([a-f0-9]{24})$`),
						"must be a valid 12 byte hexadecimal appservices_app_id",
					),
				},
			},
			"function_name": schema.StringAttribute{
				Description: "Name of function to deploy to App Services.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"function_code": schema.StringAttribute{
				Description: "Code to be deployed to App Services function.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *appFunctionResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.bearer_token = req.ProviderData.(providerData).bearer_token
}

func (r *appFunctionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan appFunctionResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := plan.ProjectID.ValueString()
	appServicesAppID := plan.AppServicesAppID.ValueString()
	functionName := plan.FunctionName.ValueString()
	functionCode := plan.FunctionCode.ValueString()

	tflog.Info(ctx, "creating mongodb atlas app services function")
	function_id, err := createAppServicesFunction(r.bearer_token, projectID, appServicesAppID, functionName, functionCode)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating App Services Function",
			"Could not create MongoDB Atlas App Services Function. Received error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(function_id)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *appFunctionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state appFunctionResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := state.ProjectID.ValueString()
	appServicesAppID := state.AppServicesAppID.ValueString()
	functionName := state.FunctionName.ValueString()

	tflog.Info(ctx, "reading mongodb atlas app services function")
	function_id, function_code, err := getAppServicesFunctionByName(r.bearer_token, projectID, appServicesAppID, functionName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading App Services Function",
			"Could not read MongoDB Atlas App Services Function. Received error: "+err.Error()+","+req.State.Raw.String(),
		)
		return
	}

	state.ID = types.StringValue(function_id)
	state.FunctionCode = types.StringValue(function_code)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *appFunctionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state appFunctionResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var plan appFunctionResourceModel
	diags = req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasChange := false
	if state.FunctionName != plan.FunctionName || state.ProjectID != plan.ProjectID || state.FunctionCode != plan.FunctionCode {
		hasChange = true
	}

	tflog.Info(ctx, "checking mongodb atlas app services function for deltas")
	if hasChange {
		tflog.Info(ctx, "updating mongodb atlas app services function")
		function_id, err := createAppServicesFunction(r.bearer_token, plan.ProjectID.ValueString(), plan.AppServicesAppID.ValueString(), plan.FunctionName.ValueString(), plan.FunctionCode.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Creating App Services Function",
				"Could not create MongoDB Atlas App Services Function. Received error: "+err.Error(),
			)
			return
		}
		err = deleteAppServicesFunction(r.bearer_token, state.ProjectID.ValueString(), state.AppServicesAppID.ValueString(), state.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Deleting App Services Function",
				"Could not delete MongoDB Atlas App Services Function. Received error: "+err.Error(),
			)
			return
		}

		state.ID = types.StringValue(function_id)
		state.ProjectID = plan.ProjectID
		state.FunctionName = plan.FunctionName
		state.FunctionCode = plan.FunctionCode
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *appFunctionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state appFunctionResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := state.ProjectID.ValueString()
	appServicesAppID := state.AppServicesAppID.ValueString()
	functionID := state.ID.ValueString()

	tflog.Info(ctx, "deleting mongodb atlas app services function")
	err := deleteAppServicesFunction(r.bearer_token, projectID, appServicesAppID, functionID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting App Services Function",
			"Could not delete MongoDB Atlas App Services Function. Received error: "+err.Error(),
		)
		return
	}
}

func (r *appFunctionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, ",")

	if len(idParts) != 3 {
		resp.Diagnostics.AddError(
			"Error Importing App Services Function",
			"Could not import MongoDB Atlas App Services Function.\nPlease ensure you run terraform import with project_id,appservices_app_id,function_id",
		)
		return
	}

	found_function_name, found_function_code, err := getAppServicesFunctionByID(r.bearer_token, idParts[0], idParts[1], idParts[2])
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Retrieving App Services Function Code",
			"Could not retrieve MongoDB Atlas App Services Function code. Received error: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), idParts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("appservices_app_id"), idParts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("function_name"), found_function_name)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("function_code"), found_function_code)...)
}
