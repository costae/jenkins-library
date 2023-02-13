package cmd

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/Jeffail/gabs/v2"
	"github.com/SAP/jenkins-library/pkg/cpi"
	piperhttp "github.com/SAP/jenkins-library/pkg/http"
	"github.com/SAP/jenkins-library/pkg/log"
	"github.com/SAP/jenkins-library/pkg/telemetry"
	"github.com/pkg/errors"
)

func getPackageList(config getPackageListOptions, telemetryData *telemetry.CustomData, commonPipelineEnvironment *getPackageListCommonPipelineEnvironment) {
	// Utils can be used wherever the command.ExecRunner interface is expected.
	// It can also be used for example as a mavenExecRunner.
	httpClient := &piperhttp.Client{}

	// For HTTP calls import  piperhttp "github.com/SAP/jenkins-library/pkg/http"
	// and use a  &piperhttp.Client{} in a custom system
	// Example: step checkmarxExecuteScan.go

	// Error situations should be bubbled up until they reach the line below which will then stop execution
	// through the log.Entry().Fatal() call leading to an os.Exit(1) in the end.
	err := runGetPackageList(&config, telemetryData, httpClient, commonPipelineEnvironment)
	if err != nil {
		log.Entry().WithError(err).Fatal("step execution failed")
	}
}

func runGetPackageList(config *getPackageListOptions, telemetryData *telemetry.CustomData, httpClient piperhttp.Sender, commonPipelineEnvironment *getPackageListCommonPipelineEnvironment) error {
	clientOptions := piperhttp.ClientOptions{}
	header := make(http.Header)
	header.Add("Accept", "application/json")
	serviceKey, err := cpi.ReadCpiServiceKey(config.APIServiceKey)
	if err != nil {
		return err
	}
	servieEndpointURL := fmt.Sprintf("%s/api/v1/IntegrationPackages", serviceKey.OAuth.Host)
	tokenParameters := cpi.TokenParameters{TokenURL: serviceKey.OAuth.OAuthTokenProviderURL, Username: serviceKey.OAuth.ClientID, Password: serviceKey.OAuth.ClientSecret, Client: httpClient}
	token, err := cpi.CommonUtils.GetBearerToken(tokenParameters)
	if err != nil {
		return errors.Wrap(err, "failed to fetch Bearer Token")
	}
	clientOptions.Token = fmt.Sprintf("Bearer %s", token)
	httpClient.SetOptions(clientOptions)
	httpMethod := "GET"
	serviceEndpointResp, httpErr := httpClient.SendRequest(httpMethod, servieEndpointURL, nil, header, nil)

	if httpErr != nil {
		return errors.Wrapf(httpErr, "HTTP %v request to %v failed with error", httpMethod, servieEndpointURL)
	}

	if serviceEndpointResp != nil && serviceEndpointResp.Body != nil {
		defer serviceEndpointResp.Body.Close()
	}

	if serviceEndpointResp == nil {
		return errors.Errorf("did not retrieve a HTTP response: %v", httpErr)
	}

	if serviceEndpointResp.StatusCode == 200 {
		bodyText, readErr := ioutil.ReadAll(serviceEndpointResp.Body)
		if readErr != nil {
			return errors.Wrap(readErr, "HTTP response body could not be read")
		}
		jsonResponse, parsingErr := gabs.ParseJSON([]byte(bodyText))
		if parsingErr != nil {
			return errors.Wrapf(parsingErr, "HTTP response body could not be parsed as JSON: %v", string(bodyText))
		}

		for _, child := range jsonResponse.S("d", "results").Children() {
			// iflowID := strings.ReplaceAll(child.Path("Name").String(), "\"", "")
			// if iflowID == config.IntegrationFlowID {
			entryPoints := child.S("Id")
			finalEndpoint := entryPoints.Data().(string)
			commonPipelineEnvironment.custom.integrationPackageList = finalEndpoint
			return nil

		}
	}

	responseBody, readErr := ioutil.ReadAll(serviceEndpointResp.Body)

	if readErr != nil {
		return errors.Wrapf(readErr, "HTTP response body could not be read, Response status code: %v", serviceEndpointResp.StatusCode)
	}

	log.Entry().Errorf("a HTTP error occurred!  Response body: %v, Response status code: %v", string(responseBody), serviceEndpointResp.StatusCode)
	return errors.Errorf("Unable to get integration flow service endpoint, Response Status code: %v", serviceEndpointResp.StatusCode)
}
