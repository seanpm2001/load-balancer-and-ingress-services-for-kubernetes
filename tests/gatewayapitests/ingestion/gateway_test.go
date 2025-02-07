/*
 * Copyright 2023-2024 VMware, Inc.
 * All Rights Reserved.
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*   http://www.apache.org/licenses/LICENSE-2.0
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
*/

package ingestion

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayfake "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned/fake"

	akogatewayapik8s "github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/ako-gateway-api/k8s"
	akogatewayapilib "github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/ako-gateway-api/lib"
	"github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/internal/k8s"
	"github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/internal/lib"
	"github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/pkg/utils"
	akogatewayapitests "github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/tests/gatewayapitests"
	"github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/tests/integrationtest"
)

var keyChan chan string

const (
	DEFAULT_NAMESPACE = "default"
)

func waitAndverify(t *testing.T, key string) {
	waitChan := make(chan int)
	go func() {
		time.Sleep(20 * time.Second)
		waitChan <- 1
	}()

	select {
	case data := <-keyChan:
		if key == "" {
			t.Fatalf("unpexpected key: %v", data)
		} else if data != key {
			t.Fatalf("error in match expected: %v, got: %v", key, data)
		}
	case <-waitChan:
		if key != "" {
			t.Fatalf("timed out waiting for %v", key)
		}
	}

}

func syncFuncForTest(key interface{}, wg *sync.WaitGroup) error {
	keyStr, ok := key.(string)
	if !ok {
		return nil
	}
	keyChan <- keyStr
	return nil
}

func setupQueue(stopCh <-chan struct{}) {
	ingestionQueue := utils.SharedWorkQueue().GetQueueByName(utils.ObjectIngestionLayer)
	wgIngestion := &sync.WaitGroup{}

	ingestionQueue.SyncFunc = syncFuncForTest
	ingestionQueue.Run(stopCh, wgIngestion)
}

func TestMain(m *testing.M) {
	akogatewayapitests.KubeClient = k8sfake.NewSimpleClientset()
	akogatewayapitests.GatewayClient = gatewayfake.NewSimpleClientset()
	integrationtest.KubeClient = akogatewayapitests.KubeClient

	os.Setenv("CLUSTER_NAME", "cluster")
	os.Setenv("CLOUD_NAME", "CLOUD_VCENTER")
	os.Setenv("SEG_NAME", "Default-Group")
	os.Setenv("POD_NAMESPACE", utils.AKO_DEFAULT_NS)
	os.Setenv("POD_NAME", "ako-0")

	// Set the user with prefix
	_ = lib.AKOControlConfig()
	lib.SetAKOUser(akogatewayapilib.Prefix)
	lib.SetNamePrefix(akogatewayapilib.Prefix)
	akoControlConfig := akogatewayapilib.AKOControlConfig()
	akoControlConfig.SetEventRecorder(lib.AKOGatewayEventComponent, akogatewayapitests.KubeClient, true)
	registeredInformers := []string{
		utils.ServiceInformer,
		utils.EndpointInformer,
		utils.SecretInformer,
		utils.NSInformer,
	}
	args := make(map[string]interface{})
	utils.NewInformers(utils.KubeClientIntf{ClientSet: akogatewayapitests.KubeClient}, registeredInformers, args)
	akoApi := integrationtest.InitializeFakeAKOAPIServer()
	defer akoApi.ShutDown()

	defer integrationtest.AviFakeClientInstance.Close()
	ctrl := akogatewayapik8s.SharedGatewayController()
	ctrl.InitGatewayAPIInformers(akogatewayapitests.GatewayClient)
	akoControlConfig.SetGatewayAPIClientset(akogatewayapitests.GatewayClient)
	stopCh := utils.SetupSignalHandler()
	ctrl.Start(stopCh)
	keyChan = make(chan string)

	ctrl.DisableSync = false

	ctrl.SetupEventHandlers(k8s.K8sinformers{Cs: akogatewayapitests.KubeClient})
	numWorkers := uint32(1)
	ctrl.SetupGatewayApiEventHandlers(numWorkers)
	setupQueue(stopCh)
	os.Exit(m.Run())
}

