/*
Copyright The KCP Authors.

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
	json "encoding/json"
	"fmt"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"

	v1alpha1 "github.com/kcp-dev/contrib-tmc/apis/scheduling/v1alpha1"
	schedulingv1alpha1 "github.com/kcp-dev/contrib-tmc/client/applyconfiguration/scheduling/v1alpha1"
)

// FakePlacements implements PlacementInterface
type FakePlacements struct {
	Fake *FakeSchedulingV1alpha1
}

var placementsResource = v1alpha1.SchemeGroupVersion.WithResource("placements")

var placementsKind = v1alpha1.SchemeGroupVersion.WithKind("Placement")

// Get takes name of the placement, and returns the corresponding placement object, and an error if there is any.
func (c *FakePlacements) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.Placement, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(placementsResource, name), &v1alpha1.Placement{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Placement), err
}

// List takes label and field selectors, and returns the list of Placements that match those selectors.
func (c *FakePlacements) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.PlacementList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(placementsResource, placementsKind, opts), &v1alpha1.PlacementList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.PlacementList{ListMeta: obj.(*v1alpha1.PlacementList).ListMeta}
	for _, item := range obj.(*v1alpha1.PlacementList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested placements.
func (c *FakePlacements) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(placementsResource, opts))
}

// Create takes the representation of a placement and creates it.  Returns the server's representation of the placement, and an error, if there is any.
func (c *FakePlacements) Create(ctx context.Context, placement *v1alpha1.Placement, opts v1.CreateOptions) (result *v1alpha1.Placement, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(placementsResource, placement), &v1alpha1.Placement{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Placement), err
}

// Update takes the representation of a placement and updates it. Returns the server's representation of the placement, and an error, if there is any.
func (c *FakePlacements) Update(ctx context.Context, placement *v1alpha1.Placement, opts v1.UpdateOptions) (result *v1alpha1.Placement, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(placementsResource, placement), &v1alpha1.Placement{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Placement), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakePlacements) UpdateStatus(ctx context.Context, placement *v1alpha1.Placement, opts v1.UpdateOptions) (*v1alpha1.Placement, error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateSubresourceAction(placementsResource, "status", placement), &v1alpha1.Placement{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Placement), err
}

// Delete takes name of the placement and deletes it. Returns an error if one occurs.
func (c *FakePlacements) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteActionWithOptions(placementsResource, name, opts), &v1alpha1.Placement{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakePlacements) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(placementsResource, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.PlacementList{})
	return err
}

// Patch applies the patch and returns the patched placement.
func (c *FakePlacements) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.Placement, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(placementsResource, name, pt, data, subresources...), &v1alpha1.Placement{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Placement), err
}

// Apply takes the given apply declarative configuration, applies it and returns the applied placement.
func (c *FakePlacements) Apply(ctx context.Context, placement *schedulingv1alpha1.PlacementApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.Placement, err error) {
	if placement == nil {
		return nil, fmt.Errorf("placement provided to Apply must not be nil")
	}
	data, err := json.Marshal(placement)
	if err != nil {
		return nil, err
	}
	name := placement.Name
	if name == nil {
		return nil, fmt.Errorf("placement.Name must be provided to Apply")
	}
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(placementsResource, *name, types.ApplyPatchType, data), &v1alpha1.Placement{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Placement), err
}

// ApplyStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating ApplyStatus().
func (c *FakePlacements) ApplyStatus(ctx context.Context, placement *schedulingv1alpha1.PlacementApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.Placement, err error) {
	if placement == nil {
		return nil, fmt.Errorf("placement provided to Apply must not be nil")
	}
	data, err := json.Marshal(placement)
	if err != nil {
		return nil, err
	}
	name := placement.Name
	if name == nil {
		return nil, fmt.Errorf("placement.Name must be provided to Apply")
	}
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(placementsResource, *name, types.ApplyPatchType, data, "status"), &v1alpha1.Placement{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Placement), err
}
