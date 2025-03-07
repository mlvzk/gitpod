// Copyright (c) 2022 Gitpod GmbH. All rights reserved.
// Licensed under the MIT License. See License-MIT.txt in the project root for license information.

package public_api_server

import (
	"fmt"

	"github.com/gitpod-io/gitpod/common-go/baseserver"

	"github.com/gitpod-io/gitpod/installer/pkg/cluster"
	"github.com/gitpod-io/gitpod/installer/pkg/common"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
)

const (
	configmapVolume = "config"
	configMountPath = "/config.json"
)

func deployment(ctx *common.RenderContext) ([]runtime.Object, error) {

	configHash, err := common.ObjectHash(configmap(ctx))
	if err != nil {
		return nil, err
	}

	labels := common.CustomizeLabel(ctx, Component, common.TypeMetaDeployment)
	return []runtime.Object{
		&appsv1.Deployment{
			TypeMeta: common.TypeMetaDeployment,
			ObjectMeta: metav1.ObjectMeta{
				Name:      Component,
				Namespace: ctx.Namespace,
				Labels:    labels,
				Annotations: common.CustomizeAnnotation(ctx, Component, common.TypeMetaDeployment, func() map[string]string {
					return map[string]string{
						common.AnnotationConfigChecksum: configHash,
					}
				}),
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{MatchLabels: labels},
				Replicas: common.Replicas(ctx, Component),
				Strategy: common.DeploymentStrategy,
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Name:        Component,
						Namespace:   ctx.Namespace,
						Labels:      labels,
						Annotations: common.CustomizeAnnotation(ctx, Component, common.TypeMetaDeployment),
					},
					Spec: corev1.PodSpec{
						Affinity:                      common.NodeAffinity(cluster.AffinityLabelMeta),
						ServiceAccountName:            Component,
						EnableServiceLinks:            pointer.Bool(false),
						DNSPolicy:                     "ClusterFirst",
						RestartPolicy:                 "Always",
						TerminationGracePeriodSeconds: pointer.Int64(30),
						Containers: []corev1.Container{
							{
								Name:  Component,
								Image: ctx.ImageName(ctx.Config.Repository, Component, ctx.VersionManifest.Components.PublicAPIServer.Version),
								Args: []string{
									"run",
									fmt.Sprintf("--config=%s", configMountPath),
									"--json-log=true",
								},
								ImagePullPolicy: corev1.PullIfNotPresent,
								Resources: common.ResourceRequirements(ctx, Component, Component, corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										"cpu":    resource.MustParse("100m"),
										"memory": resource.MustParse("32Mi"),
									},
								}),
								Ports: []corev1.ContainerPort{
									{
										ContainerPort: GRPCContainerPort,
										Name:          GRPCPortName,
									},
								},
								SecurityContext: &corev1.SecurityContext{
									Privileged: pointer.Bool(false),
								},
								Env: common.CustomizeEnvvar(ctx, Component, common.MergeEnv(
									common.DefaultEnv(&ctx.Config),
								)),
								LivenessProbe: &corev1.Probe{
									ProbeHandler: corev1.ProbeHandler{
										HTTPGet: &corev1.HTTPGetAction{
											Path:   "/live",
											Port:   intstr.IntOrString{IntVal: baseserver.BuiltinHealthPort},
											Scheme: corev1.URISchemeHTTP,
										},
									},
									FailureThreshold: 3,
									SuccessThreshold: 1,
									TimeoutSeconds:   1,
								},
								ReadinessProbe: &corev1.Probe{
									ProbeHandler: corev1.ProbeHandler{
										HTTPGet: &corev1.HTTPGetAction{
											Path:   "/ready",
											Port:   intstr.IntOrString{IntVal: baseserver.BuiltinHealthPort},
											Scheme: corev1.URISchemeHTTP,
										},
									},
									FailureThreshold: 3,
									SuccessThreshold: 1,
									TimeoutSeconds:   1,
								},
								VolumeMounts: []corev1.VolumeMount{
									{
										Name:      configmapVolume,
										ReadOnly:  true,
										MountPath: configMountPath,
										SubPath:   configJSONFilename,
									},
								},
							},
							*common.KubeRBACProxyContainerWithConfig(ctx),
						},
						Volumes: []corev1.Volume{
							{
								Name: configmapVolume,
								VolumeSource: corev1.VolumeSource{
									ConfigMap: &corev1.ConfigMapVolumeSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: Component,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}, nil
}
