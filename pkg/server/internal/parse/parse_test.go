// Copyright 2023 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package parse

import (
	"testing"

	"github.com/pingcap/tidb/pkg/expression"
	"github.com/pingcap/tidb/pkg/parser/mysql"
	"github.com/pingcap/tidb/pkg/parser/terror"
	"github.com/pingcap/tidb/pkg/server/internal/util"
	"github.com/pingcap/tidb/pkg/sessionctx/stmtctx"
	"github.com/pingcap/tidb/pkg/types"
	"github.com/stretchr/testify/require"
)

func TestParseExecArgs(t *testing.T) {
	type args struct {
		args        []expression.Expression
		boundParams [][]byte
		nullBitmap  []byte
		paramTypes  []byte
		paramValues []byte
	}
	tests := []struct {
		args   args
		err    error
		expect interface{}
	}{
		// Tests for int overflow
		{
			args{
				expression.Args2Expressions4Test(1),
				[][]byte{nil},
				[]byte{0x0},
				[]byte{1, 0},
				[]byte{0xff},
			},
			nil,
			int64(-1),
		},
		{
			args{
				expression.Args2Expressions4Test(1),
				[][]byte{nil},
				[]byte{0x0},
				[]byte{2, 0},
				[]byte{0xff, 0xff},
			},
			nil,
			int64(-1),
		},
		{
			args{
				expression.Args2Expressions4Test(1),
				[][]byte{nil},
				[]byte{0x0},
				[]byte{3, 0},
				[]byte{0xff, 0xff, 0xff, 0xff},
			},
			nil,
			int64(-1),
		},
		// Tests for date/datetime/timestamp
		{
			args{
				expression.Args2Expressions4Test(1),
				[][]byte{nil},
				[]byte{0x0},
				[]byte{12, 0},
				[]byte{0x0b, 0xda, 0x07, 0x0a, 0x11, 0x13, 0x1b, 0x1e, 0x01, 0x00, 0x00, 0x00},
			},
			nil,
			"2010-10-17 19:27:30.000001",
		},
		{
			args{
				expression.Args2Expressions4Test(1),
				[][]byte{nil},
				[]byte{0x0},
				[]byte{10, 0},
				[]byte{0x04, 0xda, 0x07, 0x0a, 0x11},
			},
			nil,
			"2010-10-17",
		},
		{
			args{
				expression.Args2Expressions4Test(1),
				[][]byte{nil},
				[]byte{0x0},
				[]byte{7, 0},
				[]byte{0x0b, 0xda, 0x07, 0x0a, 0x11, 0x13, 0x1b, 0x1e, 0x01, 0x00, 0x00, 0x00},
			},
			nil,
			"2010-10-17 19:27:30.000001",
		},
		{
			args{
				expression.Args2Expressions4Test(1),
				[][]byte{nil},
				[]byte{0x0},
				[]byte{7, 0},
				[]byte{0x07, 0xda, 0x07, 0x0a, 0x11, 0x13, 0x1b, 0x1e},
			},
			nil,
			"2010-10-17 19:27:30",
		},
		{
			args{
				expression.Args2Expressions4Test(1),
				[][]byte{nil},
				[]byte{0x0},
				[]byte{7, 0},
				[]byte{0x0d, 0xdb, 0x07, 0x02, 0x03, 0x04, 0x05, 0x06, 0x40, 0xe2, 0x01, 0x00, 0xf2, 0x02},
			},
			nil,
			"2011-02-03 04:05:06.123456+12:34",
		},
		{
			args{
				expression.Args2Expressions4Test(1),
				[][]byte{nil},
				[]byte{0x0},
				[]byte{7, 0},
				[]byte{0x0d, 0xdb, 0x07, 0x02, 0x03, 0x04, 0x05, 0x06, 0x40, 0xe2, 0x01, 0x00, 0x0e, 0xfd},
			},
			nil,
			"2011-02-03 04:05:06.123456-12:34",
		},
		{
			args{
				expression.Args2Expressions4Test(1),
				[][]byte{nil},
				[]byte{0x0},
				[]byte{7, 0},
				[]byte{0x00},
			},
			nil,
			types.ZeroDatetimeStr,
		},
		// Tests for time
		{
			args{
				expression.Args2Expressions4Test(1),
				[][]byte{nil},
				[]byte{0x0},
				[]byte{11, 0},
				[]byte{0x0c, 0x01, 0x78, 0x00, 0x00, 0x00, 0x13, 0x1b, 0x1e, 0x01, 0x00, 0x00, 0x00},
			},
			nil,
			"-120 19:27:30.000001",
		},
		{
			args{
				expression.Args2Expressions4Test(1),
				[][]byte{nil},
				[]byte{0x0},
				[]byte{11, 0},
				[]byte{0x08, 0x01, 0x78, 0x00, 0x00, 0x00, 0x13, 0x1b, 0x1e},
			},
			nil,
			"-120 19:27:30",
		},
		{
			args{
				expression.Args2Expressions4Test(1),
				[][]byte{nil},
				[]byte{0x0},
				[]byte{11, 0},
				[]byte{0x00},
			},
			nil,
			"0",
		},
		// For error test
		{
			args{
				expression.Args2Expressions4Test(1),
				[][]byte{nil},
				[]byte{0x0},
				[]byte{7, 0},
				[]byte{10},
			},
			mysql.ErrMalformPacket,
			nil,
		},
		{
			args{
				expression.Args2Expressions4Test(1),
				[][]byte{nil},
				[]byte{0x0},
				[]byte{11, 0},
				[]byte{10},
			},
			mysql.ErrMalformPacket,
			nil,
		},
		{
			args{
				expression.Args2Expressions4Test(1),
				[][]byte{nil},
				[]byte{0x0},
				[]byte{11, 0},
				[]byte{8, 2},
			},
			mysql.ErrMalformPacket,
			nil,
		},
	}
	for _, tt := range tests {
		err := ExecArgs(stmtctx.NewStmtCtx(), tt.args.args, tt.args.boundParams, tt.args.nullBitmap, tt.args.paramTypes, tt.args.paramValues, nil)
		require.Truef(t, terror.ErrorEqual(err, tt.err), "err %v", err)
		if err == nil {
			require.Equal(t, tt.expect, tt.args.args[0].(*expression.Constant).Value.GetValue())
		}
	}
}

