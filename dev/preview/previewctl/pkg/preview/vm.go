// Copyright (c) 2022 Gitpod GmbH. All rights reserved.
// Licensed under the GNU Affero General Public License (AGPL).
// See License-AGPL.txt in the project root for license information.

package preview

import (
	"context"
	"time"

	"github.com/pkg/errors"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
	virtv1 "kubevirt.io/api/core/v1"
)

var (
	vmResource = schema.GroupVersionResource{
		Group:    "kubevirt.io",
		Version:  "v1",
		Resource: "virtualmachines",
	}

	vmInstanceResource = schema.GroupVersionResource{
		Group:    "kubevirt.io",
		Version:  "v1",
		Resource: "virtualmachineinstances",
	}

	errGettingVMIpAddress       = errors.New("error parsing vm ip address")
	errVMInterfacesLengthIsZero = errors.New("vm interfaces length is 0")
	errVMNoDefaultIPAddrFound   = errors.New("vm no default ip addr found")
	errVmNotReady               = errors.New("vm not ready")
)

func getVMInstanceIPAddress(vm virtv1.VirtualMachineInstance) (string, error) {
	if len(vm.Status.Interfaces) == 0 {
		return "", errVMInterfacesLengthIsZero
	}

	for _, i := range vm.Status.Interfaces {
		if i.Name == "default" {
			return i.IP, nil
		}
	}

	return "", errVMNoDefaultIPAddrFound
}

func parseVMInstanceIPAddress(obj *unstructured.Unstructured) (string, error) {
	var vm virtv1.VirtualMachineInstance
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), &vm)
	if err != nil {
		return "", errors.Wrap(err, errGettingVMIpAddress.Error())
	}

	return getVMInstanceIPAddress(vm)
}

func (p *Preview) getVMStatus(ctx context.Context) error {
	_, err := p.harvesterKubeClient.CoreV1().Namespaces().Get(ctx, p.namespace, metav1.GetOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return err
	}

	if kerrors.IsNotFound(err) {
		p.logger.Infof("namespace [%s] not found", p.namespace)
		return errors.Wrap(errVmNotReady, err.Error())
	}

	vmClient := p.harvesterDynamicClient.Resource(vmInstanceResource).Namespace(p.namespace)
	res, err := vmClient.Get(ctx, p.name, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(errVmNotReady, err.Error())
	}

	ip, err := parseVMInstanceIPAddress(res)
	if err != nil {
		return errors.Wrap(errVmNotReady, err.Error())
	}

	if ip == "" {
		return errVmNotReady
	}

	return nil
}

func (p *Preview) waitVMReady(ctx context.Context, doneCh chan struct{}) error {
	kubeInformerFactory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(p.harvesterDynamicClient, time.Second*30, p.namespace, nil)
	vmInformer := kubeInformerFactory.ForResource(vmInstanceResource).Informer()

	vmInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			res := obj.(*unstructured.Unstructured)
			ip, err := parseVMInstanceIPAddress(res)
			if err != nil {
				p.logger.Warnf("error parsing VM IP Addr: %v", err)
				return
			}

			if ip == "" {
				return
			}

			doneCh <- struct{}{}
		},
		// We also watch for updates
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			res := newObj.(*unstructured.Unstructured)
			ip, err := parseVMInstanceIPAddress(res)
			if err != nil {
				p.logger.Warnf("error parsing VM IP Addr: %v", err)
				return
			}

			if ip == "" {
				return
			}

			doneCh <- struct{}{}
		},
	})

	stopCh := make(chan struct{})
	defer close(stopCh)

	kubeInformerFactory.Start(stopCh)
	kubeInformerFactory.WaitForCacheSync(wait.NeverStop)

	for {
		select {
		case <-doneCh:
			p.logger.Infof("[%s] vm ready", p.name)
			return nil
		case <-ctx.Done():
			return ctx.Err()
		case <-stopCh:
			return errors.New("received stop signal")
		case <-time.Tick(5 * time.Second):
			p.logger.Infof("waiting for vm [%s] to get ready", p.name)
		}
	}
}

func (p *Preview) getVMs(ctx context.Context) ([]string, error) {
	virtualMachineClient := p.harvesterDynamicClient.Resource(vmResource).Namespace("")
	vmObjs, err := virtualMachineClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var vms []string
	for _, item := range vmObjs.Items {
		vms = append(vms, item.GetName())
	}

	return vms, nil
}
