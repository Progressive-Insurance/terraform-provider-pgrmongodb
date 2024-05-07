package pgrmongodb

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// APP SERVICES APP

func createAppServicesApp(bearer_token string, projectID string, clusterName string, appName string) (map[string]interface{}, error) {
	var jsonStr = `{"name":"` + appName + `","data_source":{"name":"` + clusterName + `","type":"mongodb-atlas","config":{"clusterName":"` + clusterName + `"}}}`
	r, err := httpRequestWithBearerAuth(bearer_token, "POST", fmt.Sprintf("https://services.cloud.mongodb.com/api/admin/v3.0/groups/%s/apps", projectID), jsonStr, 10)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	resp, err := responseToMap(r)
	if err != nil {
		return nil, err
	}
	if r.StatusCode == http.StatusCreated {
		return resp, nil
	} else {
		return resp, fmt.Errorf("unable to create app services app %s. Got statuscode: %d", appName, r.StatusCode)
	}
}

func getAppServicesAppByName(bearer_token string, projectID string, appName string, clusterName string, atlasTrigger bool) (string, string, error) {
	found_app_id := ""
	found_service_id := ""

	uri := ""
	if atlasTrigger {
		uri = fmt.Sprintf("https://services.cloud.mongodb.com/api/admin/v3.0/groups/%s/apps?product=atlas", projectID)
	} else {
		uri = fmt.Sprintf("https://services.cloud.mongodb.com/api/admin/v3.0/groups/%s/apps", projectID)
	}

	r, err := httpRequestWithBearerAuth(bearer_token, "GET", uri, "", 10)
	if err != nil {
		return "", "", err
	}
	defer r.Body.Close()
	apps, err := responseToArrayOfMap(r)
	if err != nil {
		return "", "", err
	}

	for _, v := range apps {
		if v["name"] == appName {
			found_app_id = v["_id"].(string)
			break
		}
	}
	if found_app_id == "" {
		if !atlasTrigger {
			return getAppServicesAppByName(bearer_token, projectID, appName, clusterName, true)
		} else {
			err = fmt.Errorf("app services app %s does not exist", appName)
		}
	} else {
		found_service_id, err = getAppServicesLinkedDatasourceByAppID(bearer_token, projectID, found_app_id, clusterName)
	}
	return found_app_id, found_service_id, err
}

// gets service id where service name matches clustername (which is created by default after creating app)
func getAppServicesLinkedDatasourceByAppID(bearer_token string, projectID string, appID string, clusterName string) (string, error) {
	found_service_id := ""

	r, err := httpRequestWithBearerAuth(bearer_token, "GET", fmt.Sprintf("https://services.cloud.mongodb.com/api/admin/v3.0/groups/%s/apps/%s/services", projectID, appID), "", 10)
	if err != nil {
		return "", err
	}
	defer r.Body.Close()
	services, err := responseToArrayOfMap(r)
	if err != nil {
		return "", err
	}

	for _, v := range services {
		if v["name"] == clusterName {
			found_service_id = v["_id"].(string)
			break
		}
	}
	if found_service_id == "" {
		err = fmt.Errorf("app services app with name %s does not exist", clusterName)
	}
	return found_service_id, err
}

func deleteAppServicesApp(bearer_token string, projectID string, appID string) error {
	r, err := httpRequestWithBearerAuth(bearer_token, "DELETE", fmt.Sprintf("https://services.cloud.mongodb.com/api/admin/v3.0/groups/%s/apps/%s", projectID, appID), "", 10)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	if r.StatusCode == http.StatusNoContent {
		return nil
	} else {
		return fmt.Errorf("failed to delete app services app, got http status code %d", r.StatusCode)
	}
}

// APP SERVICES FUNCTION

func getAppServicesFunctionByName(bearer_token string, projectID string, appServicesAppID string, functionName string) (string, string, error) {
	found_function_id := ""
	found_function_code := ""

	r, err := httpRequestWithBearerAuth(bearer_token, "GET", fmt.Sprintf("https://services.cloud.mongodb.com/api/admin/v3.0/groups/%s/apps/%s/functions", projectID, appServicesAppID), "", 10)
	if err != nil {
		return "", "", err
	}
	defer r.Body.Close()
	apps, err := responseToArrayOfMap(r)
	if err != nil {
		return "", "", err
	}

	for _, v := range apps {
		if v["name"] == functionName {
			found_function_id = v["_id"].(string)
			_, found_function_code, err = getAppServicesFunctionByID(bearer_token, projectID, appServicesAppID, found_function_id)
			if err != nil {
				return "", "", err
			}
			break
		}
	}
	if found_function_id == "" {
		err = fmt.Errorf("app function %s does not exist", functionName)
	}
	return found_function_id, found_function_code, err
}

