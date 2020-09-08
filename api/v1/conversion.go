package v1

import (
"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
"k8s.io/apimachinery/pkg/util/rand"
)

func (s *VpcSpec) ConvertToCreateRequest() *vpc.CreateVpcRequest {
	req := vpc.CreateCreateVpcRequest()
	req.Scheme = "https"
	req.ClientToken = rand.String(32)

	req.VpcName = s.VpcName
	req.CidrBlock = s.CidrBlock
	req.Description = s.Description
	return req

}

func (s *VSwitchSpec) ConvertToCreateRequest() *vpc.CreateVSwitchRequest {
	req := vpc.CreateCreateVSwitchRequest()
	req.Scheme = "https"
	req.ClientToken = rand.String(32)

	req.VpcId = s.VpcId
	req.Description = s.Description
	req.CidrBlock = s.CidrBlock
	req.VSwitchName = s.VSwitchName
	req.ZoneId = s.ZoneId

	return req
}

