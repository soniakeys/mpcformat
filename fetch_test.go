// Public domain.

//+build fetch

package mpcformat_test

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/soniakeys/mpcformat"
)

func TestFetch(t *testing.T) {
	// git a temp file name
	f, err := ioutil.TempFile("", "testfetch")
	if err != nil {
		t.Fatal(err)
	}
	fn := f.Name()
	defer os.Remove(fn)
	f.Close()

	// fetch obscode.dat, write to temp file
	if err = mpcformat.FetchObscodeDat(fn); err != nil {
		t.Fatal(err)
	}

	// read the temp file
	m, err := mpcformat.ReadObscodeDatFile(fn)
	if err != nil {
		t.Fatal(err)
	}

	// there should be lots
	if len(m) < 1800 {
		t.Fatal("Loaded only", len(m), "sites, want > 1800")
	}

	// and the test cases should pass
	testParallaxMap(m, t)
}
