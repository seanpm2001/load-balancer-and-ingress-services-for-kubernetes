/*
Copyright The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v1beta1 "github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/pkg/apis/ako/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeHTTPRules implements HTTPRuleInterface
type FakeHTTPRules struct {
	Fake *FakeAkoV1beta1
	ns   string
}

var httprulesResource = schema.GroupVersionResource{Group: "ako.vmware.com", Version: "v1beta1", Resource: "httprules"}

var httprulesKind = schema.GroupVersionKind{Group: "ako.vmware.com", Version: "v1beta1", Kind: "HTTPRule"}

// Get takes name of the hTTPRule, and returns the corresponding hTTPRule object, and an error if there is any.
func (c *FakeHTTPRules) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1beta1.HTTPRule, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(httprulesResource, c.ns, name), &v1beta1.HTTPRule{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.HTTPRule), err
}

// List takes label and field selectors, and returns the list of HTTPRules that match those selectors.
func (c *FakeHTTPRules) List(ctx context.Context, opts v1.ListOptions) (result *v1beta1.HTTPRuleList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(httprulesResource, httprulesKind, c.ns, opts), &v1beta1.HTTPRuleList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1beta1.HTTPRuleList{ListMeta: obj.(*v1beta1.HTTPRuleList).ListMeta}
	for _, item := range obj.(*v1beta1.HTTPRuleList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested hTTPRules.
func (c *FakeHTTPRules) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(httprulesResource, c.ns, opts))

}

// Create takes the representation of a hTTPRule and creates it.  Returns the server's representation of the hTTPRule, and an error, if there is any.
func (c *FakeHTTPRules) Create(ctx context.Context, hTTPRule *v1beta1.HTTPRule, opts v1.CreateOptions) (result *v1beta1.HTTPRule, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(httprulesResource, c.ns, hTTPRule), &v1beta1.HTTPRule{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.HTTPRule), err
}

// Update takes the representation of a hTTPRule and updates it. Returns the server's representation of the hTTPRule, and an error, if there is any.
func (c *FakeHTTPRules) Update(ctx context.Context, hTTPRule *v1beta1.HTTPRule, opts v1.UpdateOptions) (result *v1beta1.HTTPRule, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(httprulesResource, c.ns, hTTPRule), &v1beta1.HTTPRule{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.HTTPRule), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeHTTPRules) UpdateStatus(ctx context.Context, hTTPRule *v1beta1.HTTPRule, opts v1.UpdateOptions) (*v1beta1.HTTPRule, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(httprulesResource, "status", c.ns, hTTPRule), &v1beta1.HTTPRule{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.HTTPRule), err
}

// Delete takes name of the hTTPRule and deletes it. Returns an error if one occurs.
func (c *FakeHTTPRules) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(httprulesResource, c.ns, name, opts), &v1beta1.HTTPRule{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeHTTPRules) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(httprulesResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1beta1.HTTPRuleList{})
	return err
}

// Patch applies the patch and returns the patched hTTPRule.
func (c *FakeHTTPRules) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1beta1.HTTPRule, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(httprulesResource, c.ns, name, pt, data, subresources...), &v1beta1.HTTPRule{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.HTTPRule), err
}
