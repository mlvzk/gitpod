// Copyright (c) 2022 Gitpod GmbH. All rights reserved.
// Licensed under the GNU Affero General Public License (AGPL).
// See License-AGPL.txt in the project root for license information.

package v1

import (
	"path/filepath"

	"github.com/gitpod-io/gitpod/common-go/cgroups"
)

const (
	TotalInactiveFile = "total_inactive_file"
)

type Memory struct {
	path string
}

func NewMemoryController(path string) *Memory {
	path = filepath.Join(path, "memory")
	return &Memory{
		path: path,
	}
}

// Limit returns the memory limit in bytes
func (m *Memory) Limit() (uint64, error) {
	path := filepath.Join(m.path, "memory.limit_in_bytes")
	return cgroups.ReadSingleValue(path)
}

// Usage returns the memory usage in bytes
func (m *Memory) Usage() (uint64, error) {
	path := filepath.Join(m.path, "memory.usage_in_bytes")
	return cgroups.ReadSingleValue(path)
}

func (m *Memory) Stat() (*MemoryStats, error) {
	path := filepath.Join(m.path, "memory.stat")
	statMap, err := cgroups.ReadFlatKeyedFile(path)
	if err != nil {
		return nil, err
	}

	return &MemoryStats{
		InactiveFileTotal: statMap[TotalInactiveFile],
	}, nil
}

type MemoryStats struct {
	InactiveFileTotal uint64
}
