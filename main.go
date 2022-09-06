package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-02-01/resources"
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

func readJSON(path string) (*map[string]interface{}, error) {
	data, err := ioutil.ReadFile(path)

	if err != nil {
		return nil, errors.Wrap(err, "Can't open the file")
	}

	contents := make(map[string]interface{})
	err = json.Unmarshal(data, &contents)

	if err != nil {
		err = errors.Wrap(err, "Can't unmarshal file")
	}

	return &contents, err
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

func getGroups(sess *AzureSession) ([]string, error) {
	tab := make([]string, 0)
	var err error

	grClient := resources.NewGroupsClient(sess.SubscriptionID)
	grClient.Authorizer = sess.Authorizer

	for list, err := grClient.ListComplete(context.Background(), "", nil); list.NotDone(); err = list.Next() {
		if err != nil {
			return nil, errors.Wrap(err, "error traverising RG list")
		}
		rgName := *list.Value().ID
		tab = append(tab, rgName)
	}
	return tab, err
}

func getVM(sess *AzureSession, rg string) string {
	fmt.Println("get vm")

	vmClient := compute.NewVirtualMachinesClient(sess.SubscriptionID)
	vmClient.Authorizer = sess.Authorizer
	vm, err := vmClient.Get(context.Background(), "jfacchet-rg", "test-jfacchet", compute.InstanceView)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(*vm.Name)
	fmt.Println(*vm.ID)

	for vm, err := vmClient.ListComplete(context.Background(), rg); vm.NotDone(); err = vm.Next() {
		if err != nil {
			log.Print("got error while traverising RG list: ", err)
		}

		i := vm.Value()
		tags := []string{}
		for k, v := range i.Tags {
			tags = append(tags, fmt.Sprintf("%s?%s", k, *v))
		}
		tagsS := strings.Join(tags, "%")

		if len(i.Tags) > 0 {
			fmt.Printf("%s,%s,%s,<%s>\n", rg, *i.Name, *i.ID, tagsS)
		} else {
			fmt.Printf("%s,%s,%s\n", rg, *i.Name, *i.ID)
		}
	}
	fmt.Println("done")
	return *vm.ID
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

	return customMarshal(resource)
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

	fmt.Println("getgroups")
	groups, err := getGroups(sess)
	fmt.Println(len(groups))

	//fmt.Println(len(groups))

	if err != nil {
		fmt.Println("error 2")
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	id := getVM(sess, "jfacchet-rg")

	resource, err := rc.GetByID(context.Background(), id, "2022-08-01")
	fmt.Println(err)
	bytes := customMarshal(resource)
	fmt.Println(string(bytes))
}
