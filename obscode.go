// Public domain.

package mpcformat

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/soniakeys/observation"
	"github.com/soniakeys/unit"
)

// ObscodeDatURL links to the present location of the file known as obscode.dat,
// a flat file containing three-character MPC assigned observatory codes,
// associated with parallax constants and observatory names.
//
// See http://www.minorplanetcenter.net/iau/lists/ObsCodesF.html for
// a page containing this url.
var ObscodeDatURL = "http://www.minorplanetcenter.net/iau/lists/ObsCodes.html"

// FetchObscodeDat gets a fresh copy of the data at ObscodeDatURL (obscode.dat)
// and writes it to a new file with the path and file name ocdFile.
func FetchObscodeDat(ocdFile string) error {
	r, err := http.Get(ObscodeDatURL)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	f, err := os.Create(ocdFile)
	if err != nil {
		return err
	}
	if _, err = io.Copy(f, r.Body); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// ReadObscodeDatFile reads an MPC obscode.dat file.
//
// See ReadObscodeDat().
func ReadObscodeDatFile(ocdFile string) (observation.ParallaxMap, error) {
	f, err := os.Open(ocdFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	m, err := ReadObscodeDat(f)
	if err != nil {
		// add filename to error message
		err = fmt.Errorf("file %s: %s", ocdFile, err)
	}
	return m, err
}

// ReadObscodeDat parses parallax data from the format of the MPC obscode.dat
// file.
//
// Note that files prepared for display in a web browser will have column
// headings and additional markup.  This function does not require these lines;
// it quietly ignores lines that do not parse as data.
//
// Returned is a map from 3-character MPC obs codes to parallax constants.
//
// If rhoCosPhi and rhoSinPhi both == 0, nil is stored as the map value.
func ReadObscodeDat(r io.Reader) (observation.ParallaxMap, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	ocdMap := make(observation.ParallaxMap)
	var longitude, rhoCosPhi, rhoSinPhi float64

	for _, line := range strings.Split(string(b), "\n") {
		if len(line) < 30 {
			continue // quietly ignore extraneous lines such as <pre>
		}

		// scale factor = earth radius in m / 1 AU in m
		const sf = 6.37814e6 / 149.59787e9

		if ts := strings.TrimSpace(line[4:13]); len(ts) == 0 {
			longitude = 0 // blank fields default to 0
		} else {
			longitude, err = strconv.ParseFloat(ts, 64)
			if err != nil || longitude < 0 || longitude >= 360 {
				// quietly ignore lines with invalid longitude,
				// such as column heading line.
				continue
			}
		}

		if ts := strings.TrimSpace(line[13:21]); len(ts) == 0 {
			rhoCosPhi = 0
		} else {
			rhoCosPhi, err = strconv.ParseFloat(ts, 64)
			if err != nil || rhoCosPhi < 0 || rhoCosPhi > 1 {
				continue
			}
			rhoCosPhi *= sf
		}

		if ts := strings.TrimSpace(line[21:30]); len(ts) == 0 {
			rhoSinPhi = 0
		} else {
			rhoSinPhi, err = strconv.ParseFloat(ts, 64)
			if err != nil || rhoSinPhi < -1 || rhoSinPhi > 1 {
				continue
			}
			rhoSinPhi *= sf
		}

		if rhoCosPhi == 0 && rhoSinPhi == 0 {
			ocdMap[line[0:3]] = nil
		} else {
			ocdMap[line[0:3]] =
				&observation.ParallaxConst{
					Longitude: unit.AngleFromDeg(longitude),
					RhoCosPhi: rhoCosPhi,
					RhoSinPhi: rhoSinPhi,
				}
		}
	}
	if len(ocdMap) == 0 {
		return nil, errors.New("Obscode data unreadable")
	}
	return ocdMap, nil
}