func getAppServicesFunctionByID(bearer_token string, projectID string, appServicesAppID string, functionID string) (string, string, error) {
	r, err := httpRequestWithBearerAuth(bearer_token, "GET", fmt.Sprintf("https://services.cloud.mongodb.com/api/admin/v3.0/groups/%s/apps/%s/functions/%s", projectID, appServicesAppID, functionID), "", 10)
	if err != nil {
		return "", "", err
	}
	defer r.Body.Close()
	respjson, err := responseToMap(r)
	if err != nil {
		return "", "", err
	}
	found_function_name := respjson["name"].(string)
	found_function_code := respjson["source"].(string)

	return found_function_name, found_function_code, err
}

func executeAppServicesFunctionByName(bearer_token string, projectID string, appServicesAppID string, functionName string, functionArgs []string, executionTimeout int64) error {
	var jsonStr string

	if len(functionArgs) == 0 {
		jsonStr = fmt.Sprintf(`{
			"name": "%s",
			"arguments": []
		}
		`, functionName)
	} else {
		jsonStr = fmt.Sprintf(`{
			"name": "%s",
			"arguments": ["%v"]
		}
		`, functionName, strings.Join(functionArgs, "\", \""))
	}

	if executionTimeout == 0 {
		executionTimeout = 10
	}

	r, err := httpRequestWithBearerAuth(bearer_token, "POST", fmt.Sprintf("https://services.cloud.mongodb.com/api/admin/v3.0/groups/%s/apps/%s/debug/execute_function?run_as_system=true", projectID, appServicesAppID), jsonStr, executionTimeout)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	if r.StatusCode == http.StatusOK {
		return nil
	} else {
		return fmt.Errorf("unable to execute app function %s. Got statuscode: %d. Using json: %s", functionName, r.StatusCode, jsonStr)
	}
}

func createAppServicesFunction(bearer_token string, projectID string, appServicesAppID string, functionName string, functionCode string) (string, error) {
	// normalize function code
	functionCode = strings.Replace(functionCode, "\n", "\\n", -1)
	functionCode = strings.Replace(functionCode, "\t", "\\t", -1)
	var jsonStr = fmt.Sprintf(`
		{
			"name": "%s",
			"private": false,
			"source": "%s",
			"run_as_system": true
	  	}
	`, functionName, functionCode)

	r, err := httpRequestWithBearerAuth(bearer_token, "POST", fmt.Sprintf("https://services.cloud.mongodb.com/api/admin/v3.0/groups/%s/apps/%s/functions", projectID, appServicesAppID), jsonStr, 10)
	if err != nil {
		return "", err
	}
	defer r.Body.Close()

	if r.StatusCode == http.StatusCreated {
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			return "", err
		}
		var respjson map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &respjson); err != nil { // Parse []byte to go struct pointer
			return "", err
		}
		created_function_id := respjson["_id"].(string)
		return created_function_id, err
	} else {
		return "", fmt.Errorf("unable to create app function %s. Got statuscode: %d. Using json: %s", functionName, r.StatusCode, jsonStr)
	}
}

func deleteAppServicesFunction(bearer_token string, projectID string, appServicesAppID string, functionID string) error {
	r, err := httpRequestWithBearerAuth(bearer_token, "DELETE", fmt.Sprintf("https://services.cloud.mongodb.com/api/admin/v3.0/groups/%s/apps/%s/functions/%s", projectID, appServicesAppID, functionID), "", 10)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	if r.StatusCode == http.StatusOK {
		return nil
	}
	return fmt.Errorf("failed to delete app function, got http status code %d", r.StatusCode)
}

// APP SERVICES FUNCTION DEPENDENCY

func getAppFunctionDependencies(bearer_token string, projectID string, appServicesAppID string) ([]types.String, error) {
	r, err := httpRequestWithBearerAuth(bearer_token, "GET", fmt.Sprintf("https://services.cloud.mongodb.com/api/admin/v3.0/groups/%s/apps/%s/dependencies", projectID, appServicesAppID), "", 10)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	respjson, err := responseToMap(r)
	if err != nil {
		return nil, err
	}

	elements := make([]types.String, 0, len(respjson["dependencies_list"].([]interface{})))
	for i := 0; i < len(respjson["dependencies_list"].([]interface{})); i++ {
		elements = append(elements, basetypes.NewStringValue(respjson["dependencies_list"].([]interface{})[i].(map[string]interface{})["name"].(string)+" "+respjson["dependencies_list"].([]interface{})[i].(map[string]interface{})["version"].(string)))
	}
	return elements, err
}

