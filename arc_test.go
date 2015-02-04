package mpcformat_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/soniakeys/mpcformat"
)

const (
	// a single obs
	o1Desig = "NE00030"
	o1      = `     NE00030  C2004 09 16.15206 16 13 11.57 +20 52 23.7          21.1 Vd     291
`
	// two obs
	o2Desig = "NE00199"
	o2      = `     NE00199  C2007 02 09.24234 06 08 06.06 +43 13 26.2          20.1  c     704
     NE00199  C2007 02 09.25415 06 08 05.51 +43 13 01.7          20.1  c     704
`
	// three obs
	o3Desig = "NE00269"
	o3      = `     NE00269  C2003 01 06.51893 12 40 50.09 +18 27 46.9          21.4 Vd     291
     NE00269  C2003 01 06.52850 12 40 50.71 +18 27 46.1          21.8 Vd     291
     NE00269  C2003 01 06.54359 12 40 51.68 +18 27 42.5          21.9 Vd     291
`
	// two-line satellite obs
	satDesig = "03620"
	sat      = `03620         S1996 08 30.51477 21 07 31.918-05 22 00.82                27764250
03620         s1996 08 30.51477 1 -  344.3553 - 6919.1239 +  872.2948   27764250
`
	short = `NE00030 C2004 09 16.15206 16 13 11.57 +20 52 23.7 21.1 V 291
`
	bad = `REALLY BRIGHT IN THE EAST JUST AFTER SUNSET
`
)

type arcRes struct {
	desig string
	nObs  int
	ok    bool // false means parse error
}

var arcTests = []struct {
	desc  string
	obs80 string
	want  []arcRes
}{
	{"no data", "", nil},
	{"single obs", o1, []arcRes{
		{o1Desig, 1, true},
	}},
	{"two obs", o2, []arcRes{
		{o2Desig, 2, true},
	}},
	{"two arcs", o1 + o2, []arcRes{
		{o1Desig, 1, true},
		{o2Desig, 2, true},
	}},
	{"satellite", sat, []arcRes{
		{satDesig, 1, true},
	}},
	{"mix", o3 + sat + sat + o1, []arcRes{
		{o3Desig, 3, true},
		{satDesig, 2, true},
		{o1Desig, 1, true},
	}},
	{"bad", bad, []arcRes{
		{"", 0, false},
	}},
	{"short", short, []arcRes{
		{"", 0, false},
	}},
	{"bad mix", o1 + short + sat + bad + bad + o3, []arcRes{
		{o1Desig, 1, true},
		{"", 0, false},
		{satDesig, 1, true},
		{"", 0, false},
		{"", 0, false},
		{o3Desig, 3, true},
	}},
	{"no final lf", o3[:len(o3)-1], []arcRes{
		{o3Desig, 3, true},
	}},
	{"single obs missing final lf", o3 + o1[:len(o1)-1], []arcRes{
		{o3Desig, 3, true},
		{o1Desig, 1, true},
	}},
}

func TestArcSplitter(t *testing.T) {
	for _, tc := range arcTests {
		f := mpcformat.ArcSplitter(bytes.NewBufferString(tc.obs80), pMap)
		for _, want := range tc.want {
			got, gotErr := f()
			switch {
			case gotErr != nil:
				if want.ok {
					t.Fatalf("%s err: %s", tc.desc, gotErr)
				}
				if _, ok := gotErr.(mpcformat.ArcError); !ok {
					t.Fatalf("%s error %s type %T, want mpcformat.ArcError",
						tc.desc, gotErr, gotErr)
				}
				continue
			case !want.ok:
				t.Fatalf("%s want parse error", tc.desc)
			}
			// check result
			if got.Desig != want.desig {
				t.Fatalf("%s .Desig = %s, want %s",
					tc.desc, got.Desig, want.desig)
			}
			if len(got.Obs) != want.nObs {
				t.Fatalf("%s returned %d obs, want %d",
					tc.desc, len(got.Obs), want.nObs)
			}
		}
		// additional read should return EOF
		_, gotErr := f()
		if gotErr != io.EOF {
			t.Fatalf("%s read past end got err = %v, want io.EOF",
				tc.desc, gotErr)
		}
	}
}
