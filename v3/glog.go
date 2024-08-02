package sn

import (
	"encoding/json"
	"log/slog"
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

// DeepCopy provides a deep copy of the game
func (g *Game[S, T, P]) DeepCopy() *Game[S, T, P] {
	return deepCopy(g)
}

func deepCopy[T any](obj T) T {
	v, err := json.Marshal(obj)
	if err != nil {
		slog.Error("unable to marshal object: %v", err)
		panic("unable to marshal object")
	}

	var obj2 T
	err = json.Unmarshal(v, &obj2)
	if err != nil {
		slog.Error("unable to unmarshal object: %v", err)
		panic("unable to unmarshal object")
	}
	return obj2
}

// NewEntry adds a new log entry to the game log
func (g *Game[S, T, P]) NewEntry(template string, data H, updatedAt time.Time) {
	g.newEntry(template, data, updatedAt)
}

func (g *Game[S, T, P]) newEntry(template string, data H, updatedAt time.Time) {
	g.Log = append(g.Log, entry{Template: template, Data: data, UpdatedAt: updatedAt})
}

// UpdateLastEntry updates the last log entry in the game log
func (g *Game[S, T, P]) UpdateLastEntry(data H, updatedAt time.Time) {
	lastIndex := len(g.Log) - 1
	if lastIndex < 0 {
		slog.Warn("no log entry")
		return
	}
	g.Log[lastIndex].UpdatedAt = updatedAt
	for k, v := range data {
		g.Log[lastIndex].Data[k] = v
	}
}

// NewSubEntry adds a new sub entry to the last entry in the game log
func (g *Game[S, T, P]) NewSubEntry(template string, data H, updatedAt time.Time) {
	lastEntryIndex := len(g.Log) - 1
	if lastEntryIndex < 0 {
		slog.Warn("no log entry")
		return
	}
	g.Log[lastEntryIndex].UpdatedAt = updatedAt
	g.Log[lastEntryIndex].SubEntries = append(g.Log[lastEntryIndex].SubEntries, subentry{Template: template, Data: data})
}

// UpdateLastSubEntry updates the last sub entry in the game log
func (g *Game[S, T, P]) UpdateLastSubEntry(data H) {
	lastEntryIndex := len(g.Log) - 1
	if lastEntryIndex < 0 {
		slog.Warn("no log entry")
		return
	}

	lastSubEntryIndex := len(g.Log[lastEntryIndex].SubEntries) - 1
	if lastSubEntryIndex < 0 {
		slog.Warn("no log subentry")
		return
	}
	for k, v := range data {
		g.Log[lastEntryIndex].SubEntries[lastSubEntryIndex].Data[k] = v
	}
}
