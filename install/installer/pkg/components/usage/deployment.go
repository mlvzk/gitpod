// Copyright (c) 2022 Gitpod GmbH. All rights reserved.
// Licensed under the MIT License. See License-MIT.txt in the project root for license information.

package usage

import (
	"fmt"
	"path/filepath"

	"github.com/gitpod-io/gitpod/common-go/baseserver"
	"github.com/gitpod-io/gitpod/installer/pkg/cluster"
	"github.com/gitpod-io/gitpod/installer/pkg/common"
	"github.com/gitpod-io/gitpod/installer/pkg/config/v1/experimental"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
)

func deployment(ctx *common.RenderContext) ([]runtime.Object, error) {
	labels := common.CustomizeLabel(ctx, Component, common.TypeMetaDeployment)

	args := []string{
		"run",
		"--schedule=$(RECONCILER_SCHEDULE)",
	}

	var volumes []corev1.Volume
	var volumeMounts []corev1.VolumeMount
	_ = ctx.WithExperimental(func(cfg *experimental.Config) error {
		if cfg.WebApp != nil && cfg.WebApp.Server != nil && cfg.WebApp.Server.StripeSecret != "" {
			stripeSecret := cfg.WebApp.Server.StripeSecret

			volumes = append(volumes,
				corev1.Volume{
					Name: "stripe-secret",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: stripeSecret,
						},
					},
				})

			volumeMounts = append(volumeMounts, corev1.VolumeMount{
				Name:      "stripe-secret",
				MountPath: stripeSecretMountPath,
				ReadOnly:  true,
			})

			args = append(args, fmt.Sprintf("--stripe-secret-path=%s", filepath.Join(stripeSecretMountPath, stripeKeyFilename)))
		}
		return nil
	})

	return []runtime.Object{
		&appsv1.Deployment{
			TypeMeta: common.TypeMetaDeployment,
			ObjectMeta: metav1.ObjectMeta{
				Name:        Component,
				Namespace:   ctx.Namespace,
				Labels:      labels,
				Annotations: common.CustomizeAnnotation(ctx, Component, common.TypeMetaDeployment),
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
						InitContainers:                []corev1.Container{*common.DatabaseWaiterContainer(ctx)},
						Volumes:                       volumes,
						Containers: []corev1.Container{{
							Name:            Component,
							Image:           ctx.ImageName(ctx.Config.Repository, Component, ctx.VersionManifest.Components.Usage.Version),
							Args:            args,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Resources: common.ResourceRequirements(ctx, Component, Component, corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									"cpu":    resource.MustParse("100m"),
									"memory": resource.MustParse("64Mi"),
								},
							}),
							Ports: []corev1.ContainerPort{},
							SecurityContext: &corev1.SecurityContext{
								Privileged: pointer.Bool(false),
							},
							Env: common.CustomizeEnvvar(ctx, Component, common.MergeEnv(
								common.DefaultEnv(&ctx.Config),
								common.DatabaseEnv(&ctx.Config),
								[]corev1.EnvVar{{
									Name: "RECONCILER_SCHEDULE",
									ValueFrom: &corev1.EnvVarSource{
										ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{Name: fmt.Sprintf("%s-config", Component)},
											Key:                  "schedule",
										},
									},
								}},
							)),
							VolumeMounts: volumeMounts,
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
						},
							*common.KubeRBACProxyContainerWithConfig(ctx),
						},
					},
				},
			},
		},
	}, nil
}
