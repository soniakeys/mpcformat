// Public domain.

package mpcformat

import (
	"bufio"
	"errors"
	"io"

	"github.com/soniakeys/observation"
)

// Split splits an observation stream by designation, yielding observation arcs.
//
// The stream iObs is a stream of observations in the MPC 80 column format.
// The stream must have observations already grouped by designation and sorted
// chronologically within each object.  That is, this function does not sort
// them, but logic within the function relies on them being already sorted.
//
// Valid arcs are parsed against ocdMap and retuned on channel arcCh.
// Read errors are relayed on errCh should be considered fatal.
// Parse errors are not fatal.  They are quietly ignored and not relayed
// on errCh.  Lines causing parse errors and lines not forming valid arcs
// are dropped without notification.
func Split(iObs io.Reader, ocdMap observation.ParallaxMap,
	arcCh chan *observation.Arc, errCh chan error) {
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
			errCh <- errors.New("Split: unexpected long line")
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
			sendValid(des0, obuf, arcCh)
			obuf = obuf[:0]
		default:
			sendValid(des0, obuf, arcCh)
			fallthrough
		case len(obuf) == 0:
			des0 = desig
			obuf = obuf[:1]
			obuf[0] = o
		case desig == des0:
			obuf = append(obuf, o)
		}
	}
	sendValid(des0, obuf, arcCh)
	close(arcCh)
}

// checks that observations make a valid arc, allocates and sends.
func sendValid(
	desig string,
	obuf []observation.VObs,
	arcCh chan *observation.Arc,
) {
	if len(obuf) < 2 {
		return
	}
	// the first observation time must be positive and
	// observation times must increase after that
	var t0 float64
	for i := range obuf {
		t := obuf[i].Meas().MJD
		if t <= t0 {
			return
		}
		t0 = t
	}
	// object must show motion over the arc
	first := obuf[0].Meas()
	last := obuf[len(obuf)-1].Meas()
	if first.RA == last.RA && first.Dec == last.Dec {
		return
	}
	arcCh <- &observation.Arc{
		Desig: desig,
		Obs:   append([]observation.VObs{}, obuf...),
	}
}