func getAppFunctionDependenciesStatus(bearer_token string, projectID string, appServicesAppID string) (string, string) {
	r, err := httpRequestWithBearerAuth(bearer_token, "GET", fmt.Sprintf("https://services.cloud.mongodb.com/api/admin/v3.0/groups/%s/apps/%s/dependencies/status", projectID, appServicesAppID), "", 10)
	if err != nil {
		return "", ""
	}
	defer r.Body.Close()
	respjson, err := responseToMap(r)
	if err != nil {
		return "", ""
	}
	return respjson["status"].(string), respjson["status_message"].(string)
}

func createAppFunctionDependencies(bearer_token string, projectID string, appServicesAppID string, dependencies []basetypes.StringValue) error {
	for i := 0; i < len(dependencies); i++ {
		depTokens := strings.Split(dependencies[i].ValueString(), " ")
		err := manageAppFunctionDependency(bearer_token, projectID, appServicesAppID, depTokens[0], depTokens[1], "PUT")
		if err != nil {
			return err
		}
	}
	return nil
}

func manageAppFunctionDependency(bearer_token string, projectID string, appServicesAppID string, dependency string, version string, http_method string) error {
	r, err := httpRequestWithBearerAuth(bearer_token, http_method, fmt.Sprintf("https://services.cloud.mongodb.com/api/admin/v3.0/groups/%s/apps/%s/dependencies/%s?version=%s", projectID, appServicesAppID, url.QueryEscape(dependency), version), "", 10)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	if r.StatusCode == http.StatusNoContent {
		tries := 0
		for ok := true; ok; /*ok = ok*/ {
			status, status_message := getAppFunctionDependenciesStatus(bearer_token, projectID, appServicesAppID)
			tries = tries + 1
			if status != "successful" {
				if status == "failed" {
					return fmt.Errorf("managing dependency %s : %s failed: %s (%s)", dependency, version, status_message, http_method)
				}
				time.Sleep(5 * time.Second)
				if tries >= 240 {
					return fmt.Errorf("exceeded max number of tries for successfully managing dependency %s : %s (%s)", dependency, version, http_method)
				}
			} else {
				break
			}
		}
	} else {
		return fmt.Errorf("unable to manage app function dependency %s : %s (%s). Got statuscode: %d", dependency, version, http_method, r.StatusCode)
	}
	return nil
}

func deleteAllAppFunctionDependencies(bearer_token string, projectID string, appServicesAppID string) error {
	dependencies, err := getAppFunctionDependencies(bearer_token, projectID, appServicesAppID)
	if err != nil {
		return fmt.Errorf("unable to get app function dependencies for subsequent deletion. got error: " + err.Error())
	}
	for i := 0; i < len(dependencies); i++ {
		depTokens := strings.Split(dependencies[i].ValueString(), " ")
		err := manageAppFunctionDependency(bearer_token, projectID, appServicesAppID, depTokens[0], depTokens[1], "DELETE")
		if err != nil {
			return err
		}
	}
	return nil
}

// ATLAS CLUSTER CONTAINER
func getClusterContainers(pubkey string, privkey string, projectID string, providerName string) (map[string]string, map[string]string, error) {
	cidrs := make(map[string]string)
	ids := make(map[string]string)
	url := fmt.Sprintf("https://cloud.mongodb.com/api/atlas/v2/groups/%s/containers?providerName=%s", projectID, providerName)
	response, err := digestRequest("GET", url, pubkey, privkey, []byte(""), "application/vnd.atlas.2023-01-01+json")
	if err != nil {
		return nil, nil, err
	}
	ncontainers := len(response["results"].([]interface{}))
	for i := 0; i < ncontainers; i++ {
		container := response["results"].([]interface{})[i].(map[string]interface{})
		regionNormalized := "ERR"
		if providerName == "AWS" {
			regionNormalized = container["regionName"].(string)
		} else if providerName == "AZURE" {
			regionNormalized = container["region"].(string)
		} else {
			return nil, nil, fmt.Errorf("%s not supported provider", providerName)
		}
		container_key := fmt.Sprintf("%s:%s", container["providerName"], regionNormalized)
		cidrs[container_key] = container["atlasCidrBlock"].(string)
		ids[container_key] = container["id"].(string)
	}
	return ids, cidrs, err
}
