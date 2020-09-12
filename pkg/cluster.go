package pkg

import (
	v1 "cloudplus.io/alicloud-cluster-controller/api/v1"
	"cloudplus.io/alicloud-cluster-controller/pkg/retry"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/cs"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
)

type ClusterClient struct {
	logr.Logger
	cli *cs.Client
}

func NewClusterClient(logger logr.Logger, regionId string) (*ClusterClient, error) {
	cli, err := cs.NewClientWithAccessKey(regionId, AccessKeyId, AccessKeySecret)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create vpc client")
	}
	return &ClusterClient{
		Logger: logger.WithValues("client", "cluster"),
		cli:    cli,
	}, nil
}

func (c *ClusterClient) Create(spec v1.ClusterSpec) (string, error) {
	logger := c.WithValues("SDKAction", "Create")
	req := spec.ConvertToCreateRequest()
	var resp *cs.CreateClusterResponse
	if err := retry.Try(retry.DefaultBackOff, func() error {
		logger.Info("sending CreateCluster request")
		var err error
		resp, err = c.cli.CreateCluster(req)
		if err != nil {
			logger.Info("error:" + err.Error())
		}

		return errors.Wrap(err, "CreateVpc")
	}); err != nil {
		return "", err
	}
	logger.Info("success to create cluster", "clusterId", resp.ClusterId)
	return resp.ClusterId, nil

}

func (c *ClusterClient) Describe(clusterId string) (*v1.Cluster, error) {
	logger := c.WithValues("SDKAction", "Describe")
	req := cs.CreateDescribeClusterDetailRequest()
	req.Scheme = "https"

	req.ClusterId = clusterId
	var resp *cs.DescribeClusterDetailResponse
	if err := retry.Try(retry.DefaultBackOff, func() error {
		logger.Info("sending DescribeClusterDetail request")
		var err error
		resp, err = c.cli.DescribeClusterDetail(req)
		if err != nil {
			logger.Info("error: " + err.Error())

		}
		return errors.Wrap(err, "DescribeClusterDetail")

	}); err != nil {
		return nil, errors.Wrap(err, "DescribeClusterDetail")
	}
	logger.Info("success", "describe cluster id", clusterId)

	ret := &v1.Cluster{}
	ret.Fill(resp)
	return ret, nil
}

func (c *ClusterClient) Delete(clusterId string) error {
	logger := c.WithValues("SDKAction", "Delete")
	req := cs.CreateDeleteClusterRequest()
	req.Scheme = "http"
	req.ClusterId = clusterId

	var resp *cs.DeleteClusterResponse
	if err := retry.Try(retry.DefaultBackOff, func() error {
		logger.Info("sending delete cluster request")

		var err error
		resp, err = c.cli.DeleteCluster(req)
		if err != nil {
			logger.Info("error: " + err.Error())
		}

		return errors.Wrap(err, "DeleteCluster")

	}); err != nil {
		return err
	}
	logger.Info("success", "deleteCluster request id", resp.RequestId)
	return nil
}

//func (c *ClusterClient) WaitReady()
