// Copyright (c) 2022 Gitpod GmbH. All rights reserved.
// Licensed under the GNU Affero General Public License (AGPL).
// See License-AGPL.txt in the project root for license information.

package db

import (
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"testing"
	"time"
)

func TestVarcharTime_Scan(t *testing.T) {
	type Expectation struct {
		Time  VarcharTime
		Error bool
	}

	for _, scenario := range []struct {
		Name string

		Input    interface{}
		Expected Expectation
	}{
		{
			Name:  "nil value does not error and sets invalid",
			Input: nil,
			Expected: Expectation{
				Error: false,
			},
		},
		{
			Name:  "empty uint8 slice does not error and sets invalid",
			Input: []uint8{},
			Expected: Expectation{
				Error: false,
			},
		},
		{
			Name:  "parses valid ISO 8601 from TypeScript from []uint8",
			Input: []uint8("2019-05-10T09:54:28.185Z"),
			Expected: Expectation{
				Time: VarcharTime{
					t:     time.Date(2019, 05, 10, 9, 54, 28, 185000000, time.UTC),
					valid: true,
				},
				Error: false,
			},
		},
		{
			Name:  "invalid string errors",
			Input: "2019-05-10T09:54:28.185Z-not-a-datetime",
			Expected: Expectation{
				Error: true,
			},
		},
		{
			Name:  "string is parsed",
			Input: "2019-05-10T09:54:28.185Z",
			Expected: Expectation{
				Time: VarcharTime{
					t:     time.Date(2019, 05, 10, 9, 54, 28, 185000000, time.UTC),
					valid: true,
				},
				Error: false,
			},
		},
	} {
		t.Run(scenario.Name, func(t *testing.T) {
			var vt VarcharTime
			err := vt.Scan(scenario.Input)

			require.Equal(t, scenario.Expected, Expectation{
				Time:  vt,
				Error: err != nil,
			})
		})
	}
}

func TestVarcharTime_Value_ISO8601(t *testing.T) {
	for _, scenario := range []struct {
		Time     VarcharTime
		Expected string
	}{
		{
			Time:     NewVarcharTime(time.Date(2019, 05, 10, 9, 54, 28, 185000000, time.UTC)),
			Expected: "2019-05-10T09:54:28.185Z",
		},
		{
			Time:     VarcharTime{},
			Expected: "",
		},
	} {
		wireFormat, err := scenario.Time.Value()
		require.NoError(t, err)
		require.Equal(t, scenario.Expected, wireFormat)
	}
}

func TestVarcharTime_String_ISO8601(t *testing.T) {
	for _, scenario := range []struct {
		Time     VarcharTime
		Expected string
	}{
		{
			Time:     NewVarcharTime(time.Date(2019, 05, 10, 9, 54, 28, 185000000, time.UTC)),
			Expected: "2019-05-10T09:54:28.185Z",
		},
		{
			Time:     VarcharTime{},
			Expected: "",
		},
	} {
		require.Equal(t, scenario.Expected, scenario.Time.String())
	}
}

func TestVarcharTime_SerializeAndDeserialize(t *testing.T) {
	// Custom table to be able to exercise serialization easily, independent of other models
	type VarcharModel struct {
		ID   int         `gorm:"primaryKey"`
		Time VarcharTime `gorm:"column:time;type:varchar(255);"`
	}

	conn := ConnectForTests(t)
	require.NoError(t, conn.AutoMigrate(&VarcharModel{}))

	conn.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&VarcharModel{})

	for _, scenario := range []struct {
		Description string
		Input       VarcharModel
		Expected    VarcharModel
	}{
		{
			Description: "empty value for VarcharTime",
			Input: VarcharModel{
				ID:   1,
				Time: VarcharTime{},
			},
			Expected: VarcharModel{
				ID:   1,
				Time: VarcharTime{},
			},
		},
	} {
		tx := conn.Create(scenario.Input)
		require.NoError(t, tx.Error)

		var read VarcharModel
		tx = conn.First(&read, scenario.Input.ID)
		require.NoError(t, tx.Error)

		require.Equal(t, scenario.Expected, read)
	}
}
