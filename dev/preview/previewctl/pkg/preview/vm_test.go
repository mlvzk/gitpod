// Copyright (c) 2022 Gitpod GmbH. All rights reserved.
// Licensed under the GNU Affero General Public License (AGPL).
// See License-AGPL.txt in the project root for license information.

package preview

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	dfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
	virtv1 "kubevirt.io/api/core/v1"
)

func TestParseVMInstanceIPAddress(t *testing.T) {
	tests := []struct {
		name        string
		obj         virtv1.VirtualMachineInstance
		expectedErr error
	}{
		{
			obj:         virtv1.VirtualMachineInstance{},
			name:        "interfaces length is 0",
			expectedErr: errVMInterfacesLengthIsZero,
		},
		{
			obj: virtv1.VirtualMachineInstance{
				Status: virtv1.VirtualMachineInstanceStatus{
					Interfaces: []virtv1.VirtualMachineInstanceNetworkInterface{
						{
							Name: "test",
						},
					},
				},
			},
			name:        "no default",
			expectedErr: errVMNoDefaultIPAddrFound,
		},
		{
			obj: virtv1.VirtualMachineInstance{
				Status: virtv1.VirtualMachineInstanceStatus{
					Interfaces: []virtv1.VirtualMachineInstanceNetworkInterface{
						{
							Name: "default",
							IP:   "127.0.0.1",
						},
					},
				},
			},
			name:        "all good",
			expectedErr: nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := getVMInstanceIPAddress(test.obj)

			assert.ErrorIs(t, err, test.expectedErr)
		})
	}
}

func TestGetVMStatus(t *testing.T) {
	p := &Preview{
		name:      "test",
		branch:    "test",
		namespace: "preview-test",
		logger:    logrus.NewEntry(logrus.New()),
	}

	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: p.namespace,
		},
	}

	testCases := []struct {
		name       string
		objects    []runtime.Object
		dynObjects []runtime.Object
		err        error
	}{
		{
			name: "namespace not found error, nothing exists",
			err:  errors.Wrap(errVmNotReady, kerrors.NewNotFound(v1.Resource("namespaces"), p.namespace).Error()),
		},
		{
			name:    "vminstance not found error, namespace exists",
			objects: []runtime.Object{ns},
			err:     errors.Wrap(errVmNotReady, kerrors.NewNotFound(virtv1.Resource("virtualmachineinstances"), p.name).Error()),
		},
		// Ideally we should add more tests for getting the VM
		// But trying to retrieve the VirtualMachineInstance{} type with the dynamic client fails
		// As there's an uint64 field in the spec, and the unstructured converter panics:
		// https://github.com/kubernetes/apimachinery/blob/a58f9b57c0c7f9c017891e44431fe3a032f12f8c/pkg/runtime/converter.go#L611-L641
		// TODO: figure out a workaround for this
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			p.harvesterKubeClient = fake.NewSimpleClientset(test.objects...)

			scheme := runtime.NewScheme()
			_ = virtv1.AddToScheme(scheme)

			p.harvesterDynamicClient = dfake.NewSimpleDynamicClient(scheme, test.dynObjects...)
			err := p.getVMStatus(context.TODO())

			if test.err != nil {
				// we have to compare on error value, as otherwise it will always fail since the errors are pointers
				assert.Equal(t, errors.Unwrap(test.err), errors.Unwrap(err))
			} else {
				assert.ErrorIs(t, errors.Unwrap(err), test.err)
			}
		})
	}
}
