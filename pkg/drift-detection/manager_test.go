/*
Copyright 2022.

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

package driftdetection_test

import (
	"context"
	"reflect"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2/textlogger"

	driftdetection "github.com/projectsveltos/drift-detection-manager/pkg/drift-detection"
	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
)

var _ = Describe("Manager: registration", func() {
	var watcherCtx context.Context
	var resource corev1.Namespace
	var logger logr.Logger

	BeforeEach(func() {
		logger = textlogger.NewLogger(textlogger.NewConfig(textlogger.Verbosity(1)))

		driftdetection.Reset()
		watcherCtx, cancel = context.WithCancel(context.Background())

		resource = corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: randomString(),
			},
		}
	})

	AfterEach(func() {
		cancel()
	})

	It("RegisterResource: start tracking a resource. UnRegisterResource stop tracking a resource", func() {
		Expect(testEnv.Create(watcherCtx, &resource)).To(Succeed())
		Expect(waitForObject(watcherCtx, testEnv.Client, &resource)).To(Succeed())

		Expect(addTypeInformationToObject(scheme, &resource)).To(Succeed())

		Expect(driftdetection.InitializeManager(watcherCtx, logger, testEnv.Config, testEnv.Client, scheme,
			randomString(), randomString(), randomString(), libsveltosv1beta1.ClusterTypeCapi, evaluateTimeout,
			false)).To(Succeed())
		manager, err := driftdetection.GetManager()
		Expect(err).To(BeNil())

		resourceRef := corev1.ObjectReference{
			Name:       resource.Name,
			Kind:       resource.Kind,
			APIVersion: resource.APIVersion,
		}

		resourceSummary := getResourceSummary(&resourceRef, nil, nil)
		hash, err := manager.RegisterResource(watcherCtx, &resourceRef, driftdetection.KustomizeResource, resourceSummary)
		Expect(err).To(BeNil())

		resources := manager.GetKustomizeResources()
		Expect(len(resources)).To(Equal(1))
		consumers := resources[resourceRef]
		Expect(consumers.Len()).To(Equal(1))

		resources = manager.GetHelmResources()
		Expect(len(resources)).To(Equal(0))

		resources = manager.GetResources()
		Expect(len(resources)).To(Equal(0))

		hashes := manager.GetResourceHashes()
		Expect(len(hashes)).To(Equal(1))
		currentHash, ok := hashes[resourceRef]
		Expect(ok).To(BeTrue())
		Expect(reflect.DeepEqual(currentHash, hash)).To(BeTrue())

		watchers := manager.GetWatchers()
		Expect(len(watchers)).To(Equal(1))
		gvk := schema.GroupVersionKind{
			Group:   resource.GroupVersionKind().Group,
			Version: resource.GroupVersionKind().Version,
			Kind:    resource.Kind,
		}
		_, ok = watchers[gvk]
		Expect(ok).To(BeTrue())

		gvks := manager.GetGVKResources()
		Expect(len(gvks)).To(Equal(1))
		gvkResources := gvks[resourceRef.GroupVersionKind()]
		Expect(gvkResources.Len()).To(Equal(1))
		Expect(gvkResources.Items()).To(ContainElement(resourceRef))

		Expect(manager.UnRegisterResource(&resourceRef, driftdetection.KustomizeResource, resourceSummary)).To(Succeed())
		resources = manager.GetKustomizeResources()
		Expect(len(resources)).To(Equal(0))
		resources = manager.GetResources()
		Expect(len(resources)).To(Equal(0))
		resources = manager.GetHelmResources()
		Expect(len(resources)).To(Equal(0))
		hashes = manager.GetResourceHashes()
		Expect(len(hashes)).To(Equal(0))
		watchers = manager.GetWatchers()
		Expect(len(watchers)).To(Equal(0))
		gvks = manager.GetGVKResources()
		Expect(len(gvks)).To(Equal(0))
	})

	It("readResourceSummaries processes all existing ResourceSummaries", func() {
		Expect(driftdetection.InitializeManager(watcherCtx, logger, testEnv.Config, testEnv.Client, scheme,
			randomString(), randomString(), randomString(), libsveltosv1beta1.ClusterTypeCapi, evaluateTimeout,
			false)).To(Succeed())
		manager, err := driftdetection.GetManager()
		Expect(err).To(BeNil())

		resource1 := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: randomString(),
			},
		}
		Expect(addTypeInformationToObject(scheme, &resource1)).To(Succeed())

		resourceRef1 := corev1.ObjectReference{
			Name:       resource1.Name,
			Kind:       resource1.Kind,
			APIVersion: resource1.APIVersion,
		}

		resource2 := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: randomString(),
			},
		}
		Expect(addTypeInformationToObject(scheme, &resource2)).To(Succeed())
		resourceRef2 := corev1.ObjectReference{
			Name:       resource2.Name,
			Kind:       resource2.Kind,
			APIVersion: resource2.APIVersion,
		}

		resource3 := corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: randomString(),
			},
		}
		Expect(addTypeInformationToObject(scheme, &resource3)).To(Succeed())
		resourceRef3 := corev1.ObjectReference{
			Name:       resource3.Name,
			Kind:       resource3.Kind,
			APIVersion: resource3.APIVersion,
		}

		resourceSummary := getResourceSummary(&resourceRef1, &resourceRef2, &resourceRef3)

		resourceSummaryNs := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: resourceSummary.Namespace,
			},
		}
		Expect(testEnv.Create(watcherCtx, resourceSummaryNs)).To(Succeed())
		Expect(waitForObject(watcherCtx, testEnv.Client, resourceSummaryNs)).To(Succeed())

		Expect(testEnv.Create(watcherCtx, resourceSummary)).To(Succeed())
		Expect(waitForObject(watcherCtx, testEnv.Client, resourceSummary)).To(Succeed())

		// Update ResourceSummary status
		currentResourceSummary := &libsveltosv1beta1.ResourceSummary{}
		Expect(testEnv.Get(watcherCtx,
			types.NamespacedName{Namespace: resourceSummary.Namespace, Name: resourceSummary.Name},
			currentResourceSummary)).To(Succeed())
		hash := randomString()
		currentResourceSummary.Status.ResourceHashes = []libsveltosv1beta1.ResourceHash{
			{
				Hash: hash,
				Resource: libsveltosv1beta1.Resource{
					Kind:      resource1.Kind,
					Group:     resource1.GroupVersionKind().Group,
					Version:   resource1.GroupVersionKind().Version,
					Name:      resource1.Name,
					Namespace: resource1.Namespace,
				},
			},
		}
		currentResourceSummary.Status.KustomizeResourceHashes = []libsveltosv1beta1.ResourceHash{
			{
				Hash: hash,
				Resource: libsveltosv1beta1.Resource{
					Kind:      resource2.Kind,
					Group:     resource2.GroupVersionKind().Group,
					Version:   resource2.GroupVersionKind().Version,
					Name:      resource2.Name,
					Namespace: resource2.Namespace,
				},
			},
		}
		currentResourceSummary.Status.HelmResourceHashes = []libsveltosv1beta1.ResourceHash{
			{
				Hash: hash,
				Resource: libsveltosv1beta1.Resource{
					Kind:      resource3.Kind,
					Group:     resource3.GroupVersionKind().Group,
					Version:   resource3.GroupVersionKind().Version,
					Name:      resource3.Name,
					Namespace: resource3.Namespace,
				},
			},
		}
		Expect(testEnv.Status().Update(watcherCtx, currentResourceSummary)).To(Succeed())

		// wait for cache to sync
		Eventually(func() bool {
			err := testEnv.Get(watcherCtx,
				types.NamespacedName{Namespace: resourceSummary.Namespace, Name: resourceSummary.Name},
				currentResourceSummary)
			return err == nil && currentResourceSummary.Status.ResourceHashes != nil &&
				currentResourceSummary.Status.HelmResourceHashes != nil &&
				currentResourceSummary.Status.KustomizeResourceHashes != nil
		}, timeout, pollingInterval).Should(BeTrue())

		Expect(driftdetection.ReadResourceSummaries(manager, watcherCtx)).To(Succeed())

		Expect(len(manager.GetResources())).To(Equal(1))
		Expect(len(manager.GetKustomizeResources())).To(Equal(1))
		Expect(len(manager.GetHelmResources())).To(Equal(1))
		// ResourceSummary Status was set with a random hash for resource.
		// Resource has not been created by this test. So manager needs to detect that condition
		// (resource missing) as potential configuration drift which needs evaluation.
		queue := manager.GetJobQueue()
		Expect(queue.Has(&resourceRef1)).To(BeTrue())
		Expect(queue.Has(&resourceRef2)).To(BeTrue())
		Expect(queue.Has(&resourceRef3)).To(BeTrue())
	})
})

func getObjRefFromResourceSummary(resourceSummary *libsveltosv1beta1.ResourceSummary) *corev1.ObjectReference {
	gvk := schema.GroupVersionKind{
		Group:   libsveltosv1beta1.GroupVersion.Group,
		Version: libsveltosv1beta1.GroupVersion.Version,
		Kind:    libsveltosv1beta1.ResourceSummaryKind,
	}

	apiVersion, kind := gvk.ToAPIVersionAndKind()

	return &corev1.ObjectReference{
		Namespace:  resourceSummary.Namespace,
		Name:       resourceSummary.Name,
		APIVersion: apiVersion,
		Kind:       kind,
	}
}