func TestGatewayCUD(t *testing.T) {
	gwName := "gw-example-00"
	gwClassName := "gw-class-example-00"
	gwKey := "Gateway/" + DEFAULT_NAMESPACE + "/" + gwName
	gateway := gatewayv1.Gateway{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "gateway.networking.k8s.io/v1beta1",
			Kind:       "Gateway",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      gwName,
			Namespace: "default",
		},
		Spec:   gatewayv1.GatewaySpec{},
		Status: gatewayv1.GatewayStatus{},
	}
	akogatewayapitests.SetGatewayGatewayClass(&gateway, gwClassName)
	akogatewayapitests.AddGatewayListener(&gateway, "listener-example", 80, gatewayv1.HTTPProtocolType, false)
	akogatewayapitests.SetListenerHostname(&gateway.Spec.Listeners[0], "foo.example.com")

	//create
	gw, err := akogatewayapitests.GatewayClient.GatewayV1().Gateways("default").Create(context.TODO(), &gateway, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Couldn't create, err: %+v", err)
	}
	t.Logf("Created %+v", gw.Name)
	waitAndverify(t, gwKey)

	//update
	akogatewayapitests.SetGatewayGatewayClass(&gateway, "gw-class-new")
	gw, err = akogatewayapitests.GatewayClient.GatewayV1().Gateways("default").Update(context.TODO(), &gateway, metav1.UpdateOptions{})
	if err != nil {
		t.Fatalf("Couldn't update, err: %+v", err)
	}
	t.Logf("Updated %+v", gw.Name)
	waitAndverify(t, gwKey)

	//delete
	err = akogatewayapitests.GatewayClient.GatewayV1().Gateways("default").Delete(context.TODO(), gateway.Name, metav1.DeleteOptions{})
	if err != nil {
		t.Fatalf("Couldn't delete, err: %+v", err)
	}
	t.Logf("Deleted %+v", gw.Name)
	waitAndverify(t, gwKey)
}

func TestGatewayInvalidListenerCount(t *testing.T) {
	gwName := "gw-example-01"
	gwClassName := "gw-class-example-01"
	gwKey := "Gateway/" + DEFAULT_NAMESPACE + "/" + gwName
	gateway := gatewayv1.Gateway{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "gateway.networking.k8s.io/v1beta1",
			Kind:       "Gateway",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      gwName,
			Namespace: "default",
		},
		Spec:   gatewayv1.GatewaySpec{},
		Status: gatewayv1.GatewayStatus{},
	}
	akogatewayapitests.SetGatewayGatewayClass(&gateway, gwClassName)

	//create
	gw, err := akogatewayapitests.GatewayClient.GatewayV1().Gateways("default").Create(context.TODO(), &gateway, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Couldn't create, err: %+v", err)
	}
	t.Logf("Created %+v", gw.Name)
	waitAndverify(t, "")

	//update
	akogatewayapitests.AddGatewayListener(&gateway, "listener-example", 80, gatewayv1.HTTPProtocolType, false)
	akogatewayapitests.SetListenerHostname(&gateway.Spec.Listeners[0], "*.example.com")
	gw, err = akogatewayapitests.GatewayClient.GatewayV1().Gateways("default").Update(context.TODO(), &gateway, metav1.UpdateOptions{})
	if err != nil {
		t.Fatalf("Couldn't update, err: %+v", err)
	}
	t.Logf("Updated %+v", gw.Name)
	waitAndverify(t, gwKey)

	akogatewayapitests.TeardownGateway(t, gwName, DEFAULT_NAMESPACE)
	waitAndverify(t, gwKey)

}

