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
	"errors"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	"time"

	demov1 "github.com/mortent/stsbug/api/v1"
)

// StsBugReconciler reconciles a StsBug object
type StsBugReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=demo.mortent.no,resources=stsbugs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=demo.mortent.no,resources=stsbugs/status,verbs=get;update;patch

func (r *StsBugReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	_ = r.Log.WithValues("stsbug", req.NamespacedName)
	ts := strconv.Itoa(int(time.Now().Unix()))

	var stsBug demov1.StsBug
	if err := r.Get(ctx, req.NamespacedName, &stsBug); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var stsList appsv1.StatefulSetList
	if err := r.List(ctx, &stsList, client.MatchingFields{stsOwnerKey: req.Name}); err != nil {
		return ctrl.Result{}, err
	}

	if len(stsList.Items) > 1 {
		return ctrl.Result{}, errors.New("found more than one statefulset")
	}

	if len(stsList.Items) == 1 {
		sts := stsList.Items[0]
		sts.Spec.Template.Spec.PriorityClassName = "NotExisting" + ts
		err := r.Update(ctx, &sts)
		return ctrl.Result{}, err
	}

	if len(stsList.Items) == 0 {
		sts := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      req.Name,
				Namespace: req.Namespace,
			},
			Spec: appsv1.StatefulSetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"pod": "mypod",
					},
				},
				Template: v1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"pod": "mypod",
						},
					},
					Spec: v1.PodSpec{
						PriorityClassName: "NotExisting" + ts,
						Containers: []v1.Container{
							{
								Name:  "nginx",
								Image: "k8s.gcr.io/gninx-slim:0.8",
							},
						},
					},
				},
			},
		}
		if err := ctrl.SetControllerReference(&stsBug, sts, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}
		err := r.Create(ctx, sts)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

var (
	apiGVStr    = demov1.GroupVersion.String()
	stsOwnerKey = ".metadata.controller"
)

func (r *StsBugReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(&appsv1.StatefulSet{}, stsOwnerKey, func(rawObj runtime.Object) []string {
		sts := rawObj.(*appsv1.StatefulSet)
		owner := metav1.GetControllerOf(sts)
		if owner == nil {
			return nil
		}
		if owner.APIVersion != apiGVStr || owner.Kind != "StsBug" {
			return nil
		}
		return []string{owner.Name}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&demov1.StsBug{}).
		Owns(&appsv1.StatefulSet{}).
		Complete(r)
}
