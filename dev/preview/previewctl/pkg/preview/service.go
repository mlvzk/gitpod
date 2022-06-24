// Copyright (c) 2022 Gitpod GmbH. All rights reserved.
// Licensed under the GNU Affero General Public License (AGPL).
// See License-AGPL.txt in the project root for license information.

package preview

import (
	"context"
	"time"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

var errSvcNotReady = errors.New("proxy service not ready")

const proxySvcName = "proxy"

func (p *Preview) getVMProxySvcStatus(ctx context.Context) error {
	_, err := p.harvesterKubeClient.CoreV1().Namespaces().Get(ctx, p.namespace, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(errSvcNotReady, err.Error())
	}

	svc, err := p.harvesterKubeClient.CoreV1().Services(p.namespace).Get(ctx, proxySvcName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(errSvcNotReady, err.Error())
	}

	if svc.Spec.ClusterIP == "" {
		return errSvcNotReady
	}

	return nil
}

func (p *Preview) waitProxySvcReady(ctx context.Context, doneCh chan struct{}) error {
	kubeInformerFactory := informers.NewSharedInformerFactoryWithOptions(p.harvesterKubeClient, time.Second*30, informers.WithNamespace(p.namespace))
	svcInformer := kubeInformerFactory.Core().V1().Services().Informer()

	stopCh := make(chan struct{})
	defer close(stopCh)

	svcInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			svc := obj.(*v1.Service)
			if svc.Namespace == p.namespace && svc.Name == proxySvcName {
				stopCh <- struct{}{}
			}
		},
	})

	kubeInformerFactory.Start(stopCh)
	kubeInformerFactory.WaitForCacheSync(wait.NeverStop)

	for {
		select {
		case <-doneCh:
			p.logger.Infof("service [%s] created", proxySvcName)
			return nil
		case <-ctx.Done():
			return ctx.Err()
		case <-stopCh:
			return errors.New("received stop signal")
		case <-time.Tick(5 * time.Second):
			p.logger.Infof("waiting for svc [%s] to get created", proxySvcName)
		}
	}
}
