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

package nodes

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	akogatewayapilib "github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/ako-gateway-api/lib"
	"github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/internal/lib"
	"github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/internal/nodes"
	"github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/pkg/utils"
)

type AviObjectGraph struct {
	*nodes.AviObjectGraph
}

func NewAviObjectGraph() *AviObjectGraph {
	return &AviObjectGraph{&nodes.AviObjectGraph{}}
}

func (o *AviObjectGraph) BuildGatewayVs(gateway *gatewayv1.Gateway, key string) {
	o.Lock.Lock()
	defer o.Lock.Unlock()

	var vsNode *nodes.AviEvhVsNode
	vsNode = o.BuildGatewayParent(gateway, key)

	o.AddModelNode(vsNode)
	utils.AviLog.Infof("key: %s, msg: checksum for AVI VS object %v", key, vsNode.GetCheckSum())
}

func (o *AviObjectGraph) BuildGatewayParent(gateway *gatewayv1.Gateway, key string) *nodes.AviEvhVsNode {
	vsName := akogatewayapilib.GetGatewayParentName(gateway.Namespace, gateway.Name)
	parentVsNode := &nodes.AviEvhVsNode{
		Name:               vsName,
		Tenant:             lib.GetTenant(),
		ServiceEngineGroup: lib.GetSEGName(),
		ApplicationProfile: utils.DEFAULT_L7_APP_PROFILE,
		NetworkProfile:     utils.DEFAULT_TCP_NW_PROFILE,
		EVHParent:          true,
		VrfContext:         lib.GetVrf(),
		ServiceMetadata: lib.ServiceMetadataObj{
			Gateway: gateway.Namespace + "/" + gateway.Name,
		},
	}

	parentVsNode.PortProto = BuildPortProtocols(gateway, key)

	tlsNodes := BuildTLSNodesForGateway(gateway, key)
	if len(tlsNodes) > 0 {
		parentVsNode.SSLKeyCertRefs = tlsNodes
	}

	vsvipNode := BuildVsVipNodeForGateway(gateway, parentVsNode.Name)
	parentVsNode.VSVIPRefs = []*nodes.AviVSVIPNode{vsvipNode}

	return parentVsNode
}

func BuildPortProtocols(gateway *gatewayv1.Gateway, key string) []nodes.AviPortHostProtocol {
	var portProtocols []nodes.AviPortHostProtocol
	for _, listener := range gateway.Spec.Listeners {
		pp := nodes.AviPortHostProtocol{Port: int32(listener.Port), Protocol: string(listener.Protocol)}
		//TLS config on listener is present
		if listener.TLS != nil && len(listener.TLS.CertificateRefs) > 0 {
			pp.EnableSSL = true
		}
		portProtocols = append(portProtocols, pp)
	}
	return portProtocols
}

func BuildTLSNodesForGateway(gateway *gatewayv1.Gateway, key string) []*nodes.AviTLSKeyCertNode {
	var tlsNodes []*nodes.AviTLSKeyCertNode
	var ns, name string
	cs := utils.GetInformers().ClientSet
	for _, listener := range gateway.Spec.Listeners {
		if listener.TLS != nil {
			for _, certRef := range listener.TLS.CertificateRefs {
				//kind is validated at ingestion
				if certRef.Namespace == nil || *certRef.Namespace == "" {
					ns = gateway.Namespace
				} else {
					ns = string(*certRef.Namespace)
				}
				name = string(certRef.Name)
				secretObj, err := cs.CoreV1().Secrets(ns).Get(context.TODO(), name, metav1.GetOptions{})
				if err != nil || secretObj == nil {
					utils.AviLog.Warnf("key: %s, msg: secret %s has been deleted, err: %s", key, name, err)
					continue
				}
				tlsNode := TLSNodeFromSecret(secretObj, string(*listener.Hostname), name, key)
				tlsNodes = append(tlsNodes, tlsNode)
			}
		}
	}
	return tlsNodes
}

