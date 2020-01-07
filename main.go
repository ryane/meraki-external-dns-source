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

package main

import (
	"errors"
	"flag"
	"os"
	"time"

	"github.com/kubernetes-incubator/external-dns/endpoint"
	dnsv1alpha1 "github.com/ryane/meraki-external-dns-source/api/v1alpha1"
	"github.com/ryane/meraki-external-dns-source/controllers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = dnsv1alpha1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme

	// register DNSEndpoint from external-dns
	metav1.AddToGroupVersion(scheme, dnsv1alpha1.DNSEndpointGroupVersion)
	scheme.AddKnownTypes(
		dnsv1alpha1.DNSEndpointGroupVersion,
		&endpoint.DNSEndpoint{},
		&endpoint.DNSEndpointList{},
	)
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var throttleInterval time.Duration
	var requeueInterval time.Duration
	var apiKey string
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	flag.DurationVar(&throttleInterval, "throttle-interval", 1*time.Minute, "Attempt to restrict Meraki API calls to only occur once within this interval. There are conditions where this does not apply.")
	flag.DurationVar(&requeueInterval, "requeue-interval", 5*time.Minute, "How long to wait before requeueing Meraki Sources.")
	flag.StringVar(&apiKey, "api-key", "", "The API key for the Meraki API.")
	flag.Parse()

	ctrl.SetLogger(zap.New(func(o *zap.Options) {
		o.Development = true
	}))

	if val, ok := os.LookupEnv("MERAKI_API_KEY"); ok {
		apiKey = val
	}

	if apiKey == "" {
		setupLog.Error(errors.New("A Meraki API key is required"), "unable to start manager")
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		LeaderElection:     enableLeaderElection,
		Port:               9443,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.MerakiSourceReconciler{
		Client:             mgr.GetClient(),
		Log:                ctrl.Log.WithName("controllers").WithName("MerakiSource"),
		Scheme:             mgr.GetScheme(),
		APIKey:             "bf64076d521240fe38175969fbc8b46c6e0af625",
		APIThrottleInteral: throttleInterval,
		RequeueInterval:    requeueInterval,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "MerakiSource")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
