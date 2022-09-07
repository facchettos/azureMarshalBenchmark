package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/Azure/ARO-RP/pkg/util/arm"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/pkg/errors"
)

const fqdn = "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"

type rcWrapper struct {
	rc   mgmtfeatures.ResourcesClient
	sess *AzureSession
}

// This a somewhat modified version of the code from autorest. It returns a map[string]interface{} instead of
// the object. Most of the code in main is just used to get account creds

// GetByID gets a resource by ID.
// Parameters:
// resourceID - the fully qualified ID of the resource, including the resource name and resource type. Use the
// format,
// /subscriptions/{guid}/resourceGroups/{resource-group-name}/{resource-provider-namespace}/{resource-type}/{resource-name}
// APIVersion - the API version to use for the operation.
func (client rcWrapper) GetByID(ctx context.Context, resourceID string, APIVersion string) (map[string]interface{}, error) {
	client.rc.Authorizer = client.sess.Authorizer
	req, err := client.rc.GetByIDPreparer(ctx, resourceID, APIVersion)
	if err != nil {
		fmt.Println("error preparing")
		err = autorest.NewErrorWithError(err, "features.ResourcesClient", "GetByID", nil, "Failure preparing request")
		return nil, err
	}
	resp, err := client.rc.GetByIDSender(req)
	if err != nil {
		fmt.Println("error sending")
		return nil, autorest.NewErrorWithError(err, "features.ResourcesClient", "GetByID", resp, "Failure sending request")
	}
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		fmt.Println(string(b))
		return nil, errors.New("not expected status code")
	}

	result := make(map[string]interface{})
	err = json.NewDecoder(resp.Body).Decode(&result)

	return result, err
}

// AzureSession is an object representing session for subscription
type AzureSession struct {
	SubscriptionID string
	Authorizer     autorest.Authorizer
}

func newSessionFromFile() (*AzureSession, error) {
	authorizer, err := auth.NewAuthorizerFromFile(azure.PublicCloud.ResourceManagerEndpoint)

	if err != nil {
		return nil, errors.Wrap(err, "Can't initialize authorizer")
	}

	sess := AzureSession{
		SubscriptionID: subID,
		Authorizer:     authorizer,
	}

	return &sess, nil
}

var fieldsToTransfer []string = []string{"id", "name", "type", "condition", "apiVersion", "dependsOn", "location", "tags", "copy", "comments"}

func customMarshal(resource map[string]interface{}) []byte {
	outer := make(map[string]interface{})
	outer["properties"] = resource["properties"]
	for _, v := range fieldsToTransfer {
		if resource[v] != nil {
			outer[v] = resource[v]
		}
	}

	bytes, err := json.Marshal(outer)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	return bytes
}

func (rc rcWrapper) getAndMarshal(id, apiversion string) []byte {
	resource, err := rc.GetByID(context.Background(), id, apiversion)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	delete(resource, "etag")
	bytes, _ := json.Marshal(resource)

	return bytes
}

var subID string

func main() {
	var ok bool
	subID, ok = os.LookupEnv("AZURE_SUBSCRIPTION_ID")
	if !ok {
		fmt.Println("no sub id")
		os.Exit(1)
	}

	res := arm.Resource{}
	fmt.Println(res)
	sess, err := newSessionFromFile()
	rc := rcWrapper{
		rc:   mgmtfeatures.NewResourcesClient(subID),
		sess: sess,
	}

	if err != nil {
		fmt.Printf("%v\n", err)
		fmt.Println("error")
		os.Exit(1)
	}

	if err != nil {
		fmt.Println("error 2")
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	resource, err := rc.GetByID(context.Background(), fmt.Sprintf("/subscriptions/%s/resourceGroups/jfacchet-rg/providers/Microsoft.Network/networkSecurityGroups/test-jfacchet-nsg", subID), "2022-05-01")

	fmt.Println(err)
	bytes := customMarshal(resource)
	fmt.Println(string(bytes))
}
