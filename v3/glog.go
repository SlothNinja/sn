package sn

import (
	"encoding/json"
	"time"
)

type entry struct {
	Template   string
	Data       H
	UpdatedAt  time.Time
	SubEntries []subentry
}

type subentry struct {
	Template string
	Data     H
}

type glog []entry

// H provides a general data map
// corresponds to H data map used in gin package.
type H map[string]any

func (g *Game[S, T, P]) DeepCopy() *Game[S, T, P] {
	return deepCopy(g)
}

func deepCopy[T any](obj T) T {
	v, err := json.Marshal(obj)
	if err != nil {
		Errorf("unable to marshal object: %v", err)
		panic("unable to marshal object")
	}

	var obj2 T
	err = json.Unmarshal(v, &obj2)
	if err != nil {
		Errorf("unable to unmarshal game: %v", err)
		panic("unable to unmarshal game")
	}
	return obj2
}

func (g *Game[S, T, P]) NewEntry(template string, data H, updatedAt time.Time) {
	g.newEntry(template, data, updatedAt)
}

func (g *Game[S, T, P]) newEntry(template string, data H, updatedAt time.Time) {
	g.Log = append(g.Log, entry{Template: template, Data: data, UpdatedAt: updatedAt})
}

func (g *Game[S, T, P]) UpdateLastEntry(data H, updatedAt time.Time) {
	lastIndex := len(g.Log) - 1
	if lastIndex < 0 {
		Warningf("no log entry")
		return
	}
	g.Log[lastIndex].UpdatedAt = updatedAt
	for k, v := range data {
		g.Log[lastIndex].Data[k] = v
	}
}

func (g *Game[S, T, P]) NewSubEntry(template string, data H, updatedAt time.Time) {
	lastEntryIndex := len(g.Log) - 1
	if lastEntryIndex < 0 {
		Warningf("no log entry")
		return
	}
	g.Log[lastEntryIndex].UpdatedAt = updatedAt
	g.Log[lastEntryIndex].SubEntries = append(g.Log[lastEntryIndex].SubEntries, subentry{Template: template, Data: data})
}

func (g *Game[S, T, P]) UpdateLastSubEntry(template string, data H) {
	lastEntryIndex := len(g.Log) - 1
	if lastEntryIndex < 0 {
		Warningf("no log entry")
		return
	}

	lastSubEntryIndex := len(g.Log[lastEntryIndex].SubEntries) - 1
	if lastSubEntryIndex < 0 {
		Warningf("no log subentry")
		return
	}
	for k, v := range data {
		g.Log[lastEntryIndex].SubEntries[lastSubEntryIndex].Data[k] = v
	}
}
