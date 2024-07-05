/*
 * Copyright 1999-2020 Alibaba Group Holding Ltd.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package node

import (
	"context"

	"github.com/zexiplus/chaosblade-spec-go/spec"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"

	"github.com/zexiplus/chaosblade-operator/channel"
	"github.com/zexiplus/chaosblade-operator/exec/model"
	"github.com/zexiplus/chaosblade-operator/pkg/apis/chaosblade/v1alpha1"
)

type ExpController struct {
	model.BaseExperimentController
}

func NewExpController(client *channel.Client) model.ExperimentController {
	return &ExpController{
		model.BaseExperimentController{
			Client:            client,
			ResourceModelSpec: NewResourceModelSpec(client),
		},
	}
}

func (*ExpController) Name() string {
	return "node"
}

func (e *ExpController) Create(ctx context.Context, expSpec v1alpha1.ExperimentSpec) *spec.Response {
	expModel := model.ExtractExpModelFromExperimentSpec(expSpec)
	experimentId := model.GetExperimentIdFromContext(ctx)
	logrusField := logrus.WithField("experiment", experimentId)
	// get nodes
	nodes, resp := e.getMatchedNodeResources(ctx, *expModel)
	if !resp.Success {
		logrusField.Errorf("uid: %s, get matched node resources failed, %v", experimentId, resp.Err)
		resp.Result = v1alpha1.CreateFailExperimentStatus(resp.Err, []v1alpha1.ResourceStatus{})
		return resp
	}
	logrusField.Infof("creating node experiment, node count is %d", len(nodes))
	containerMatchedList := getContainerMatchedList(nodes)
	ctx = model.SetContainerObjectMetaListToContext(ctx, containerMatchedList)
	return e.Exec(ctx, expModel)
}

func (e *ExpController) Destroy(ctx context.Context, expSpec v1alpha1.ExperimentSpec, oldExpStatus v1alpha1.ExperimentStatus) *spec.Response {
	logrus.WithField("experiment", model.GetExperimentIdFromContext(ctx)).Infoln("start to destroy")
	expModel := model.ExtractExpModelFromExperimentSpec(expSpec)
	statuses := oldExpStatus.ResStatuses
	if statuses == nil {
		return spec.ReturnSuccess(v1alpha1.CreateSuccessExperimentStatus([]v1alpha1.ResourceStatus{}))
	}
	containerObjectMetaList := model.ContainerMatchedList{}
	for _, status := range statuses {
		if !status.Success {
			continue
		}
		containerObjectMeta := model.ParseIdentifier(status.Identifier)
		containerObjectMeta.Id = status.Id
		containerObjectMetaList = append(containerObjectMetaList, containerObjectMeta)
	}
	ctx = model.SetContainerObjectMetaListToContext(ctx, containerObjectMetaList)
	return e.Exec(ctx, expModel)
}

// getContainerMatchedList transports selected pods
func getContainerMatchedList(nodes []v1.Node) model.ContainerMatchedList {
	containerObjectMetaList := model.ContainerMatchedList{}
	for _, n := range nodes {
		containerObjectMetaList = append(containerObjectMetaList, model.ContainerObjectMeta{
			NodeName: n.Name,
		})
	}
	return containerObjectMetaList
}
