package pgrmongodb

import (
	"context"
	"regexp"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource = &appFunctionExecuteDataSource{}
)

func NewAppFunctionExecuteDataSource() datasource.DataSource {
	return &appFunctionExecuteDataSource{}
}

type appFunctionExecuteDataSource struct {
	bearer_token string
}

type appFunctionExecuteDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	ProjectID        types.String `tfsdk:"project_id"`
	AppServicesAppID types.String `tfsdk:"appservices_app_id"`
	FunctionName     types.String `tfsdk:"function_name"`
	FunctionArgs     types.Set    `tfsdk:"function_args"`
	ExecuteNextRun   types.Bool   `tfsdk:"execute_next_run"`
	ExecutionTimeout types.Int64  `tfsdk:"execution_timeout"`
	LastRun          types.String `tfsdk:"last_run"`
}

func (r *appFunctionExecuteDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_appfunctionexecute"
}

func (r *appFunctionExecuteDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Executes a MongoDB Atlas App Services Function",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"project_id": schema.StringAttribute{
				Description: "MongoDB Atlas project identifier. Sometime referred to as group id.",
				Required:    true,
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
			},
			"function_args": schema.SetAttribute{
				Description: "List of arguments to pass to the function for execution.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
			},
			"execute_next_run": schema.BoolAttribute{
				Description: "Determines if the next Terraform apply will execute the function or not.",
				Required:    true,
			},
			"execution_timeout": schema.Int64Attribute{
				Description: "Sets the timeout value for function invocation (seconds).",
				Optional:    true,
				Computed:    true,
			},
			"last_run": schema.StringAttribute{
				Description: "Timestamp for the last time this function executed successfully.",
				Computed:    true,
			},
		},
	}
}

func (r *appFunctionExecuteDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.bearer_token = req.ProviderData.(providerData).bearer_token
}

func (r *appFunctionExecuteDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state appFunctionExecuteDataSourceModel
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := state.ProjectID.ValueString()
	appServicesAppID := state.AppServicesAppID.ValueString()
	functionName := state.FunctionName.ValueString()
	exuecteNextRun := state.ExecuteNextRun.ValueBool()
	functionArgs := make([]string, 0, len(state.FunctionArgs.Elements()))
	executionTimeout := state.ExecutionTimeout.ValueInt64()
	diags = state.FunctionArgs.ElementsAs(ctx, &functionArgs, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if exuecteNextRun {
		tflog.Info(ctx, "executing mongodb atlas app services function")

		err := executeAppServicesFunctionByName(r.bearer_token, projectID, appServicesAppID, functionName, functionArgs, executionTimeout)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Execute MongoDB Atlas App Services Function",
				err.Error(),
			)
			return
		}

		currentTime := time.Now().UTC()
		iso8601Time := currentTime.Format(time.RFC3339)
		state.LastRun = types.StringValue(iso8601Time)
	} else {
		tflog.Info(ctx, "execute_next_run set to false for mongodb atlas app services function")
	}

	state.ID = state.ProjectID

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
