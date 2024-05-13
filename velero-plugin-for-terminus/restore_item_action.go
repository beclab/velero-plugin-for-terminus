package main

import (
	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type RestoreItemAction struct {
	log logrus.FieldLogger
}

func newRestoreItemAction(logger logrus.FieldLogger) *RestoreItemAction {
	return &RestoreItemAction{log: logger}
}

func (p *RestoreItemAction) AppliesTo() (velero.ResourceSelector, error) {
	p.log.Info("<plugin> Restore AppliesTo")
	return velero.ResourceSelector{}, nil
}

// Execute allows the RestorePlugin to perform arbitrary logic with the item being restored,
// in this case, setting a custom annotation on the item being restored.
func (p *RestoreItemAction) Execute(input *velero.RestoreItemActionExecuteInput) (*velero.RestoreItemActionExecuteOutput, error) {
	restore := input.Restore

	p.log.Infof("<plugin> Restore Execute bucket: %s", restore.Name)

	inputMap := input.Item.UnstructuredContent()

	// kind := inputMap["kind"]
	// if kind == "ProviderRegistry" {
	// 	inputMap["status"] = map[string]interface{}{"state": "active"}
	// }

	kind := inputMap["kind"]
	if kind == "DistributedRedisCluster" {
		spec := inputMap["spec"].(map[string]interface{})
		spec["init"] = nil
	}

	return velero.NewRestoreItemActionExecuteOutput(&unstructured.Unstructured{Object: inputMap}), nil
}
