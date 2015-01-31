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
	ok    bool // error == nil
	eof   bool // the only special value expected from readErr
}

var arcTests = []struct {
	desc  string
	obs80 string
	want  []arcRes
}{
	{"no data", "", nil},
	{"single obs", o1, []arcRes{
		{o1Desig, 1, true, true},
	}},
	{"two obs", o2, []arcRes{
		{o2Desig, 2, true, true},
	}},
	{"two arcs", o1 + o2, []arcRes{
		{o1Desig, 1, true, false},
		{o2Desig, 2, true, true},
	}},
	{"satellite", sat, []arcRes{
		{satDesig, 1, true, true},
	}},
	{"mix", o3 + sat + sat + o1, []arcRes{
		{o3Desig, 3, true, false},
		{satDesig, 2, true, false},
		{o1Desig, 1, true, true},
	}},
	{"bad", bad, []arcRes{
		{"", 0, false, true},
	}},
	{"short", short, []arcRes{
		{"", 0, false, true},
	}},
	{"bad mix", o1 + short + sat + bad + bad + o3, []arcRes{
		{o1Desig, 1, true, false},
		{"", 0, false, false},
		{satDesig, 1, true, false},
		{"", 0, false, false},
		{"", 0, false, false},
		{o3Desig, 3, true, true},
	}},
}

func TestArcSplitter(t *testing.T) {
	for _, tc := range arcTests {
		f := mpcformat.ArcSplitter(bytes.NewBufferString(tc.obs80), pMap)
		for _, want := range tc.want {
			got, gotErr := f()
			switch {
			case gotErr == io.EOF:
				if !want.eof {
					t.Fatalf("%s: EOF", tc.desc)
				}
			case gotErr != nil:
				if want.ok {
					t.Fatalf("%s err: %s", tc.desc, gotErr)
				}
				if _, ok := gotErr.(mpcformat.ArcError); !ok {
					t.Fatalf("%s error %s type %T, want mpcformat.ArcError",
						tc.desc, gotErr, gotErr)
				}
				continue
			case want.eof:
				t.Fatalf("%s want EOF", tc.desc)
			case !want.ok:
				t.Fatalf("%s want parse error", tc.desc)
			}
			// eof was as expected, otherwise all okay, check result
			if got.Desig != want.desig {
				t.Fatalf("%s .Desig = %s, want %s",
					tc.desc, got.Desig, want.desig)
			}
			if len(got.Obs) != want.nObs {
				t.Fatalf("%s returned %d obs, want %d",
					tc.desc, len(got.Obs), want.nObs)
			}
		}
		// additional read should return EOF with no observations
		got, gotErr := f()
		if gotErr != io.EOF {
			t.Fatalf("%s read past end should return io.EOF", tc.desc)
		}
		if len(got.Obs) != 0 {
			t.Fatalf("%s read past end returned %d observations.  want 0.",
				tc.desc, len(got.Obs))
		}
	}
}
