package main

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/Azure/ARO-RP/pkg/util/arm"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
)

var result []byte
var bytesFromAzure []byte
var session *AzureSession
var resultFromNew []byte
var rcWrapperGlobal rcWrapper
var rcCurrent mgmtfeatures.ResourcesClient
var genericObjectGlobal mgmtfeatures.GenericResource
var mapInterfaceNew map[string]interface{}

func init() {
	var ok bool
	subID, ok = os.LookupEnv("AZURE_SUBSCRIPTION_ID")
	if !ok {
		fmt.Println("no sub id")
		os.Exit(1)
	}
	session, _ = newSessionFromFile()
	rcWrapperGlobal = rcWrapper{
		rc:   mgmtfeatures.NewResourcesClient(subID),
		sess: session,
	}
	rcCurrent = mgmtfeatures.NewResourcesClient(subID)
	rcCurrent.Authorizer = session.Authorizer

	mapInterfaceNew, _ = rcWrapperGlobal.GetByID(context.Background(), fmt.Sprintf("/subscriptions/%s/resourceGroups/jfacchet-rg/providers/Microsoft.Compute/virtualMachines/test-jfacchet", subID), "2022-08-01")

	genericObjectGlobal, _ = rcCurrent.GetByID(context.Background(), fmt.Sprintf("/subscriptions/%s/resourceGroups/jfacchet-rg/providers/Microsoft.Compute/virtualMachines/test-jfacchet", subID), "2022-08-01")
}

func BenchmarkGetAndMarshal(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		result = rcWrapperGlobal.getAndMarshal(fmt.Sprintf("/subscriptions/%s/resourceGroups/jfacchet-rg/providers/Microsoft.Compute/virtualMachines/test-jfacchet", subID), "2022-08-01")
	}
}

func BenchmarkCurrentVersion(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		gr, err := rcCurrent.GetByID(context.Background(), fmt.Sprintf("/subscriptions/%s/resourceGroups/jfacchet-rg/providers/Microsoft.Compute/virtualMachines/test-jfacchet", subID), "2022-08-01")
		if err != nil {
			fmt.Println(err)
		}

		resource := arm.Resource{
			Resource: gr,
		}
		result, _ = resource.MarshalJSON()
	}
}

func BenchmarkMarshalOnlyNew(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		result = customMarshal(mapInterfaceNew)
	}
}

func BenchmarkMarshalOnlyOld(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {

		resource := arm.Resource{
			Resource: genericObjectGlobal,
		}
		result, _ = resource.MarshalJSON()
	}
}

func TestResults(t *testing.T) {
	sess, _ := newSessionFromFile()
	rc := mgmtfeatures.NewResourcesClient(subID)
	rc.Authorizer = sess.Authorizer

	gr, err := rc.GetByID(context.Background(), fmt.Sprintf("/subscriptions/%s/resourceGroups/jfacchet-rg/providers/Microsoft.Compute/virtualMachines/test-jfacchet", subID), "2022-08-01")
	if err != nil {
		fmt.Println(err)
	}

	resource := arm.Resource{
		Resource: gr,
	}
	bytes, _ := resource.MarshalJSON()
	fmt.Println(string(bytes))
	fmt.Println("")
	rcW := rcWrapper{
		rc:   mgmtfeatures.NewResourcesClient(subID),
		sess: sess,
	}
	bytes = rcW.getAndMarshal(fmt.Sprintf("/subscriptions/%s/resourceGroups/jfacchet-rg/providers/Microsoft.Compute/virtualMachines/test-jfacchet", subID), "2022-08-01")
	fmt.Println(string(bytes))
	fmt.Println("")
}
