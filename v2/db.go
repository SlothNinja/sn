package sn

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"cloud.google.com/go/datastore"
	"github.com/SlothNinja/log"
	"github.com/gin-gonic/gin"
)

const (
	statusKey = "Status"
	countKey  = "Count"
	NoCount   = -1
)

func getAllQuery(c *gin.Context) *datastore.Query {
	return datastore.NewQuery("Game").Ancestor(GamesRoot(c))
}

func (client *Client) getFiltered(c *gin.Context, status, sid, start, length string, t Type) (Gamers, int64, error) {
	client.Log.Debugf("Entering")
	defer client.Log.Debugf("Exiting")

	q := getAllQuery(c).
		KeysOnly()

	if status != "" {
		st := ToStatus[strings.ToLower(status)]
		q = q.Filter("Status=", int(st))
		WithStatus(c, st)
	}

	if sid != "" {
		if id, err := strconv.Atoi(sid); err == nil {
			q = q.Filter("UserIDS=", id)
		}
	}

	if t != All {
		q = q.Filter("Type=", int(t)).
			Order("-UpdatedAt")
	} else {
		q = q.Order("-UpdatedAt")
	}

	cnt, err := client.DS.Count(c, q)
	if err != nil {
		return nil, 0, err
	}

	if start != "" {
		if st, err := strconv.ParseInt(start, 10, 32); err == nil {
			q = q.Offset(int(st))
		}
	}

	if length != "" {
		if l, err := strconv.ParseInt(length, 10, 32); err == nil {
			q = q.Limit(int(l))
		}
	}

	ks, err := client.DS.GetAll(c, q, nil)
	if err != nil {
		return nil, 0, err
	}

	l := len(ks)
	gs := make([]Gamer, l)
	hs := make([]*Header, l)
	for i := range gs {
		var ok bool
		if t == All {
			k := strings.ToLower(ks[i].Parent.Kind)
			if t, ok = ToType[k]; !ok {
				return nil, 0, fmt.Errorf("Unknown Game Type For: %s", k)
			}
		}
		gs[i] = factories[t](c)
		hs[i] = gs[i].GetHeader()
	}

	err = client.DS.GetMulti(c, ks, hs)
	if err != nil {
		return nil, 0, err
	}

	if client.afterLoad {
		for i := range hs {
			err = client.AfterLoad(c, hs[i])
			if err != nil {
				return nil, 0, err
			}
		}
	}

	return gs, int64(cnt), nil
}

func WithStatus(c *gin.Context, s Status) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	c.Set(statusKey, s)
}

func StatusFrom(c *gin.Context) (s Status) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	if s = ToStatus[strings.ToLower(c.Param("status"))]; s != NoStatus {
		WithStatus(c, s)
	} else {
		s, _ = c.Value(statusKey).(Status)
	}
	return
}

func withCount(c *gin.Context, cnt int64) *gin.Context {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	c.Set(countKey, cnt)
	return c
}

func countFrom(c *gin.Context) (cnt int64) {
	cnt, _ = c.Value(countKey).(int64)
	return
}

func (client Client) GetFiltered(t Type) gin.HandlerFunc {
	return func(c *gin.Context) {
		client.Log.Debugf("Entering")
		defer client.Log.Debugf("Exiting")

		gs, cnt, err := client.getFiltered(c, c.Param("status"), c.Param("uid"), c.PostForm("start"), c.PostForm("length"), t)

		if err != nil {
			client.Log.Errorf(err.Error())
			c.Redirect(http.StatusSeeOther, homePath)
			c.Abort()
		}
		withGamers(withCount(c, cnt), gs)
	}
}

func (client Client) GetRunning(c *gin.Context) {
	client.Log.Debugf("Entering")
	defer client.Log.Debugf("Exiting")

	gs, cnt, err := client.getFiltered(c, c.Param("status"), "", "", "", All)

	if err != nil {
		client.Log.Errorf(err.Error())
		c.Redirect(http.StatusSeeOther, homePath)
		c.Abort()
	}
	withGamers(withCount(c, cnt), gs)
}
