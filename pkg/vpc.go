package pkg

import (
	v1 "cloudplus.io/alicloud-cluster-controller/api/v1"
	"cloudplus.io/alicloud-cluster-controller/pkg/retry"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
)

func NewVpcClient(logger logr.Logger, regionId string) (*VpcClient, error) {
	cli, err := vpc.NewClientWithAccessKey(regionId, AccessKeyId, AccessKeySecret)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create vpc client")
	}
	return &VpcClient{
		Logger: logger.WithValues("client", "vpc"),
		cli:    cli,
	}, nil
}

type VpcClient struct {
	logr.Logger
	cli *vpc.Client
}

func (c *VpcClient) Describe(vpcId string) (*vpc.Vpc, error) {
	logger := c.WithValues("SDKAction", "Describe", "id", vpcId)
	req := vpc.CreateDescribeVpcsRequest()
	req.Scheme = "https"

	req.VpcId = vpcId

	var resp *vpc.DescribeVpcsResponse
	if err := retry.Try(retry.DefaultBackOff, func() error {
		logger.Info("sending DescribeVpc request")
		var err error
		resp, err = c.cli.DescribeVpcs(req)
		if err != nil {
			logger.Info("error: " + err.Error())
		}
		return errors.Wrap(err, "DescribeVpc")
	}); err != nil {
		return nil, err
	}

	logger.Info("success", "TotalCount", resp.TotalCount)
	if resp.TotalCount == 0 {
		return nil, nil
	}
	ret := &resp.Vpcs.Vpc[0]
	return ret, nil
}

func (c *VpcClient) Create(spec *v1.VpcSpec) (string, error) {
	logger := c.WithValues("SDKAction", "Create")
	req := spec.ConvertToCreateRequest()

	var resp *vpc.CreateVpcResponse
	if err := retry.Try(retry.DefaultBackOff, func() error {
		logger.Info("sending CreateVpc request")
		var err error
		resp, err = c.cli.CreateVpc(req)
		if err != nil {
			logger.Info("error: " + err.Error())
		}
		return errors.Wrap(err, "CreateVpc")
	}); err != nil {
		return "", err
	}

	logger.Info("success to create vpc", "vpcId", resp.VpcId)
	return resp.VpcId, nil
}

func (c *VpcClient) WaitReady(vpcId string) (*vpc.Vpc, error) {

	logger := c.WithValues("SDKAction", "WaitReady")

	var vpc *vpc.Vpc
	if err := retry.Try(retry.DefaultBackOff, func() error {
		logger.Info("describe vpc", "vpcId", vpcId)

		var err error
		vpc, err = c.Describe(vpcId)

		if err != nil {
			logger.Info("error: ", err.Error())
			return errors.Wrap(err, "Describe vpc")
		}
		if vpc == nil {
			logger.Info("no vpc found")
			return retry.ErrRetry
		}
		if vpc.Status != v1.StatusAvailable {
			logger.Info("vpc not ready")
			return retry.ErrRetry
		}
		return nil
	}); err != nil {
		return nil, err
	}
	logger.Info("vpc " + vpcId + " is ready")
	return vpc, nil
}

func (c *VpcClient) Delete(vpcId string) error {
	logger := c.WithValues("SDKAction", "Delete", "id", vpcId)
	req := vpc.CreateDeleteVpcRequest()
	req.Scheme = "https"
	req.VpcId = vpcId

	var resp *vpc.DeleteVpcResponse
	if err := retry.Try(retry.DefaultBackOff, func() error {
		logger.Info("sending delete vpc request")
		var err error
		resp, err = c.cli.DeleteVpc(req)
		if err != nil {
			logger.Info("error: " + err.Error())
		}
		return errors.Wrap(err, "DescribeVpc")
	}); err != nil {
		return err
	}

	logger.Info("success to delete vpc", "requestId", resp.RequestId, "vpcId", vpcId)
	return nil
}
