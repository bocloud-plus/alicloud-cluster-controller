package controllers

import (
	v1 "cloudplus.io/alicloud-cluster-controller/api/v1"
	"cloudplus.io/alicloud-cluster-controller/pkg"
	"cloudplus.io/alicloud-cluster-controller/pkg/retry"
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewExecutor(
	ctx context.Context,
	logger logr.Logger,
	client client.Client,
	regionId string,
	cluster *v1.AlicloudCluster,
) (*Executor, error) {

	vpcCli, err := pkg.NewVpcClient(logger, regionId)
	if err != nil {
		return nil, err
	}
	vswitchCli, err := pkg.NewVSwitchClient(logger, regionId)
	if err != nil {
		return nil, err
	}
	clusterCli, err := pkg.NewClusterClient(logger, regionId)
	if err != nil {
		return nil, err
	}

	return &Executor{
		vswitchCli: vswitchCli,
		vpcCli:     vpcCli,
		clusterCli: clusterCli,
		cluster:    cluster,
		Client:     client,
		Logger:     logger,
		ctx:        ctx,
	}, nil
}

type Executor struct {
	vswitchCli *pkg.VSwitchClient
	vpcCli     *pkg.VpcClient
	clusterCli *pkg.ClusterClient
	cluster    *v1.AlicloudCluster
	client.Client
	logr.Logger
	ctx context.Context
}

func (e *Executor) ReconcileNormal() (ctrl.Result, error) {

	if !Contains(e.cluster.Finalizers, v1.Finalizer) {
		e.cluster.Finalizers = append(e.cluster.Finalizers, v1.Finalizer)
	}

	if rs, err := e.ReconcileVpc(); err != nil {
		return rs, err
	}
	if rs, err := e.ReconcileVSwitch(); err != nil {
		return rs, err
	}
	if rs, err := e.ReconcileCluster(); err != nil {
		return rs, err
	}
	return ctrl.Result{}, nil

}

func (e *Executor) ReconcileVpc() (ctrl.Result, error) {
	if len(e.cluster.Status.Vpc.VpcId) > 0 {
		return ctrl.Result{}, nil
	}
	spec := e.cluster.Spec
	vpcId := spec.Vpc.VpcId

	var err error

	var target *v1.Vpc
	if len(vpcId) > 0 {
		target, err = e.vpcCli.Describe(vpcId)
		if err != nil {
			return ctrl.Result{}, errors.Wrap(err, "Describe "+vpcId)
		}
		if target == nil {
			return ctrl.Result{}, errors.Errorf("target not found")
		}
		if target.Status != v1.StatusAvailable {
			target, err = e.vpcCli.WaitReady(vpcId)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// Create
		id, err := e.vpcCli.Create(e.cluster.Spec.Vpc)
		if err != nil {
			return ctrl.Result{}, errors.Wrap(err, "Create error")
		}
		target, err = e.vpcCli.Describe(id)
		if err != nil {
			return ctrl.Result{}, errors.Wrap(err, "Describe")
		}

		if target.Status != v1.StatusAvailable {
			target, err = e.vpcCli.WaitReady(id)
			if err != nil {
				return ctrl.Result{}, errors.Wrap(err, "WaitReady")
			}
		}
	}
	e.Info("reconcileVpc success")
	target.DeepCopyInto(&e.cluster.Status.Vpc)

	e.cluster.Spec.Cluster.Vpcid = target.VpcId
	if err := e.Update(e.ctx, e.cluster); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil

}

func (e *Executor) ReconcileVSwitch() (ctrl.Result, error) {
	if len(e.cluster.Status.VSwitch.VSwitchId) > 0 {
		return ctrl.Result{}, nil
	}
	spec := e.cluster.Spec

	vswId := spec.VSwitch.VSwitchId

	var err error
	var target *v1.VSwitch
	if len(vswId) > 0 {
		target, err = e.vswitchCli.Describe(vswId)
		if err != nil {
			return ctrl.Result{}, errors.Wrap(err, "Describe "+vswId)
		}
		if target == nil {
			return ctrl.Result{}, errors.Errorf("target not found")
		}
		if target.Status != v1.StatusAvailable {
			target, err = e.vswitchCli.WaitReady(vswId)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// Create
		e.cluster.Spec.VSwitch.VpcId = e.cluster.Status.Vpc.VpcId
		id, err := e.vswitchCli.Create(e.cluster.Spec.VSwitch)
		if err != nil {
			return ctrl.Result{}, errors.Wrap(err, "Create error")
		}
		target, err = e.vswitchCli.Describe(id)
		if err != nil {
			return ctrl.Result{}, errors.Wrap(err, "Describe")
		}

		if target.Status != v1.StatusAvailable {
			target, err = e.vswitchCli.WaitReady(id)
			if err != nil {
				return ctrl.Result{}, errors.Wrap(err, "WaitReady")
			}
		}
	}
	e.Info("reconcileVSwitch success")
	target.DeepCopyInto(&e.cluster.Status.VSwitch)

	for i := 0; i < e.cluster.Spec.Cluster.MasterCount; i++ {
		e.cluster.Spec.Cluster.MasterVswitchIds =
			append(e.cluster.Spec.Cluster.MasterVswitchIds, target.VSwitchId)
	}

	for i := 0; i < e.cluster.Spec.Cluster.NumOfNodes; i++ {
		e.cluster.Spec.Cluster.WorkerVswitchIds =
			append(e.cluster.Spec.Cluster.WorkerVswitchIds, target.VSwitchId)
	}

	fmt.Println(e.cluster.Spec.Cluster.WorkerVswitchIds)
	if err := e.Update(e.ctx, e.cluster); err != nil {
		return ctrl.Result{}, errors.Wrap(err, "Update")
	}

	return ctrl.Result{}, nil
}

func (e *Executor) ReconcileCluster() (ctrl.Result, error) {
	e.Logger.Info("start to reconcile cluster")

	switch e.cluster.Status.Phase {
	case v1.PhasePending:
		clusterId, err := e.clusterCli.Create(e.cluster.Spec.Cluster)
		if err != nil {
			return ctrl.Result{}, err
		}
		e.cluster.Status.ClusterId = clusterId
		e.cluster.Status.Phase = v1.PhaseCreating

		if err := e.Update(e.ctx, e.cluster); err != nil {
			return ctrl.Result{}, err
		}
	case v1.PhaseCreating:
		clusterId := e.cluster.Status.ClusterId
		cluster, err := e.clusterCli.Describe(clusterId)
		if err != nil {
			return ctrl.Result{}, err
		}
		if cluster.State == "running" {
			e.cluster.Status.Phase = v1.PhaseRunning
		}

		if err := e.Update(e.ctx, e.cluster); err != nil {
			return ctrl.Result{}, err
		}
	case v1.PhaseRunning:
		//
	case v1.PhaseScalingOut:
		//
	case v1.PhaseRemovingNode:
		//
	case v1.PhaseDeleting:

	}

	return ctrl.Result{}, nil

}

func (e *Executor) ReconcileDelete() (ctrl.Result, error) {
	e.Logger.Info("ReconcileDelete")

	//if rs, err := e.DeleteCluster(); err != nil {
	//	return rs, errors.Wrap(err, "DeleteCluster")
	//}

	if rs, err := e.DeleteVSwitch(); err != nil {
		return rs, errors.Wrap(err, "deleteVSwitch")

	}

	if rs, err := e.DeleteVpc(); err != nil {
		return rs, errors.Wrap(err, "deleteVpc")
	}

	e.cluster.Finalizers = Filter(e.cluster.Finalizers, v1.Finalizer)
	return ctrl.Result{}, nil

}

func (e *Executor) DeleteVSwitch() (ctrl.Result, error) {
	vswId := e.cluster.Status.VSwitch.VSwitchId

	if len(vswId) == 0 {
		return ctrl.Result{}, nil
	}
	target, err := e.vswitchCli.Describe(vswId)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "deleteVSwitch Describe")
	}
	if target == nil {
		return ctrl.Result{}, nil
	}
	err = e.vswitchCli.Delete(vswId)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "DeleteVSwitch")
	}
	return ctrl.Result{}, retry.Try(retry.DefaultBackOff, func() error {
		target, err := e.vswitchCli.Describe(vswId)
		if err != nil {
			return err
		}
		if target != nil {
			return retry.ErrRetry
		}
		return nil
	})

}

func (e *Executor) DeleteVpc() (ctrl.Result, error) {
	vpcId := e.cluster.Status.Vpc.VpcId

	if len(vpcId) == 0 {
		return ctrl.Result{}, nil
	}
	target, err := e.vpcCli.Describe(vpcId)
	if err != nil {
		return ctrl.Result{}, err
	}
	if target == nil {
		return ctrl.Result{}, nil
	}
	err = e.vpcCli.Delete(vpcId)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "DeleteVpc")
	}

	return ctrl.Result{}, retry.Try(retry.DefaultBackOff, func() error {
		target, err := e.vpcCli.Describe(vpcId)
		if err != nil {
			return err
		}

		if target != nil {
			return retry.ErrRetry
		}

		return nil
	})
}

//func DeletCluster()
func Contains(list []string, target string) bool {
	for _, str := range list {
		if str == target {
			return true
		}
	}

	return false

}

func Filter(list []string, target string) (newList []string) {

	for _, str := range list {
		if str == target {
			continue
		}
		newList = append(newList, str)
	}

	return
}
