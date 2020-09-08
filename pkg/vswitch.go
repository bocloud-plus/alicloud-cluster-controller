package pkg

import (
	v1 "cloudplus.io/alicloud-cluster-controller/api/v1"
	"cloudplus.io/alicloud-cluster-controller/pkg/retry"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
)

func NewVSwitchClient(logger logr.Logger, regionId string) (*VSwitchClient, error) {
	cli, err := vpc.NewClientWithAccessKey(regionId, AccessKeyId, AccessKeySecret)
	if err != nil {
		return nil, errors.Wrap(err, "fail to create vSwitch client")

	}

	return &VSwitchClient{
		Logger: logger.WithValues("client", "vSwitch"),
		cli:    cli,
	}, nil
}

type VSwitchClient struct {
	logr.Logger
	cli *vpc.Client
}

func (c *VSwitchClient) Describe(vSwitchId string) (*vpc.VSwitch, error) {
	logger := c.WithValues("SDKAction", "Describe", "id", vSwitchId)

	req := vpc.CreateDescribeVSwitchesRequest()
	req.Scheme = "https"
	req.VSwitchId = vSwitchId

	var resp *vpc.DescribeVSwitchesResponse
	if err := retry.Try(retry.DefaultBackOff, func() error {
		logger.Info("sending DescribeVSwitch request")
		var err error
		resp, err = c.cli.DescribeVSwitches(req)
		if err != nil {
			logger.Info("error: " + err.Error())
		}

		return errors.Wrap(err, "DescribeVSwitch")
	}); err != nil {
		return nil, err
	}

	logger.Info("success", "TotalCount", resp.TotalCount)
	if resp.TotalCount == 0 {
		return nil, nil
	}
	ret := &resp.VSwitches.VSwitch[0]
	return ret, nil
}
func (c *VSwitchClient) Create(spec *v1.VSwitchSpec) (string, error) {
	logger := c.WithValues("SDKAction", "Create")
	req := spec.ConvertToCreateRequest()

	var resp *vpc.CreateVSwitchResponse

	if err := retry.Try(retry.DefaultBackOff, func() error {
		logger.Info("sending CreateVSwitch request")
		var err error
		resp, err = c.cli.CreateVSwitch(req)
		if err != nil {
			logger.Info("error: " + err.Error() + resp.RequestId)
		}

		if resp.VSwitchId == "" {
			logger.Info("VSwitchId is nil, retry")
			return retry.ErrRetry
		}
		return errors.Wrap(err, "CreateVSwitch")
	}); err != nil {
		return "", err
	}

	return resp.VSwitchId, nil
}

func (c *VSwitchClient) WaitReady(vSwitchId string) (*vpc.VSwitch, error) {

	logger := c.WithValues("SDKAction", "WaitReady")

	var vSwitch *vpc.VSwitch
	if err := retry.Try(retry.DefaultBackOff, func() error {
		logger.Info("describe vpc", "vpcId", vSwitchId)

		var err error
		vSwitch, err = c.Describe(vSwitchId)

		if err != nil {
			logger.Info("error: ", err.Error())
			return errors.Wrap(err, "DescribeVSwitch")
		}
		if vSwitch == nil {
			logger.Info("no vSwitch found")
			return retry.ErrRetry
		}
		if vSwitch.Status != v1.StatusAvailable {
			logger.Info("vSwitch not ready")
			return retry.ErrRetry
		}
		return nil
	}); err != nil {
		return nil, err
	}
	logger.Info("vSwitch " + vSwitchId + " is ready")
	return vSwitch, nil
}

func (c *VSwitchClient) Delete(vSwitchId string) error {
	logger := c.WithValues("SDKAction", "Delete", "id", vSwitchId)
	req := vpc.CreateDeleteVSwitchRequest()
	req.Scheme = "https"
	req.VSwitchId = vSwitchId

	var resp *vpc.DeleteVSwitchResponse

	if err := retry.Try(retry.DefaultBackOff, func() error {
		logger.Info("sending delete vSwitch request")
		var err error
		resp, err = c.cli.DeleteVSwitch(req)
		if err != nil {
			logger.Info("error: " + err.Error())
		}
		return errors.Wrap(err, "DescribeVSwitch")
	}); err != nil {
		return err
	}

	logger.Info("success to delete vSwitch", "requestId", resp.RequestId, "vSwitchId", vSwitchId)
	return nil
}
