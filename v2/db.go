package sn

import (
	"fmt"
	"strconv"

	"cloud.google.com/go/datastore"
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

func (client *Client) getFiltered(c *gin.Context, status Status, sid, start, length string, t Type) (Gamers, int64, error) {
	client.Log.Debugf("Entering")
	defer client.Log.Debugf("Exiting")

	q := getAllQuery(c).
		KeysOnly()

	if status != NoStatus {
		q = q.Filter("Status=", status)
	}

	if sid != "" {
		if id, err := strconv.Atoi(sid); err == nil {
			q = q.Filter("UserIDS=", id)
		}
	}

	if t != All {
		q = q.Filter("Type=", string(t)).
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
		if t == All {
			t2 := ToType(ks[i].Parent.Kind)
			if t2 == NoType {
				return nil, 0, fmt.Errorf("Missing Type")
			}
			gs[i] = factories[t2](c)
		} else {
			gs[i] = factories[t](c)
		}
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

// func (client Client) GetFiltered(t Type) gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		client.Log.Debugf("Entering")
// 		defer client.Log.Debugf("Exiting")
//
// 		status := ToStatus(c.Param("status"))
// 		gs, cnt, err := client.getFiltered(c, status, c.Param("uid"), c.PostForm("start"), c.PostForm("length"), t)
//
// 		if err != nil {
// 			client.Log.Errorf(err.Error())
// 			c.Redirect(http.StatusSeeOther, homePath)
// 			c.Abort()
// 		}
// 		withGamers(withCount(c, cnt), gs)
// 	}
// }
//
// func (client Client) GetRunning(c *gin.Context) {
// 	client.Log.Debugf("Entering")
// 	defer client.Log.Debugf("Exiting")
//
// 	gs, cnt, err := client.getFiltered(c, c.Param("status"), "", "", "", All)
//
// 	if err != nil {
// 		client.Log.Errorf(err.Error())
// 		c.Redirect(http.StatusSeeOther, homePath)
// 		c.Abort()
// 	}
// 	withGamers(withCount(c, cnt), gs)
// }
