package sn

import (
	"maps"

	"google.golang.org/protobuf/types/known/timestamppb"
)

type entry struct {
	Template   string
	Data       H
	UpdatedAt  *timestamppb.Timestamp
	SubEntries []*subentry
}

type subentry struct {
	Template string
	Data     H
}

type glog []*entry

// H provides a general data map
// corresponds to H data map used in gin package.
type H map[string]any

// NewEntry adds a new log entry to the game log
func (g *Game[S, T, P]) NewEntry(template string, data H) {
	g.newEntry(template, data)
}

func (g *Game[S, T, P]) newEntry(template string, data H) {
	g.Log = append(g.Log, &entry{Template: template, Data: data, UpdatedAt: timestamppb.Now()})
}

// NewEntryFor adds a new log entry to the game log
func (g *Game[S, T, P]) NewEntryFor(p P, template string, data H) {
	g.newEntryFor(p, template, data)
}

func (g *Game[S, T, P]) newEntryFor(p P, template string, data H) {
	g.NewEntry(template, data)
	p.setLog(g.lastEntry())
}

// UpdateLastEntry updates the last log entry in the game log
func (g *Game[S, T, P]) UpdateLastEntry(data H) {
	e := g.lastEntry()
	e.UpdatedAt = timestamppb.Now()

	if e.Data == nil {
		e.Data = make(H, len(data))
	}
	maps.Insert(e.Data, maps.All(data))
}

func (g *Game[S, T, P]) UpdateLastEntryFor(p P, data H) {
	g.UpdateLastEntry(data)
	p.setLog(g.lastEntry())
}

// NewSubEntry adds a new sub entry to the last entry in the game log
func (g *Game[S, T, P]) NewSubEntry(template string, data H) {
	i := g.lastEntryIndex()
	g.Log[i].UpdatedAt = timestamppb.Now()
	g.Log[i].SubEntries = append(g.lastEntry().SubEntries, &subentry{Template: template, Data: data})
}

// NewSubEntryFor adds a new sub entry to the last entry in the game log and player log
func (g *Game[S, T, P]) NewSubEntryFor(p P, template string, data H) {
	g.NewSubEntry(template, data)
	p.setLog(g.lastEntry())
}

// UpdateLastSubEntry updates the last sub entry in the game log
func (g *Game[S, T, P]) UpdateLastSubEntry(data H) {
	sub := g.lastSubEntry()
	if sub.Data == nil {
		sub.Data = make(H, len(data))
	}
	maps.Insert(sub.Data, maps.All(data))
}

func (g *Game[S, T, P]) UpdateLastSubEntryFor(p P, data H) {
	g.UpdateLastSubEntry(data)
	p.setLog(g.lastEntry())
}

func (g *Game[S, T, P]) IsLastEntryFor(template string, p P) bool {
	e := g.lastEntry()
	return e.Template == template && e.Data["PID"] == p.PID()
}

func (g *Game[S, T, P]) NewEntryDemoteLastFor(p P, template string, data H) {
	e := g.lastEntry()
	subTemplate := e.Template
	subData := e.Data

	e.Template = template
	e.Data = data
	e.UpdatedAt = timestamppb.Now()
	g.NewSubEntryFor(p, subTemplate, subData)
}

func (g *Game[S, T, P]) lastEntryIndex() int {
	return len(g.Log) - 1
}

func (g *Game[S, T, P]) lastEntry() *entry {
	return g.Log[g.lastEntryIndex()]
}

func (g *Game[S, T, P]) lastSubEntries() []*subentry {
	return g.lastEntry().SubEntries
}

func (g *Game[S, T, P]) lastSubEntryIndex() int {
	return len(g.lastSubEntries()) - 1
}

func (g *Game[S, T, P]) lastSubEntry() *subentry {
	return g.lastSubEntries()[g.lastSubEntryIndex()]
}
