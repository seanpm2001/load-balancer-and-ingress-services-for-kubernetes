/*
 * Copyright 2019-2023 VMware, Inc.
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

package objects

import (
	"os"
	"sync"

	akov1alpha1 "github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/pkg/apis/ako/v1alpha1"
	"github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/pkg/utils"
)

var WCPInstance *WCPLister
var wcponce sync.Once

func SharedWCPLister() *WCPLister {
	wcponce.Do(func() {
		WCPInstance = &WCPLister{
			NamespaceTier1LrCache: NewObjectMapStore(),
			NamespaceNetworkCache: NewObjectMapStore(),
		}
	})
	return WCPInstance
}

type WCPLister struct {
	// namespace -> tier1lr
	NamespaceTier1LrCache *ObjectMapStore
	NamespaceNetworkCache *ObjectMapStore
}

func (w *WCPLister) UpdateNamespaceTier1LrCache(namespace, t1lr string) {
	// w.NamespaceTier1LrCache.ObjLock.Lock()
	// defer w.NamespaceTier1LrCache.ObjLock.Unlock()
	w.NamespaceTier1LrCache.AddOrUpdate(namespace, t1lr)
}

func (w *WCPLister) RemoveNamespaceTier1LrCache(namespace string) {
	// w.NamespaceTier1LrCache.ObjLock.Lock()
	// defer w.NamespaceTier1LrCache.ObjLock.Unlock()
	w.NamespaceTier1LrCache.Delete(namespace)
}

func (w *WCPLister) GetT1LrForNamespace(namespace ...string) string {
	if utils.IsVCFCluster() {
		found, t1lr := w.NamespaceTier1LrCache.Get(namespace[0])
		if found {
			return t1lr.(string)
		}
	}
	return os.Getenv("NSXT_T1_LR")
}

func (w *WCPLister) UpdateNamespaceNetworkCache(namespace, networkName string) {
	w.NamespaceNetworkCache.AddOrUpdate(namespace, networkName)
}

func (w *WCPLister) RemoveNamespaceNetworkCache(namespace string) {
	w.NamespaceNetworkCache.Delete(namespace)
}

func (w *WCPLister) GetNetworkForNamespace(namespace ...string) []akov1alpha1.AviInfraSettingVipNetwork {
	if utils.IsVCFCluster() && len(namespace) > 0 {
		found, networkName := w.NamespaceNetworkCache.Get(namespace[0])
		if found {
			return []akov1alpha1.AviInfraSettingVipNetwork{
				{
					NetworkName: networkName.(string),
				},
			}
		}
	}
	return utils.GetVipNetworkList()
}