package pgrmongodb

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &appFunctionDependenciesResource{}
	_ resource.ResourceWithConfigure   = &appFunctionDependenciesResource{}
	_ resource.ResourceWithImportState = &appFunctionDependenciesResource{}
)

func NewAppFunctionDependencies() resource.Resource {
	return &appFunctionDependenciesResource{}
}

type appFunctionDependenciesResource struct {
	bearer_token string
}

type appFunctionDependenciesResourceModel struct {
	ID               types.String `tfsdk:"id"`
	ProjectID        types.String `tfsdk:"project_id"`
	AppServicesAppID types.String `tfsdk:"appservices_app_id"`
	Dependencies     types.Set    `tfsdk:"dependencies"`
}

func (r *appFunctionDependenciesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_appfunctiondependencies"
}

func (r *appFunctionDependenciesResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a MongoDB Atlas App Services Function Dependencies",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "identifier for resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
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
			"dependencies": schema.SetAttribute{
				Description: "List of function dependencies.",
				Required:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *appFunctionDependenciesResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.bearer_token = req.ProviderData.(providerData).bearer_token
}

func (r *appFunctionDependenciesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan appFunctionDependenciesResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := plan.ProjectID.ValueString()
	appServicesAppID := plan.AppServicesAppID.ValueString()

	dependencies := make([]types.String, 0, len(plan.Dependencies.Elements()))
	diags = plan.Dependencies.ElementsAs(ctx, &dependencies, false)
	tflog.Info(ctx, strconv.Itoa(len(dependencies)))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "creating mongodb atlas app services function dependencies")
	tflog.Debug(ctx, fmt.Sprintf("number of dependencies: %s", strconv.Itoa(len(dependencies))))
	tflog.Debug(ctx, fmt.Sprintf("dependencies: %v", dependencies))
	err := createAppFunctionDependencies(r.bearer_token, projectID, appServicesAppID, dependencies)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating App Services Function Dependencies",
			"Could not create MongoDB Atlas App Services Function Dependencies. Received error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(projectID)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *appFunctionDependenciesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state appFunctionDependenciesResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := state.ProjectID.ValueString()
	appServicesAppID := state.AppServicesAppID.ValueString()

	tflog.Info(ctx, "reading mongodb atlas app services function dependencies")
	dependencies, err := getAppFunctionDependencies(r.bearer_token, projectID, appServicesAppID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading App Services Function Depenendencies",
			"Could not read MongoDB Atlas App Services Function Depenendencies. Received error: "+err.Error()+","+req.State.Raw.String(),
		)
		return
	}

	state.ID = types.StringValue(projectID)
	state.Dependencies, diags = types.SetValueFrom(ctx, types.StringType, dependencies)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *appFunctionDependenciesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state appFunctionDependenciesResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var plan appFunctionDependenciesResourceModel
	diags = req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	sameDependencies := reflect.DeepEqual(state.Dependencies, plan.Dependencies)
	hasChange := false
	if state.ProjectID != plan.ProjectID || state.AppServicesAppID != plan.AppServicesAppID || !sameDependencies {
		hasChange = true
	}

	tflog.Info(ctx, "checking mongodb atlas app services function dependencies for deltas")
	if hasChange {
		tflog.Info(ctx, "updating mongodb atlas app services function dependencies")
		planElements := make([]types.String, 0, len(plan.Dependencies.Elements()))
		diags = plan.Dependencies.ElementsAs(ctx, &planElements, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		if !sameDependencies {
			tflog.Info(ctx, "detected delta in dependencies from state to current plan")
			// different dependencies means we have to capture the delta and create/delete accordingly
			stateElements := make([]types.String, 0, len(state.Dependencies.Elements()))
			diags = state.Dependencies.ElementsAs(ctx, &stateElements, false)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			var toBeAdded []basetypes.StringValue
			var toBeRemoved []basetypes.StringValue
			// find elements in plan that aren't in the state (to be added)
			for i := 0; i < len(planElements); i++ {
				found := false
				for j := 0; j < len(stateElements); j++ {
					if planElements[i] == stateElements[j] {
						found = true
						break
					}
				}
				if !found {
					toBeAdded = append(toBeAdded, planElements[i])
				}
			}
			for i := 0; i < len(toBeAdded); i++ {
				depTokens := strings.Split(toBeAdded[i].ValueString(), " ")
				err := manageAppFunctionDependency(r.bearer_token, plan.ProjectID.ValueString(), plan.AppServicesAppID.ValueString(), depTokens[0], depTokens[1], "PUT")
				if err != nil {
					resp.Diagnostics.AddError(
						"Error Creating App Services Function Depenendencies",
						"Could not create MongoDB Atlas App Services Function Depenendencies. Received error: "+err.Error()+","+req.State.Raw.String(),
					)
					return
				}
			}
			// find elements in state that aren't in the plan (to be removed)
			for i := 0; i < len(stateElements); i++ {
				found := false
				for j := 0; j < len(planElements); j++ {
					if stateElements[i] == planElements[j] {
						found = true
						break
					}
				}
				if !found {
					toBeRemoved = append(toBeRemoved, stateElements[i])
				}
			}
			for i := 0; i < len(toBeRemoved); i++ {
				depTokens := strings.Split(toBeRemoved[i].ValueString(), " ")
				err := manageAppFunctionDependency(r.bearer_token, state.ProjectID.ValueString(), state.AppServicesAppID.ValueString(), depTokens[0], depTokens[1], "DELETE")
				if err != nil {
					resp.Diagnostics.AddError(
						"Error Deleting App Services Function Depenendencies",
						"Could not delete MongoDB Atlas App Services Function Depenendencies. Received error: "+err.Error()+","+req.State.Raw.String(),
					)
					return
				}
			}
		} else {
			// different project or app services apps do a full create on the new project/app and full delete from the old project/app
			tflog.Info(ctx, "creating mongodb atlas app services function dependencies")
			tflog.Debug(ctx, fmt.Sprintf("number of plan dependencies to create: %s", strconv.Itoa(len(planElements))))
			tflog.Debug(ctx, fmt.Sprintf("dependencies: %v", planElements))
			err := createAppFunctionDependencies(r.bearer_token, plan.ProjectID.ValueString(), plan.AppServicesAppID.ValueString(), planElements)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error Creating App Services Function Depenendencies",
					"Could not create MongoDB Atlas App Services Function Depenendencies. Received error: "+err.Error(),
				)
				return
			}
			tflog.Info(ctx, "deleting mongodb atlas app services function dependencies")
			err = deleteAllAppFunctionDependencies(r.bearer_token, state.ProjectID.ValueString(), state.AppServicesAppID.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(
					"Error Deleting App Services Function",
					"Could not delete MongoDB Atlas App Services Function. Received error: "+err.Error(),
				)
				return
			}
		}

		state.ID = plan.ProjectID
		state.ProjectID = plan.ProjectID
		state.AppServicesAppID = plan.AppServicesAppID
		state.Dependencies = plan.Dependencies
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *appFunctionDependenciesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state appFunctionDependenciesResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := state.ProjectID.ValueString()
	appServicesAppID := state.AppServicesAppID.ValueString()

	tflog.Info(ctx, "deleting mongodb atlas app services function dependencies")
	err := deleteAllAppFunctionDependencies(r.bearer_token, projectID, appServicesAppID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting App Services Function Dependencies",
			"Could not delete MongoDB Atlas App Services Function Dependencies. Received error: "+err.Error(),
		)
		return
	}
}

func (r *appFunctionDependenciesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, ",")

	if len(idParts) != 2 {
		resp.Diagnostics.AddError(
			"Error Importing App Services Function Dependencies",
			"Could not import MongoDB Atlas App Services Function Dependencies.\nPlease ensure you run terraform import with project_id,appservices_app_id",
		)
		return
	}

	dependencies, err := getAppFunctionDependencies(r.bearer_token, idParts[0], idParts[1])
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing App Services Function Dependencies",
			"Unable to get app services function dependencies for subsequent import. got error: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("appservices_app_id"), idParts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("dependencies"), dependencies)...)
}
