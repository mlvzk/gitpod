// Copyright (c) 2022 Gitpod GmbH. All rights reserved.
// Licensed under the GNU Affero General Public License (AGPL).
// See License-AGPL.txt in the project root for license information.

package preview

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetVMProxySvcStatus(t *testing.T) {
	p := &Preview{
		branch:    "test",
		namespace: "preview-test",
		logger:    nil,
	}

	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: p.namespace,
		},
	}

	testCases := []struct {
		name    string
		objects []runtime.Object
		err     error
	}{
		{
			name: "namespace not found error, nothing exists",
			err:  errors.Wrap(errSvcNotReady, kerrors.NewNotFound(v1.Resource("namespaces"), p.namespace).Error()),
		},
		{
			name:    "service not found error, namespace exists",
			objects: []runtime.Object{ns},
			err:     errors.Wrap(errSvcNotReady, kerrors.NewNotFound(v1.Resource("services"), proxySvcName).Error()),
		},
		{
			name: "service not ready error, ns and svc exists",
			objects: []runtime.Object{
				ns,
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      proxySvcName,
						Namespace: p.namespace,
					},
					Spec: v1.ServiceSpec{
						ClusterIP: "",
					},
				},
			},
			err: errSvcNotReady,
		},
		{
			name: "no error",
			objects: []runtime.Object{
				ns,
				&v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      proxySvcName,
						Namespace: p.namespace,
					},
					Spec: v1.ServiceSpec{
						ClusterIP: "127.0.0.1",
					},
				},
			},
			err: nil,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			p.harvesterKubeClient = fake.NewSimpleClientset(test.objects...)

			err := p.getVMProxySvcStatus(context.TODO())

			if test.err != nil {
				// we have to compare on error value, as otherwise it will always fail since the errors are pointers
				assert.Equal(t, errors.Unwrap(test.err), errors.Unwrap(err))
			} else {
				assert.ErrorIs(t, err, test.err)
			}
		})
	}
}
