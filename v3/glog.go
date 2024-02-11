package sn

type entry struct {
	Template   string
	Data       H
	SubEntries []subentry
}

type subentry struct {
	Template string
	Data     H
}

type glog []entry

type H map[string]any

func (g *Game[S, T, P]) NewEntry(template string, data H) {
	g.Log = append(g.Log, entry{Template: template, Data: data})
}

func (g *Game[S, T, P]) UpdateLastEntry(data H) {
	lastIndex := len(g.Log) - 1
	if lastIndex < 0 {
		Warningf("no log entry")
		return
	}
	for k, v := range data {
		g.Log[lastIndex].Data[k] = v
	}
}

func (g *Game[S, T, P]) NewSubEntry(template string, data H) {
	lastEntryIndex := len(g.Log) - 1
	if lastEntryIndex < 0 {
		Warningf("no log entry")
		return
	}
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