func TestGatewayInvalidAddress(t *testing.T) {
	gwName := "gw-example-02"
	gwClassName := "gw-class-example-02"
	gwKey := "Gateway/" + DEFAULT_NAMESPACE + "/" + gwName
	gateway := gatewayv1.Gateway{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "gateway.networking.k8s.io/v1beta1",
			Kind:       "Gateway",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      gwName,
			Namespace: "default",
		},
		Spec:   gatewayv1.GatewaySpec{},
		Status: gatewayv1.GatewayStatus{},
	}
	akogatewayapitests.SetGatewayGatewayClass(&gateway, gwClassName)
	akogatewayapitests.AddGatewayListener(&gateway, "listener-example", 80, gatewayv1.HTTPProtocolType, false)
	akogatewayapitests.SetListenerHostname(&gateway.Spec.Listeners[0], "foo.example.com")
	hostnameType := gatewayv1.AddressType("Hostname")
	gateway.Spec.Addresses = []gatewayv1.GatewayAddress{
		{
			Type:  &hostnameType,
			Value: "some.fqdn.address",
		},
	}

	//create
	gw, err := akogatewayapitests.GatewayClient.GatewayV1().Gateways("default").Create(context.TODO(), &gateway, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Couldn't create, err: %+v", err)
	}
	t.Logf("Created %+v", gw.Name)
	waitAndverify(t, "")

	//update with IPv6
	ipAddressType := gatewayv1.AddressType("IPAddress")
	gateway.Spec.Addresses = []gatewayv1.GatewayAddress{
		{
			Type: &ipAddressType,
			//TODO replace with constant from utils
			Value: "2001:db8:3333:4444:5555:6666:7777:8888",
		},
	}
	gw, err = akogatewayapitests.GatewayClient.GatewayV1().Gateways("default").Update(context.TODO(), &gateway, metav1.UpdateOptions{})
	if err != nil {
		t.Fatalf("Couldn't update, err: %+v", err)
	}
	t.Logf("Updated %+v", gw.Name)
	waitAndverify(t, "")

	//update with IPv4
	gateway.Spec.Addresses = []gatewayv1.GatewayAddress{
		{
			Type: &ipAddressType,
			//TODO replace with constant from utils
			Value: "1.2.3.4",
		},
	}
	gw, err = akogatewayapitests.GatewayClient.GatewayV1().Gateways("default").Update(context.TODO(), &gateway, metav1.UpdateOptions{})
	if err != nil {
		t.Fatalf("Couldn't update, err: %+v", err)
	}
	t.Logf("Updated %+v", gw.Name)
	waitAndverify(t, gwKey)

	//delete
	akogatewayapitests.TeardownGateway(t, gwName, DEFAULT_NAMESPACE)
	waitAndverify(t, gwKey)
}

func TestGatewayInvalidListenerHostname(t *testing.T) {
	gwName := "gw-example-03"
	gwClassName := "gw-class-example-03"
	gwKey := "Gateway/" + DEFAULT_NAMESPACE + "/" + gwName
	gateway := gatewayv1.Gateway{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "gateway.networking.k8s.io/v1beta1",
			Kind:       "Gateway",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      gwName,
			Namespace: "default",
		},
		Spec:   gatewayv1.GatewaySpec{},
		Status: gatewayv1.GatewayStatus{},
	}
	akogatewayapitests.SetGatewayGatewayClass(&gateway, gwClassName)
	akogatewayapitests.AddGatewayListener(&gateway, "listener-example", 80, gatewayv1.HTTPProtocolType, false)
	akogatewayapitests.SetListenerHostname(&gateway.Spec.Listeners[0], "*")

	//create
	gw, err := akogatewayapitests.GatewayClient.GatewayV1().Gateways("default").Create(context.TODO(), &gateway, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Couldn't create, err: %+v", err)
	}
	t.Logf("Created %+v", gw.Name)
	waitAndverify(t, "")

	//update
	akogatewayapitests.SetListenerHostname(&gateway.Spec.Listeners[0], "*.example.com")
	gw, err = akogatewayapitests.GatewayClient.GatewayV1().Gateways("default").Update(context.TODO(), &gateway, metav1.UpdateOptions{})
	if err != nil {
		t.Fatalf("Couldn't update, err: %+v", err)
	}
	t.Logf("Updated %+v", gw.Name)
	waitAndverify(t, gwKey)

	//delete
	akogatewayapitests.TeardownGateway(t, gwName, DEFAULT_NAMESPACE)
	waitAndverify(t, gwKey)
}

