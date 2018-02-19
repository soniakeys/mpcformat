// Public domain.

package mpcformat_test

import (
	"reflect"
	"testing"

	"github.com/soniakeys/mpcformat"
)

type testCase struct {
	desc string
	arc  []mpcformat.TrackletSplitter
	want [][]int
}

// uses mock type defined in tracklet_ex_test.go
var testData = []testCase{
	{"empty arc", nil, [][]int{}},
	{
		"single observation",
		[]mpcformat.TrackletSplitter{
			mustMock("2015 01 26.0", ""),
		},
		[][]int{{0}},
	},
	{
		"single tracklet, < 3 hrs",
		[]mpcformat.TrackletSplitter{
			mustMock("2015 01 26.0", ""),
			mustMock("2015 01 26.05", ""),
			mustMock("2015 01 26.1", ""),
		},
		[][]int{{0, 1, 2}},
	},
	{
		"observations out of order",
		[]mpcformat.TrackletSplitter{
			mustMock("2015 01 26.0", ""),
			mustMock("2015 01 26.1", ""),
			mustMock("2015 01 26.05", ""),
		},
		[][]int{{0, 2, 1}},
	},
	{
		"just two obs, same night",
		[]mpcformat.TrackletSplitter{
			mustMock("2015 01 26.0", ""),
			mustMock("2015 01 26.4", ""),
		},
		[][]int{{0, 1}},
	},
	{
		"single obs one night, another obs the next night",
		[]mpcformat.TrackletSplitter{
			mustMock("2015 01 26.4", ""),
			mustMock("2015 01 27.0", ""),
		},
		[][]int{{0}, {1}},
	},
	{
		"next night follow-up",
		[]mpcformat.TrackletSplitter{
			mustMock("2015 01 26.0", ""),
			mustMock("2015 01 26.01", ""),
			mustMock("2015 01 26.02", ""),
			mustMock("2015 01 27.0", ""),
			mustMock("2015 01 27.01", ""),
			mustMock("2015 01 27.02", ""),
		},
		[][]int{{0, 1, 2}, {3, 4, 5}},
	},
	{
		"2-3 same night follow-up",
		[]mpcformat.TrackletSplitter{
			mustMock("2015 01 26.0", ""),
			mustMock("2015 01 26.01", ""),
			mustMock("2015 01 26.4", ""),
			mustMock("2015 01 26.41", ""),
			mustMock("2015 01 26.42", ""),
		},
		[][]int{{0, 1}, {2, 3, 4}},
	},
	{
		"3-2 same night follow-up",
		[]mpcformat.TrackletSplitter{
			mustMock("2015 01 26.0", ""),
			mustMock("2015 01 26.01", ""),
			mustMock("2015 01 26.02", ""),
			mustMock("2015 01 26.4", ""),
			mustMock("2015 01 26.41", ""),
		},
		[][]int{{0, 1, 2}, {3, 4}},
	},
	{
		"slow cadence but still same night",
		[]mpcformat.TrackletSplitter{
			mustMock("2015 01 26.0", ""),
			mustMock("2015 01 26.2", ""),
			mustMock("2015 01 26.4", ""),
		},
		[][]int{{0, 1, 2}},
	},
	{
		"tracklet < 6hr, including somewhat isolated single obs",
		[]mpcformat.TrackletSplitter{
			mustMock("2015 01 26.0", ""),
			mustMock("2015 01 26.01", ""),
			mustMock("2015 01 26.02", ""),
			mustMock("2015 01 26.2", ""),
		},
		[][]int{{0, 1, 2, 3}},
	},
	{
		"default: split at longest gap",
		[]mpcformat.TrackletSplitter{
			mustMock("2015 01 26.0", ""),
			mustMock("2015 01 26.01", ""),
			mustMock("2015 01 26.02", ""),
			mustMock("2015 01 26.3", ""),
		},
		[][]int{{0, 1, 2}, {3}},
	},
	{
		"multiple observers",
		[]mpcformat.TrackletSplitter{
			mustMock("2015 01 26.0", "site1"),
			mustMock("2015 01 26.01", "site2"),
			mustMock("2015 01 26.02", "site1"),
			mustMock("2015 01 26.03", "site2"),
			mustMock("2015 01 26.04", "site1"),
			mustMock("2015 01 26.05", "site2"),
		},
		[][]int{{0, 2, 4}, {1, 3, 5}},
	},
}

func TestFindTracklets(t *testing.T) {
	for _, tc := range testData {
		got := mpcformat.FindTrackletsIndex(tc.arc)
		if !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("case %s = %v, want %v", tc.desc, got, tc.want)
		}
	}
}
