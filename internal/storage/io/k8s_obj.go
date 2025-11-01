package io

import (
	"context"
	"fmt"
	"io"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"

	"github.com/slok/sloth/internal/log"
	"github.com/slok/sloth/pkg/common/model"
	plugink8stransformv1 "github.com/slok/sloth/pkg/prometheus/plugin/k8stransform/v1"
)

func NewIOWriterK8sObjectYAMLRepo(writer io.Writer, transformer plugink8stransformv1.Plugin, logger log.Logger) IOWriterK8sObjectYAMLRepo {
	return IOWriterK8sObjectYAMLRepo{
		writer:      writer,
		encoder:     json.NewYAMLSerializer(json.DefaultMetaFactory, nil, nil),
		logger:      logger.WithValues(log.Kv{"svc": "storage.io.IOWriterK8sObjectYAMLRepo"}),
		transformer: transformer,
	}
}

// IOWriterK8sObjectYAMLRepo knows to store all the SLO rules (recordings and alerts)
// grouped in an IOWriter in Kubernetes K8sObject YAML format.
type IOWriterK8sObjectYAMLRepo struct {
	writer      io.Writer
	encoder     runtime.Encoder
	logger      log.Logger
	transformer plugink8stransformv1.Plugin
}

func (i IOWriterK8sObjectYAMLRepo) StoreSLOs(ctx context.Context, kmeta model.K8sMeta, slos model.PromSLOGroupResult) error {
	k8sObjs, err := i.transformer.TransformK8sObjects(ctx, kmeta, slos)
	if err != nil {
		return fmt.Errorf("could not transform k8s objects using plugin: %w", err)
	}

	for _, obj := range k8sObjs.Items {
		obj := obj.DeepCopy()

		l := obj.GetLabels()
		if l == nil {
			l = map[string]string{}
		}
		l["app.kubernetes.io/component"] = "SLO"
		l["app.kubernetes.io/managed-by"] = "sloth"
		obj.SetLabels(l)

		_, err := i.writer.Write([]byte(yamlTopdisclaimer))
		if err != nil {
			return fmt.Errorf("could not write top disclaimer: %w", err)
		}
		err = i.encoder.Encode(obj, i.writer)
		if err != nil {
			return fmt.Errorf("could encode k8s object: %w", err)
		}
	}

	return nil
}
