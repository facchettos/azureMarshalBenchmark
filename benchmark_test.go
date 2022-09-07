package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
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
	tests := []struct {
		resourceId string
		apiversion string
	}{
		{"/subscriptions/%s/resourceGroups/jfacchet-rg/providers/Microsoft.Network/virtualNetworks/jfacchet-rg-vnet", "2022-05-01"},
		{"/subscriptions/%s/resourceGroups/jfacchet-rg/providers/Microsoft.Compute/virtualMachines/test-jfacchet", "2022-08-01"},
		{"/subscriptions/%s/resourceGroups/JFACCHET-RG/providers/Microsoft.Compute/sshPublicKeys/test-jfacchet_key", "2022-08-01"},
		{"/subscriptions/%s/resourceGroups/jfacchet-rg/providers/Microsoft.Network/publicIPAddresses/test-jfacchet-ip", "2022-05-01"},
		{"/subscriptions/%s/resourceGroups/jfacchet-rg/providers/Microsoft.Network/networkSecurityGroups/test-jfacchet-nsg", "2022-05-01"},
		//this fails because there is some dynamic guid in the result, which is constantly changing so the deep equal fails
		//		{"/subscriptions/%s/resourceGroups/jfacchet-rg/providers/Microsoft.Network/networkInterfaces/test-jfacchet986", "2022-05-01"},
		{"/subscriptions/%s/resourceGroups/JFACCHET-RG/providers/Microsoft.Compute/disks/test-jfacchet_OsDisk_1_da2eb7474ac94ffcba7416b463dabe10", "2022-07-02"},
	}

	for _, tc := range tests {
		t.Run(tc.resourceId, func(t *testing.T) {
			gr, err := rcCurrent.GetByID(context.Background(), fmt.Sprintf(tc.resourceId, subID), tc.apiversion)
			if err != nil {
				fmt.Println(err)
			}

			resource := arm.Resource{
				Resource: gr,
			}
			bytes, _ := resource.MarshalJSON()

			mapCurrent := make(map[string]interface{})
			json.Unmarshal(bytes, &mapCurrent)

			bytes = rcWrapperGlobal.getAndMarshal(fmt.Sprintf(tc.resourceId, subID), tc.apiversion)

			mapNew := make(map[string]interface{})
			json.Unmarshal(bytes, &mapNew)

			if !reflect.DeepEqual(mapCurrent, mapNew) {
				t.Error()
			}
		})
	}
}
