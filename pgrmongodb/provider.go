package pgrmongodb

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ provider.Provider = &pgrmongodb_provider{}
)

func New() provider.Provider {
	return &pgrmongodb_provider{}
}

type pgrmongodb_provider struct{}

type pgrmongodbProviderModel struct {
	PublicKey  types.String `tfsdk:"public_key"`
	PrivateKey types.String `tfsdk:"private_key"`
}

type providerData struct {
	bearer_token string
	public_key   string
	private_key  string
}

func (p *pgrmongodb_provider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "pgrmongodb"
}

func (p *pgrmongodb_provider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Progressive MongoDB Atlas provider",
		Attributes: map[string]schema.Attribute{
			"public_key": schema.StringAttribute{
				Optional:    true,
				Description: "MongoDB Atlas public API key.",
			},
			"private_key": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "MongoDB Atlas private API key.",
			},
		},
	}
}

func (p *pgrmongodb_provider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data providerData
	var config pgrmongodbProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.PublicKey.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("public_key"),
			"Unknown MongoDB Atlas public API key",
			"The provider cannot authenticate as there is an unkonwn configuration value for the public API key. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the PGRMONGODB_PUBLICKEY environment variable.",
		)
	}

	if config.PrivateKey.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("private_key"),
			"Unknown MongoDB Atlas private API key",
			"The provider cannot authenticate as there is an unkonwn configuration value for the private API key. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the PGRMONGODB_PRIVATEKEY environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	public_key := os.Getenv("PGRMONGODB_PUBLICKEY")
	private_key := os.Getenv("PGRMONGODB_PRIVATEKEY")

	if !config.PublicKey.IsNull() {
		public_key = config.PublicKey.ValueString()
	}

	if !config.PrivateKey.IsNull() {
		private_key = config.PrivateKey.ValueString()
	}

	if public_key == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("public_key"),
			"Missing MongoDB Atlas public API key.",
			"The provider cannot authenticate to MongoDB Atlas without a valid public/private API key pair.",
		)
	}

	if private_key == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("private_key"),
			"Missing MongoDB Atlas private API key.",
			"The provider cannot authenticate to MongoDB Atlas without a valid public/private API key pair.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	bearer_token, err := getAppServicesAuthBearerToken(public_key, private_key)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to authenticate to MongoDB Atlas",
			"An unexpected error occured when authenicating to MongoDB Atlas. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Error: "+err.Error(),
		)
		return
	}

	data.bearer_token = bearer_token
	data.public_key = public_key
	data.private_key = private_key

	resp.DataSourceData = data
	resp.ResourceData = data
}

func (p *pgrmongodb_provider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewAtlasClusterContainerDataSource,
		NewAppFunctionExecuteDataSource,
	}
}

func (p *pgrmongodb_provider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewAppServicesAppResource,
		NewAppFunctionResource,
		NewAppFunctionDependencies,
	}
}

type AppServicesAuthResponse struct {
	AccessToken string `json:"access_token"`
	UserID      string `json:"user_id"`
	DeviceID    string `json:"device_id"`
}

func getAppServicesAuthBearerToken(pubkey string, privkey string) (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	var jsonStr = []byte(`{"username":"` + pubkey + `","apiKey":"` + privkey + `"}`)
	req, err := http.NewRequest("POST",
		"https://services.cloud.mongodb.com/api/admin/v3.0/auth/providers/mongodb-cloud/login",
		bytes.NewBuffer(jsonStr))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	r, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer r.Body.Close()
	if r.StatusCode == http.StatusOK {
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			return "", err
		}
		var respjson AppServicesAuthResponse
		if err := json.Unmarshal(bodyBytes, &respjson); err != nil { // Parse []byte to go struct pointer
			return "", err
		}
		return respjson.AccessToken, nil
	}
	return "", nil
}
