package v1

import (
	"encoding/json"
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/cs"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
	"k8s.io/apimachinery/pkg/util/rand"
	"log"
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

func (s *ClusterSpec) ConvertToCreateRequest() *cs.CreateClusterRequest {
	req := cs.CreateCreateClusterRequest()
	req.Scheme = "https"
	req.Method = "POST"

	req.Domain = "cs.aliyuncs.com"
	//request.Version = "2015-12-15"
	//request.PathPattern = "/clusters"
	req.Headers["Content-Type"] = "application/json"
	req.QueryParams["RegionId"] = s.RegionID

	clusterJson, err := json.Marshal(s)
	if err != nil {
		log.Println("json marshal error: ", err.Error())
	}

	fmt.Println(string(clusterJson))
	fmt.Println(s.WorkerVswitchIds)
	req.Content = clusterJson

	return req
}
