// Public domain.

package mpcformat

import "sort"

// TrackletSplitter, implemented on an observation type, provides data needed
// to split an observation arc into tracklets.
type TrackletSplitter interface {
	MJD() float64     // date and time of a single observation
	Observer() string // string identifying the observer or site
}

type td struct {
	mjd   float64
	index int
}
type dated []td

func (t dated) Len() int           { return len(t) }
func (t dated) Less(i, j int) bool { return t[i].mjd < t[j].mjd }
func (t dated) Swap(i, j int)      { t[i], t[j] = t[j], t[i] }

type tk struct {
	index []int
	mean  float64
}
type tkList []tk

func (t tkList) Len() int           { return len(t) }
func (t tkList) Less(i, j int) bool { return t[i].mean < t[j].mean }
func (t tkList) Swap(i, j int)      { t[i], t[j] = t[j], t[i] }

// FindTrackletsIndex splits an observation arc into tracklets.
//
// Conceptually, a tracklet is an arc of a few observations in a short time
// period that show an object's motion.  Observations of a tracklet are
// typically by the same observer and are observed and measured under the same
// conditions.  This information is not preserved in a number of MPC formats
// so the function here uses heuristics to construct working trackets.
func FindTrackletsIndex(ts []TrackletSplitter) [][]int {
	m := map[string]dated{}
	for i, t := range ts {
		d := t.MJD()
		o := t.Observer()
		m[o] = append(m[o], td{d, i})
	}
	tl := make(tkList, 0, len(m))
	appendTl := func(set dated) {
		t := make([]int, len(set))
		s := 0.
		for i, o := range set {
			t[i] = o.index
			s += o.mjd
		}
		tl = append(tl, tk{t, s / float64(len(set))})
		return
	}
	var reduce func(set dated) // but not a mathematical set, just a list.
	reduce = func(set dated) { // set must have > 1 obs.
		d := set[len(set)-1].mjd - set[0].mjd
		// all obs with 1 hr (about .042 day) is a tracklet
		if d < .042 {
			appendTl(set)
			return
		}
		// 2-5 obs within 3 hrs make a reasonable tracklet
		if len(set) <= 5 && d < .125 {
			appendTl(set)
			return
		}
		// only 2 obs, handle now
		if len(set) == 2 {
			// both must be same night
			if d < .5 {
				appendTl(set)
			} else {
				appendTl(set[:1])
				appendTl(set[1:])
			}
			return
		}
		// split at longest gap
		split := 1
		next := set[1].mjd
		longest := next - set[0].mjd
		for s := 2; s < len(set); s++ {
			prev := next
			next = set[s].mjd
			if g := next - prev; g > longest {
				longest = g
				split = s
			}
		}
		lf := set[:split]
		rt := set[split:]
		// recurse immediately if each half has >= 3 positions
		if len(lf) >= 3 && len(rt) >= 3 {
			reduce(lf)
			reduce(rt)
			return
		}
		// if two split off from the same night, handle right away.
		if len(lf) == 2 && len(rt) >= 2 && lf[1].mjd-lf[0].mjd < .5 {
			appendTl(lf)
			reduce(rt)
			return
		}
		if len(rt) == 2 && len(lf) >= 2 && rt[1].mjd-rt[0].mjd < .5 {
			reduce(lf)
			appendTl(rt)
			return
		}
		// if whole set has 3 obs in same night, take it as a tracklet.
		if len(set) == 3 && d < .5 {
			appendTl(set)
			return
		}
		// if whole set within 6 hrs, take it regardless of number of obs.
		if d < .25 {
			appendTl(set)
			return
		}
		// otherwise recurse
		reduce(lf)
		reduce(rt)
	}
	for _, t1 := range m {
		sort.Sort(t1)
		reduce(t1)
	}
	sort.Sort(tl)
	index := make([][]int, len(tl))
	for i := range tl {
		index[i] = tl[i].index
	}
	return index
}
