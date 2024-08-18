/*
Copyright 2024. projectsveltos.io. All rights reserved.

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

package fv_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	libsveltosv1beta1 "github.com/projectsveltos/libsveltos/api/v1beta1"
)

var _ = Describe("Start watcher", Label("FV"), func() {
	const (
		namePrefix = "ignore-"
	)

	It("Mark ResourceSummary for reconciliation using annotation", func() {
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namePrefix + randomString(),
			},
		}

		By(fmt.Sprintf("Create namespace %s", namespace.Name))
		Expect(k8sClient.Create(context.TODO(), namespace))

		By(fmt.Sprintf("Create resourceSummary referencing namespace %s", namespace.Name))
		By("namespace is marked with IgnoreForConfigurationDrift")
		Expect(addTypeInformationToObject(scheme, namespace)).To(Succeed())
		resourceRef := corev1.ObjectReference{
			Name:       namespace.Name,
			Kind:       namespace.Kind,
			APIVersion: namespace.APIVersion,
		}
		resourceSummary := getResourceSummary(&resourceRef, nil, true)
		Expect(k8sClient.Create(context.TODO(), resourceSummary)).To(Succeed())

		By("Modify namespace")
		currentNamespace := &corev1.Namespace{}
		Expect(k8sClient.Get(context.TODO(),
			types.NamespacedName{Name: namespace.Name}, currentNamespace)).To(Succeed())
		currentNamespace.Labels = map[string]string{randomString(): randomString()}
		Expect(k8sClient.Update(context.TODO(), currentNamespace)).To(Succeed())

		By(fmt.Sprintf("Verify ResourceSummary %s is NOT marked for reconciliation", resourceSummary.Name))
		Consistently(func() bool {
			currentResourceSummary := &libsveltosv1beta1.ResourceSummary{}
			err := k8sClient.Get(context.TODO(),
				types.NamespacedName{Namespace: resourceSummary.Namespace, Name: resourceSummary.Name},
				currentResourceSummary)
			return err == nil && !currentResourceSummary.Status.ResourcesChanged
		}, timeout/2, pollingInterval).Should(BeTrue())

		By(fmt.Sprintf("Delete namespace %s", namespace.Name))
		Expect(k8sClient.Get(context.TODO(),
			types.NamespacedName{Name: namespace.Name}, currentNamespace)).To(Succeed())
		Expect(k8sClient.Delete(context.TODO(), currentNamespace)).To(Succeed())

		By(fmt.Sprintf("Verify ResourceSummary %s is NOT marked for reconciliation", resourceSummary.Name))
		Consistently(func() bool {
			currentResourceSummary := &libsveltosv1beta1.ResourceSummary{}
			err := k8sClient.Get(context.TODO(),
				types.NamespacedName{Namespace: resourceSummary.Namespace, Name: resourceSummary.Name},
				currentResourceSummary)
			return err == nil && !currentResourceSummary.Status.ResourcesChanged
		}, timeout/2, pollingInterval).Should(BeTrue())

		By("Delete ResourceSummary")
		currentResourceSummary := &libsveltosv1beta1.ResourceSummary{}
		Expect(k8sClient.Get(context.TODO(),
			types.NamespacedName{Namespace: resourceSummary.Namespace, Name: resourceSummary.Name},
			currentResourceSummary)).To(Succeed())
		Expect(k8sClient.Delete(context.TODO(), currentResourceSummary)).To(Succeed())

		By("Verify ResourceSummary is gone")
		Eventually(func() bool {
			err := k8sClient.Get(context.TODO(),
				types.NamespacedName{Namespace: resourceSummary.Namespace, Name: resourceSummary.Name},
				currentResourceSummary)
			return err != nil && apierrors.IsNotFound(err)
		}, timeout, pollingInterval).Should(BeTrue())
	})
})