func TestParseExecArgsAndEncode(t *testing.T) {
	dt := expression.Args2Expressions4Test(1)
	err := ExecArgs(stmtctx.NewStmtCtx(),
		dt,
		[][]byte{nil},
		[]byte{0x0},
		[]byte{mysql.TypeVarchar, 0},
		[]byte{4, 178, 226, 202, 212},
		util.NewInputDecoder("gbk"))
	require.NoError(t, err)
	require.Equal(t, "测试", dt[0].(*expression.Constant).Value.GetValue())

	err = ExecArgs(stmtctx.NewStmtCtx(),
		dt,
		[][]byte{{178, 226, 202, 212}},
		[]byte{0x0},
		[]byte{mysql.TypeString, 0},
		[]byte{},
		util.NewInputDecoder("gbk"))
	require.NoError(t, err)
	require.Equal(t, "测试", dt[0].(*expression.Constant).Value.GetString())
}

func TestParseStmtFetchCmd(t *testing.T) {
	tests := []struct {
		arg       []byte
		stmtID    uint32
		fetchSize uint32
		err       error
	}{
		{[]byte{3, 0, 0, 0, 50, 0, 0, 0}, 3, 50, nil},
		{[]byte{5, 0, 0, 0, 232, 3, 0, 0}, 5, 1000, nil},
		{[]byte{5, 0, 0, 0, 0, 8, 0, 0}, 5, maxFetchSize, nil},
		{[]byte{5, 0, 0}, 0, 0, mysql.ErrMalformPacket},
		{[]byte{1, 0, 0, 0, 3, 2, 0, 0, 3, 5, 6}, 0, 0, mysql.ErrMalformPacket},
		{[]byte{}, 0, 0, mysql.ErrMalformPacket},
	}

	for _, tc := range tests {
		stmtID, fetchSize, err := StmtFetchCmd(tc.arg)
		require.Equal(t, tc.stmtID, stmtID)
		require.Equal(t, tc.fetchSize, fetchSize)
		require.Equal(t, tc.err, err)
	}
}