// Public domain.

package mpcformat

import (
	"bufio"
	"errors"
	"io"

	"github.com/soniakeys/observation"
)

// SplitTracklets splits an observation stream up into tracklets.
//
// The stream iObs is a stream of observations in the MPC 80 column format.
// The stream must have observations already grouped by designation and sorted
// chronologically within each object.  That is, this function does not sort
// them, but logic within the function relies on them being already sorted.
//
// Valid tracklets are parsed against ocdMap and retuned on channel tkCh.
// Read errors are relayed on errCh should be considered fatal.
// Parse errors are not fatal.  They are quietly ignored and not relayed
// on errCh.  Lines causing parse errors and lines not forming valid tracklets
// are dropped without notification.
func SplitTracklets(iObs io.Reader, ocdMap observation.ParallaxMap,
	tkCh chan *observation.Tracklet, errCh chan error) {
	bf := bufio.NewReader(iObs)
	var des0 string
	var o observation.VObs
	var desig string
	obuf := make([]observation.VObs, 0, 4)
	for {
		bLine, pre, err := bf.ReadLine()
		if err == io.EOF {
			break
		}
		if err != nil {
			errCh <- err
			break
		}
		if pre {
			errCh <- errors.New("splitTracklets: unexpected long line")
			break
		}
		if len(bLine) != 80 {
			continue
		}
		line := string(bLine)
		if line[14] == 's' {
			if s, ok := o.(*observation.SatObs); ok {
				ParseSat2(line, desig, s)
			}
			continue
		}
		desig, o, err = ParseObs80(line, ocdMap)
		switch {
		case err != nil:
			sendValid(des0, obuf, tkCh)
			obuf = obuf[:0]
		default:
			sendValid(des0, obuf, tkCh)
			fallthrough
		case len(obuf) == 0:
			des0 = desig
			obuf = obuf[:1]
			obuf[0] = o
		case desig == des0:
			obuf = append(obuf, o)
		}
	}
	sendValid(des0, obuf, tkCh)
	close(tkCh)
}

// checks that observations make a valid tracklet,
// allocates and sends the tracklet.
func sendValid(
	desig string,
	obuf []observation.VObs,
	tkCh chan *observation.Tracklet,
) {
	if len(obuf) < 2 {
		return
	}
	// the first observation time must be positive and
	// observation times must increase after that
	var t0 float64
	for i := range obuf {
		t := obuf[i].Meas().Mjd
		if t <= t0 {
			return
		}
		t0 = t
	}
	// object must show motion over the tracklet
	first := obuf[0].Meas()
	last := obuf[len(obuf)-1].Meas()
	if first.Ra == last.Ra && first.Dec == last.Dec {
		return
	}
	tkCh <- &observation.Tracklet{
		Desig: desig,
		Obs:   append([]observation.VObs{}, obuf...),
	}
}
