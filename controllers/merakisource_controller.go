/*

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

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/kubernetes-incubator/external-dns/endpoint"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ref "k8s.io/client-go/tools/reference"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dnsv1alpha1 "github.com/ryane/meraki-external-dns-source/api/v1alpha1"
)

// MerakiSourceReconciler reconciles a MerakiSource object
type MerakiSourceReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=dns.jossware.com,resources=merakisources,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=dns.jossware.com,resources=merakisources/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=externaldns.k8s.io,resources=dnsendpoints,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=externaldns.k8s.io,resources=dnsendpoints/status,verbs=get

func (r *MerakiSourceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("merakisource", req.NamespacedName)

	var source dnsv1alpha1.MerakiSource
	if err := r.Get(ctx, req.NamespacedName, &source); err != nil {
		if apierrs.IsNotFound(err) {
			// 404, wait for next notification
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch MerakiSource")
		return ctrl.Result{}, err
	}

	// query meraki

	// check if owned DNSEndpoints exist
	var dnsEndpointList endpoint.DNSEndpointList
	if err := r.List(ctx, &dnsEndpointList, client.InNamespace(req.Namespace), client.MatchingFields{jobOwnerKey: req.Name}); err != nil {
		return ctrl.Result{}, nil
	}
	log.V(1).Info("endpoints", "count", len(dnsEndpointList.Items))

	if len(dnsEndpointList.Items) == 0 {
		// if not, create one and requeue
		e := &endpoint.DNSEndpoint{
			ObjectMeta: metav1.ObjectMeta{
				Name:      source.Name,
				Namespace: source.Namespace,
			},
		}
		if err := ctrl.SetControllerReference(&source, e, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.Create(ctx, e); err != nil {
			log.Error(err, "unable to create dns endpoint", "dnsEndpoint", e)
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	source.Status.Endpoints = []corev1.ObjectReference{}
	for _, e := range dnsEndpointList.Items {
		ref, err := ref.GetReference(r.Scheme, &e)
		if err != nil {
			log.Error(err, "unable to make reference to dns endpoint", "dnsEndpoint", e)
			continue
		}
		source.Status.Endpoints = append(source.Status.Endpoints, *ref)
	}
	if err := r.Status().Update(ctx, &source); err != nil {
		log.Error(err, "unable to update MerakiSource status")
		return ctrl.Result{}, err
	}

	// update the spec from MerakiData

	return ctrl.Result{}, nil
}

var (
	jobOwnerKey = ".metadata.controller"
)

func (r *MerakiSourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(&endpoint.DNSEndpoint{}, jobOwnerKey, func(rawObj runtime.Object) []string {
		r.Log.V(1).Info("index field", "jobOwnerKey", jobOwnerKey, "rawObj", rawObj)
		e := rawObj.(*endpoint.DNSEndpoint)
		owner := metav1.GetControllerOf(e)
		r.Log.V(1).Info("index field", "endpoint", e.Name, "owner", owner)
		if owner == nil {
			return nil
		}
		if owner.APIVersion != dnsv1alpha1.GroupVersion.String() || owner.Kind != "MerakiSource" {
			return nil
		}
		return []string{owner.Name}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&dnsv1alpha1.MerakiSource{}).
		Owns(&endpoint.DNSEndpoint{}).
		Complete(r)
}