func TestGatewayInvalidListenerProtocol(t *testing.T) {
	gwName := "gw-example-04"
	gwClassName := "gw-class-example-04"
	gwKey := "Gateway/" + DEFAULT_NAMESPACE + "/" + gwName
	gateway := gatewayv1.Gateway{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "gateway.networking.k8s.io/v1beta1",
			Kind:       "Gateway",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      gwName,
			Namespace: "default",
		},
		Spec:   gatewayv1.GatewaySpec{},
		Status: gatewayv1.GatewayStatus{},
	}
	akogatewayapitests.SetGatewayGatewayClass(&gateway, gwClassName)
	akogatewayapitests.AddGatewayListener(&gateway, "listener-example", 80, gatewayv1.TCPProtocolType, false)
	akogatewayapitests.SetListenerHostname(&gateway.Spec.Listeners[0], "*.example.com")

	//create
	gw, err := akogatewayapitests.GatewayClient.GatewayV1().Gateways("default").Create(context.TODO(), &gateway, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Couldn't create, err: %+v", err)
	}
	t.Logf("Created %+v", gw.Name)
	waitAndverify(t, "")

	//update
	gateway.Spec.Listeners[0].Protocol = gatewayv1.HTTPProtocolType
	gw, err = akogatewayapitests.GatewayClient.GatewayV1().Gateways("default").Update(context.TODO(), &gateway, metav1.UpdateOptions{})
	if err != nil {
		t.Fatalf("Couldn't update, err: %+v", err)
	}
	t.Logf("Updated %+v", gw.Name)
	waitAndverify(t, gwKey)

	//delete
	akogatewayapitests.TeardownGateway(t, gwName, DEFAULT_NAMESPACE)
	waitAndverify(t, gwKey)
}

func TestGatewayInvalidListenerTLS(t *testing.T) {
	gwName := "gw-example-04"
	gwClassName := "gw-class-example-04"
	gwKey := "Gateway/" + DEFAULT_NAMESPACE + "/" + gwName

	ports := []int32{8080}
	secrets := []string{"secret-01"}
	for _, secret := range secrets {
		integrationtest.AddSecret(secret, DEFAULT_NAMESPACE, "cert", "key")
		waitAndverify(t, "Secret/"+DEFAULT_NAMESPACE+"/"+secret)
	}

	listeners := akogatewayapitests.GetListenersV1(ports, secrets...)
	tlsModePassthrough := gatewayv1.TLSModePassthrough
	listeners[0].TLS.Mode = &tlsModePassthrough
	//create
	akogatewayapitests.SetupGateway(t, gwName, DEFAULT_NAMESPACE, gwClassName, nil, listeners)
	waitAndverify(t, "")
	//update
	gateway, _ := akogatewayapitests.GatewayClient.GatewayV1().Gateways("default").Get(context.TODO(), gwName, metav1.GetOptions{})
	tlsModeTerminate := gatewayv1.TLSModeTerminate
	gateway.Spec.Listeners[0].TLS.Mode = &tlsModeTerminate
	gw, err := akogatewayapitests.GatewayClient.GatewayV1().Gateways("default").Update(context.TODO(), gateway, metav1.UpdateOptions{})
	if err != nil {
		t.Fatalf("Couldn't update, err: %+v", err)
	}
	t.Logf("Updated %+v", gw.Name)
	waitAndverify(t, gwKey)

	//delete
	akogatewayapitests.TeardownGateway(t, gwName, DEFAULT_NAMESPACE)
	waitAndverify(t, gwKey)
}

func TestGatewayClassCUD(t *testing.T) {
	gatewayClass := gatewayv1.GatewayClass{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "gateway.networking.k8s.io/v1beta1",
			Kind:       "GatewayClass",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "gw-class-example",
		},
		Spec: gatewayv1.GatewayClassSpec{
			ControllerName: "ako.vmware.com/avi-lb",
		},
		Status: gatewayv1.GatewayClassStatus{},
	}

	//create
	gw, err := akogatewayapitests.GatewayClient.GatewayV1().GatewayClasses().Create(context.TODO(), &gatewayClass, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Couldn't create, err: %+v", err)
	}
	t.Logf("Created %+v", gw.Name)
	waitAndverify(t, "GatewayClass/gw-class-example")

	//update
	testDesc := "test description for update"
	gatewayClass.Spec.Description = &testDesc
	gw, err = akogatewayapitests.GatewayClient.GatewayV1().GatewayClasses().Update(context.TODO(), &gatewayClass, metav1.UpdateOptions{})
	if err != nil {
		t.Fatalf("Couldn't update gatewayClass, err: %+v", err)
	}
	t.Logf("Updated %+v", gw.Name)
	waitAndverify(t, "GatewayClass/gw-class-example")

	//delete
	err = akogatewayapitests.GatewayClient.GatewayV1().GatewayClasses().Delete(context.TODO(), gatewayClass.Name, metav1.DeleteOptions{})
	if err != nil {
		t.Fatalf("Couldn't delete, err: %+v", err)
	}
	t.Logf("Deleted %+v", gw.Name)
	waitAndverify(t, "GatewayClass/gw-class-example")
}
