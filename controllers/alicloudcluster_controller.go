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
	clusterv1 "cloudplus.io/alicloud-cluster-controller/api/v1"
	"context"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// AlicloudClusterReconciler reconciles a AlicloudCluster object
type AlicloudClusterReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=cluster.cloudplus.io,resources=alicloudclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cluster.cloudplus.io,resources=alicloudclusters/status,verbs=get;update;patch
func (r *AlicloudClusterReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	logger := r.Log.WithValues("alicloudcluster", req.NamespacedName)
	// your logic here

	cluster := &clusterv1.AlicloudCluster{}

	if err := r.Get(ctx, req.NamespacedName, cluster); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	childCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	executor, err := NewExecutor(childCtx, logger, r.Client, cluster.Spec.RegionId, cluster)
	if err != nil {
		logger.Info("new executor error: " + err.Error())
	}
	if !cluster.DeletionTimestamp.IsZero() {
		// handle deleted cluster
		if ret, err := executor.ReconcileDelete(); err != nil {
			logger.Error(err, "ReconcileDelete error")
			return ret, errors.Wrap(err, "ReconcileDelete")
		}

		if err := r.Update(ctx, cluster); err != nil {
			return ctrl.Result{}, errors.Wrap(err, "Update")
		}

		return ctrl.Result{}, nil

	}

	// handle non-deleted cluster
	if ret, err := executor.ReconcileNormal(); err != nil {
		logger.Error(err, "ReconcileNormal error")
		return ret, errors.Wrap(err, "ReconcileNormal error")
	}

	if err := r.Update(ctx, cluster); err != nil {
		return ctrl.Result{}, errors.Wrap(err, "Update")
	}

	return ctrl.Result{}, nil

	//executor, err := NewExecutor()

	//log.Info("print")
	//fmt.Println(cluster.Spec.Cluster)
	//
	//jsonBytes, err := json.Marshal(cluster.Spec.Cluster)
	//
	//if err != nil {
	//	fmt.Println("json marshal error: ", err.Error())
	//
	//}
	//fmt.Println(string(jsonBytes))
	//return ctrl.Result{}, nil
}

func (r *AlicloudClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clusterv1.AlicloudCluster{}).
		Complete(r)
}
