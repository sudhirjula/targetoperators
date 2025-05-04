/*
Copyright 2025.

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

package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	opsv1 "github.com/sudhir/pod-mover-operator/api/v1"
)

// PodMoverReconciler reconciles a PodMover object
type PodMoverReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=ops.example.com,resources=podmovers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ops.example.com,resources=podmovers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ops.example.com,resources=podmovers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the PodMover object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.4/pkg/reconcile
func (r *PodMoverReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the PodMover resource
	var podMover opsv1.PodMover
	if err := r.Get(ctx, req.NamespacedName, &podMover); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// List pods in the source namespace based on the selector
	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(podMover.Spec.SourceNamespace),
		client.MatchingLabels(map[string]string{"selector": podMover.Spec.PodSelector}),
	}
	if err := r.List(ctx, podList, listOpts...); err != nil {
		logger.Error(err, "Failed to list pods")
		return ctrl.Result{}, err
	}

	// Move pods to the target namespace
	for _, pod := range podList.Items {
		newPod := pod.DeepCopy()
		newPod.Namespace = podMover.Spec.TargetNamespace
		newPod.ResourceVersion = ""

		if err := r.Create(ctx, newPod); err != nil {
			logger.Error(err, "Failed to create pod in target namespace")
			return ctrl.Result{}, err
		}
		if err := r.Delete(ctx, &pod); err != nil {
			logger.Error(err, "Failed to delete pod in source namespace")
			return ctrl.Result{}, err
		}
	}

	// Update status
	podMover.Status.MovedPodCount = len(podList.Items)
	if err := r.Status().Update(ctx, &podMover); err != nil {
		logger.Error(err, "Failed to update PodMover status")
		return ctrl.Result{}, err
	}

	logger.Info("Successfully moved pods after updating from app to selector and today is sunday sudhir", "count", len(podList.Items))
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodMoverReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&opsv1.PodMover{}).
		Named("podmover").
		Complete(r)
}
