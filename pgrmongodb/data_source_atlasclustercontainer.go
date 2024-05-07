package pgrmongodb

import (
	"context"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource = &atlasClusterContainerDataSource{}
)

func NewAtlasClusterContainerDataSource() datasource.DataSource {
	return &atlasClusterContainerDataSource{}
}

type atlasClusterContainerDataSource struct {
	public_key  string
	private_key string
}

type atlasClusterContainerDataSourceModel struct {
	ID            types.String `tfsdk:"id"`
	ProjectID     types.String `tfsdk:"project_id"`
	CIDRs         types.Map    `tfsdk:"cidrs"`
	IDs           types.Map    `tfsdk:"ids"`
	CloudProvider types.String `tfsdk:"cloud_provider"`
}

func (r *atlasClusterContainerDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_atlasclustercontainer"
}

func (r *atlasClusterContainerDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Data lookup for MongoDB Atlas Cluster network container",
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
			"cloud_provider": schema.StringAttribute{
				Description: "MongoDB Atlas cloud provider to retrieve network container information for.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"AWS", "GCP", "AZURE"}...),
				},
			},
			"cidrs": schema.MapAttribute{
				Description: "Map of container cidrs by region.",
				ElementType: types.StringType,
				Computed:    true,
			},
			"ids": schema.MapAttribute{
				Description: "Map of container ids by region.",
				ElementType: types.StringType,
				Computed:    true,
			},
		},
	}
}

func (r *atlasClusterContainerDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.public_key = req.ProviderData.(providerData).public_key
	r.private_key = req.ProviderData.(providerData).private_key
}

func (r *atlasClusterContainerDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state atlasClusterContainerDataSourceModel
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID := state.ProjectID.ValueString()
	cloudProvider := state.CloudProvider.ValueString()
	tflog.Info(ctx, "reading mongodb atlas cluster network container")
	ids, cidrs, err := getClusterContainers(r.public_key, r.private_key, projectID, cloudProvider)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Get MongoDB Atlas Cluster Containers",
			err.Error(),
		)
		return
	}

	idElements := make(map[string]attr.Value)
	for key, value := range ids {
		idElements[key] = types.StringValue(value)
	}
	cidrElements := make(map[string]attr.Value)
	for key, value := range cidrs {
		cidrElements[key] = types.StringValue(value)
	}

	ids_map, diags := types.MapValue(types.StringType, idElements)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	cidrs_map, diags := types.MapValue(types.StringType, cidrElements)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.ID = state.ProjectID
	state.IDs = ids_map
	state.CIDRs = cidrs_map

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