func TLSNodeFromSecret(secretObj *corev1.Secret, hostname, certName, key string) *nodes.AviTLSKeyCertNode {
	keycertMap := secretObj.Data
	tlscert, ok := keycertMap[utils.K8S_TLS_SECRET_CERT]
	if !ok {
		utils.AviLog.Infof("key: %s, msg: certificate not found for secret: %s", key, secretObj.Name)
	}
	tlskey, ok := keycertMap[utils.K8S_TLS_SECRET_KEY]
	if !ok {
		utils.AviLog.Infof("key: %s, msg: key not found for secret: %s", key, secretObj.Name)
	}
	tlsNode := &nodes.AviTLSKeyCertNode{
		Name:   lib.GetTLSKeyCertNodeName("", hostname, certName),
		Tenant: lib.GetTenant(),
		Type:   lib.CertTypeVS,
		Key:    tlskey,
		Cert:   tlscert,
	}
	return tlsNode
}

func BuildVsVipNodeForGateway(gateway *gatewayv1.Gateway, vsName string) *nodes.AviVSVIPNode {
	vsvipNode := &nodes.AviVSVIPNode{
		Name:        lib.GetVsVipName(vsName),
		Tenant:      lib.GetTenant(),
		VrfContext:  lib.GetVrf(),
		VipNetworks: utils.GetVipNetworkList(),
	}

	//Type is validated at ingestion
	//TODO IPV6 handdling
	if len(gateway.Spec.Addresses) == 1 {
		vsvipNode.IPAddress = gateway.Spec.Addresses[0].Value
	}
	return vsvipNode
}

func DeleteTLSNode(key string, object *AviObjectGraph, gateway *gatewayv1.Gateway, secretObj *corev1.Secret, encodedCertNameIndexMap map[string][]int) {
	var tlsNodes []*nodes.AviTLSKeyCertNode
	_, _, secretName := lib.ExtractTypeNameNamespace(key)
	evhVsCertRefs := object.GetAviEvhVS()[0].SSLKeyCertRefs
	for _, listener := range gateway.Spec.Listeners {
		if listener.TLS != nil {
			for _, certRef := range listener.TLS.CertificateRefs {
				name := string(certRef.Name)
				encodedCertName := lib.GetTLSKeyCertNodeName("", string(*listener.Hostname), name)
				indexlist, exists := encodedCertNameIndexMap[encodedCertName]
				if exists {
					if name != secretName {
						tlsNodes = append(tlsNodes, evhVsCertRefs[indexlist[0]])
					}
					if len(indexlist) > 1 {
						encodedCertNameIndexMap[encodedCertName] = indexlist[1:]
					} else {
						delete(encodedCertNameIndexMap, encodedCertName)
					}
				}
			}
		}
	}
	utils.AviLog.Infof("key: %s, msg: Updated cert_refs in parentVS: %s", key, object.GetAviEvhVS()[0].Name)
	object.GetAviEvhVS()[0].SSLKeyCertRefs = tlsNodes
}

func AddTLSNode(key string, object *AviObjectGraph, gateway *gatewayv1.Gateway, secretObj *corev1.Secret, encodedCertNameIndexMap map[string][]int) {
	var tlsNodes []*nodes.AviTLSKeyCertNode
	_, _, secretName := lib.ExtractTypeNameNamespace(key)
	evhVsCertRefs := object.GetAviEvhVS()[0].SSLKeyCertRefs
	for _, listener := range gateway.Spec.Listeners {
		if listener.TLS != nil {
			for _, certRef := range listener.TLS.CertificateRefs {
				name := string(certRef.Name)
				encodedCertName := lib.GetTLSKeyCertNodeName("", string(*listener.Hostname), name)
				indexlist, exists := encodedCertNameIndexMap[encodedCertName]
				if exists {
					tlsNodes = append(tlsNodes, evhVsCertRefs[indexlist[0]])
					if len(indexlist) > 1 {
						encodedCertNameIndexMap[encodedCertName] = indexlist[1:]
					} else {
						delete(encodedCertNameIndexMap, encodedCertName)
					}
				} else {
					if name == secretName {
						tlsNode := TLSNodeFromSecret(secretObj, string(*listener.Hostname), name, key)
						tlsNodes = append(tlsNodes, tlsNode)
					}
				}
			}
		}
	}
	utils.AviLog.Infof("key: %s, msg: Updated cert_refs in parentVS: %s", key, object.GetAviEvhVS()[0].Name)
	object.GetAviEvhVS()[0].SSLKeyCertRefs = tlsNodes
}
