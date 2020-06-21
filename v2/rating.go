package sn

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strconv"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/SlothNinja/glicko"
	"github.com/SlothNinja/log"
	"github.com/SlothNinja/restful"
	"github.com/SlothNinja/user/v2"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/iterator"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2"
	taskspb "google.golang.org/genproto/googleapis/cloud/tasks/v2"
)

func CurrentRatingsFrom(c *gin.Context) (rs CurrentRatings) {
	rs, _ = c.Value(currentRatingsKey).(CurrentRatings)
	return
}

func ProjectedFrom(c *gin.Context) (r *Rating) {
	r, _ = c.Value(projectedKey).(*Rating)
	return
}

func (cl Client) AddRoutes(prefix string, engine *gin.Engine) *gin.Engine {
	g1 := engine.Group(prefix + "s")
	g1.POST("/userUpdate/:uid/:type", cl.updateUser)

	g1.GET("/update/:type", cl.Update)

	g1.GET("/show/:type", cl.Index)

	g1.POST("/show/:type/json", cl.JSONFilteredAction)

	return engine
}

// Ratings
type Ratings []*Rating
type Rating struct {
	Key *datastore.Key `datastore:"__key__"`
	Common
}

func (r *Rating) Load(ps []datastore.Property) error {
	return datastore.LoadStruct(r, ps)
}

func (r *Rating) Save() ([]datastore.Property, error) {
	t := time.Now()
	if r.CreatedAt.IsZero() {
		r.CreatedAt = t
	}
	r.UpdatedAt = t
	return datastore.SaveStruct(r)
}

func (r *Rating) LoadKey(k *datastore.Key) error {
	r.Key = k
	return nil
}

type CurrentRatings []*CurrentRating
type CurrentRating struct {
	Key *datastore.Key `datastore:"__key__"`
	Common
}

func (r *CurrentRating) Load(ps []datastore.Property) error {
	return datastore.LoadStruct(r, ps)
}

func (r *CurrentRating) Save() ([]datastore.Property, error) {
	t := time.Now()
	if r.CreatedAt.IsZero() {
		r.CreatedAt = t
	}
	r.UpdatedAt = t
	return datastore.SaveStruct(r)
}

func (r *CurrentRating) LoadKey(k *datastore.Key) error {
	r.Key = k
	return nil
}

type Common struct {
	generated bool
	Type      Type
	R         float64
	RD        float64
	Low       float64
	High      float64
	Leader    bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (r *CurrentRating) Rank() *glicko.Rank {
	return &glicko.Rank{
		R:  r.R,
		RD: r.RD,
	}
}

func NewRating(id int64, pk *datastore.Key, t Type, params ...float64) *Rating {
	r, rd := defaultR, defaultRD
	if len(params) == 2 {
		r, rd = params[0], params[1]
	}

	rating := new(Rating)
	rating.Key = datastore.IDKey(rKind, id, pk)
	rating.R = r
	rating.RD = rd
	rating.Low = r - (2.0 * rd)
	rating.High = r + (2.0 * rd)
	rating.Type = t
	return rating
}

func NewCurrent(pk *datastore.Key, t Type, params ...float64) *CurrentRating {
	r, rd := defaultR, defaultRD
	if len(params) == 2 {
		r, rd = params[0], params[1]
	}

	rating := new(CurrentRating)
	rating.Key = datastore.NameKey(crKind, t.SString(), pk)
	rating.R = r
	rating.RD = rd
	rating.Low = r - (2.0 * rd)
	rating.High = r + (2.0 * rd)
	rating.Type = t
	return rating
}

const (
	defaultR  float64 = 1500
	defaultRD float64 = 350
)

const (
	rKind  = "Rating"
	crKind = "CurrentRating"
)

func singleError(e error) error {
	if e == nil {
		return e
	}
	if me, ok := e.(datastore.MultiError); ok {
		return me[0]
	}
	return e
}

// Get Current Rating for Type and user associated with uKey
func (cl Client) GetRating(c *gin.Context, uKey *datastore.Key, t Type) (*CurrentRating, error) {
	ratings, err := cl.GetMulti(c, []*datastore.Key{uKey}, t)
	return ratings[0], singleError(err)
}

func (cl Client) GetMulti(c *gin.Context, uKeys []*datastore.Key, t Type) (CurrentRatings, error) {
	l := len(uKeys)
	ratings := make(CurrentRatings, l)
	ks := make([]*datastore.Key, l)
	for i, uKey := range uKeys {
		ratings[i] = NewCurrent(uKey, t)
		ks[i] = ratings[i].Key
	}

	err := cl.DS.GetMulti(c, ks, ratings)
	if err == nil {
		return ratings, nil
	}

	me := err.(datastore.MultiError)
	isNil := true
	for i := range uKeys {
		if me[i] == datastore.ErrNoSuchEntity {
			ratings[i].generated = true
			me[i] = nil
		} else {
			isNil = false
		}
	}
	if isNil {
		return ratings, nil
	} else {
		return ratings, me
	}
}

func (cl Client) GetAll(c *gin.Context, uKey *datastore.Key) (CurrentRatings, error) {
	l := len(Types)
	rs := make(CurrentRatings, l)
	ks := make([]*datastore.Key, l)
	for i, t := range Types {
		rs[i] = NewCurrent(uKey, t)
		ks[i] = rs[i].Key
	}

	err := cl.DS.GetMulti(c, ks, rs)
	if err == nil {
		return nil, err
	}

	merr, ok := err.(datastore.MultiError)
	if !ok {
		return nil, err
	}

	enil := true
	for i, e := range merr {
		if e == datastore.ErrNoSuchEntity {
			rs[i].generated = true
			merr[i] = nil
		} else if e != nil {
			enil = false
		}
	}

	if enil {
		return rs, nil
	}
	return nil, merr
}

func (cl Client) GetFor(c *gin.Context, t Type) (CurrentRatings, error) {
	q := datastore.NewQuery(crKind).
		Ancestor(user.RootKey()).
		Filter("Type=", int(t)).
		Order("-Low")

	var rs CurrentRatings
	_, err := cl.DS.GetAll(c, q, &rs)
	if err != nil {
		return nil, err
	}
	return rs, nil
}

func (rs CurrentRatings) Projected(c *gin.Context, cm ContestMap) (CurrentRatings, error) {
	ratings := make(CurrentRatings, len(rs))
	for i, r := range rs {
		var err error
		if ratings[i], err = r.Projected(cm[r.Type]); err != nil {
			return nil, err
		}
	}
	return ratings, nil
}

func (r *CurrentRating) Projected(cs []*Contest) (*CurrentRating, error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	l := len(cs)
	if l == 0 && r.generated {
		return r, nil
	}

	gcs := make(glicko.Contests, l)
	for i, c := range cs {
		gcs[i] = glicko.NewContest(c.Outcome, c.R, c.RD)
	}

	rating, err := glicko.UpdateRating(r.Rank(), gcs)
	if err != nil {
		return nil, err
	}

	return NewCurrent(r.Key.Parent, r.Type, rating.R, rating.RD), nil
}

func (r *CurrentRating) Generated() bool {
	return r.generated
}

func (cl Client) Index(c *gin.Context) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	t := ToType[c.Param("type")]
	c.HTML(http.StatusOK, "rating/index", gin.H{
		"Type":      t,
		"Heading":   "Ratings: " + t.String(),
		"Types":     Types,
		"Context":   c,
		"VersionID": VersionID(),
		"CUser":     user.CurrentFrom(c),
	})
}

func getAllQuery() *datastore.Query {
	return datastore.NewQuery(crKind).Ancestor(user.RootKey())
}

func (cl Client) getFiltered(c *gin.Context, t Type, leader bool, offset, limit int32) (CurrentRatings, int64, error) {
	q := getAllQuery()

	if leader {
		q = q.Filter("Leader=", true)
	}

	if t != NoType {
		q = q.Filter("Type=", int(t))
	}

	var cnt int64
	count, err := cl.DS.Count(c, q)
	if err != nil {
		return nil, 0, err
	}
	cnt = int64(count)

	q = q.Offset(int(offset)).
		Limit(int(limit)).
		Order("-Low")

	var rs CurrentRatings
	_, err = cl.DS.GetAll(c, q, &rs)
	if err != nil {
		return nil, 0, err
	}

	return rs, cnt, err
}

func (cl Client) getUsers(c *gin.Context, rs CurrentRatings) ([]*user.User, error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	l := len(rs)
	us := make([]*user.User, l)
	ks := make([]*datastore.Key, l)
	for i := range rs {
		us[i] = user.New(0)
		ks[i] = rs[i].Key.Parent
	}

	err := cl.DS.GetMulti(c, ks, us)
	if err != nil {
		return nil, err
	}
	return us, nil
}

func (cl Client) getProjected(c *gin.Context, rs CurrentRatings) (CurrentRatings, error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	ps := make(CurrentRatings, len(rs))
	for i, r := range rs {
		uKey := r.Key.Parent

		cs, err := cl.UnappliedFor(c, uKey, r.Type)
		if err != nil {
			return nil, err
		}

		ps[i], err = r.Projected(cs)
		if err != nil {
			return nil, err
		}

		if r.generated && r.generated && len(cs) == 0 {
			ps[i].generated = true
		}
	}
	return ps, nil
}

func (cl Client) GetProjectedRating(c *gin.Context, ukey *datastore.Key, t Type) (*CurrentRating, error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	cs, err := cl.UnappliedFor(c, ukey, t)
	if err != nil {
		return nil, err
	}

	r, err := cl.GetRating(c, ukey, t)
	if err != nil {
		return nil, err
	}

	if r.generated && len(cs) == 0 {
		return r, nil
	}

	return r.Projected(cs)
}

// func (cl Client) For(c *gin.Context, ukey *datastore.Key, t Type) (*CurrentRating, error) {
// 	return cl.GetRating(c, ukey, t)
// }

func (cl Client) MultiFor(c *gin.Context, ukey *datastore.Key) (CurrentRatings, error) {
	return cl.GetAll(c, ukey)
}

func (cl Client) Update(c *gin.Context) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	t := ToType[c.Param("type")]
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	locationID := "us-central1"
	queueID := "default"

	q := user.AllQuery(c).
		KeysOnly()
	it := cl.DS.Run(c, q)

	for {
		k, err := it.Next(nil)
		if err == iterator.Done {
			break
		}

		if err != nil {
			log.Errorf(err.Error())
			c.AbortWithStatus(http.StatusInternalServerError)
		}

		if k.ID == 0 {
			continue
		}

		task, err := createTask(projectID, locationID, queueID, k.ID, t)
		if err != nil {
			log.Errorf("Task: %#v\nError: %v", task, err)
		}
	}
}

func (cl Client) updateUser(c *gin.Context) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	uid, err := strconv.ParseInt(c.Param("uid"), 10, 64)
	if err != nil {
		log.Errorf(err.Error())
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	t := ToType[c.Param("type")]

	u := user.New(uid)
	err = cl.DS.Get(c, u.Key, u)
	if err != nil {
		log.Errorf(err.Error())
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	r, err := cl.GetRating(c, u.Key, t)
	if err != nil {
		log.Errorf(err.Error())
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	cs, err := cl.UnappliedFor(c, u.Key, t)
	if err != nil {
		log.Errorf(err.Error())
		c.AbortWithStatus(http.StatusInternalServerError)
		return

	}

	p, err := r.Projected(cs)
	if err != nil {
		log.Errorf(err.Error())
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	if time.Since(time.Time(r.UpdatedAt)) < 504*time.Hour {
		log.Warningf("Did not update rating for user ID: %v", u.ID())
		log.Warningf("Rating updated %s ago.", time.Since(time.Time(r.UpdatedAt)))
		return
	}

	const threshold float64 = 200.0
	// Update leader value to indicate whether present on leader boards
	p.Leader = p.RD < threshold

	var (
		es []interface{}
		ks []*datastore.Key
	)

	if !p.Generated() {
		r := NewRating(0, p.Key.Parent, p.Type, p.R, p.RD)
		es = append(es, p, r)
		ks = append(ks, p.Key, r.Key)

	}

	for _, c := range cs {
		c.Applied = true
		es = append(es, c)
		ks = append(ks, c.Key)
	}

	_, err = cl.DS.RunInTransaction(c, func(tx *datastore.Transaction) error {
		_, err := tx.PutMulti(ks, es)
		return err
	})
	if err != nil {
		log.Errorf(err.Error())
		c.AbortWithStatus(http.StatusInternalServerError)
	}
}

func (cl Client) Fetch(c *gin.Context) {
	if CurrentRatingsFrom(c) != nil {
		return
	}

	u := user.Fetched(c)
	if u == nil {
		restful.AddErrorf(c, "Unable to get ratings.")
		c.Redirect(http.StatusSeeOther, homePath)
		return
	}

	rs, err := cl.MultiFor(c, u.Key)
	if err != nil {
		restful.AddErrorf(c, err.Error())
		c.Redirect(http.StatusSeeOther, homePath)
		return
	}
	c.Set(currentRatingsKey, rs)
}

func Fetched(c *gin.Context) CurrentRatings {
	return CurrentRatingsFrom(c)
}

func (cl Client) FetchProjected(c *gin.Context) {
	if ProjectedFrom(c) != nil {
		return
	}

	rs := Fetched(c)
	if rs == nil {
		restful.AddErrorf(c, "Unable to get projected ratings")
		c.Redirect(http.StatusSeeOther, homePath)
		return
	}

	cm, err := cl.Unapplied(c, user.Fetched(c).Key)
	if err != nil {
		restful.AddErrorf(c, err.Error())
		c.Redirect(http.StatusSeeOther, homePath)
		return
	}

	if pr, err := rs.Projected(c, cm); err != nil {
		restful.AddErrorf(c, err.Error())
		c.Redirect(http.StatusSeeOther, homePath)
	} else {
		c.Set(projectedKey, pr)
	}
}

func Projected(c *gin.Context) (pr Ratings) {
	pr, _ = c.Value("Projected").(Ratings)
	return
}

type jRating struct {
	Type template.HTML `json:"type"`
	R    float64       `json:"r"`
	RD   float64       `json:"rd"`
	Low  float64       `json:"low"`
	High float64       `json:"high"`
}

type jCombined struct {
	Rank      int           `json:"rank"`
	Gravatar  template.HTML `json:"gravatar"`
	Name      template.HTML `json:"name"`
	Type      template.HTML `json:"type"`
	Current   template.HTML `json:"current"`
	Projected template.HTML `json:"projected"`
}

func (cl Client) JSONIndexAction(c *gin.Context) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	uid, err := strconv.ParseInt(c.Param("uid"), 10, 64)
	if err != nil {
		log.Errorf("rating#JSONIndexAction BySID Error: %s", err)
		c.Redirect(http.StatusSeeOther, homePath)
		return
	}

	u := user.New(uid)
	err = cl.DS.Get(c, u.Key, u)
	if err != nil {
		log.Errorf("rating#JSONIndexAction unable to find user for uid: %d", uid)
		c.Redirect(http.StatusSeeOther, homePath)
		return
	}

	rs, err := cl.MultiFor(c, u.Key)
	if err != nil {
		log.Errorf("rating#JSONIndexAction MultiFor Error: %s", err)
		c.Redirect(http.StatusSeeOther, homePath)
		return
	}

	ps, err := cl.getProjected(c, rs)
	if err != nil {
		log.Errorf("rating#getProjected Error: %s", err)
		c.Redirect(http.StatusSeeOther, homePath)
		return
	}

	if data, err := singleUser(c, u, rs, ps); err != nil {
		c.JSON(http.StatusOK, fmt.Sprintf("%v", err))
	} else {
		c.JSON(http.StatusOK, data)
	}
}

func (cl Client) JSONFilteredAction(c *gin.Context) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	t := GetType(c)

	var offset, limit int32 = 0, -1
	if o, err := strconv.ParseInt(c.PostForm("start"), 10, 64); err == nil && o >= 0 {
		offset = int32(o)
	}

	if l, err := strconv.ParseInt(c.PostForm("length"), 10, 64); err == nil {
		limit = int32(l)
	}

	rs, cnt, err := cl.getFiltered(c, t, true, offset, limit)
	if err != nil {
		log.Errorf("rating#getFiltered Error: %s", err)
		return
	}

	us, err := cl.getUsers(c, rs)
	if err != nil {
		log.Errorf("rating#getUsers Error: %s", err)
		return
	}

	ps, err := cl.getProjected(c, rs)
	if err != nil {
		log.Errorf("rating#getProjected Error: %s", err)
		return
	}

	data, err := toCombined(c, us, rs, ps, offset, cnt)
	if err != nil {
		log.Errorf("toCombined error: %v", err)
		c.JSON(http.StatusOK, fmt.Sprintf("%v", err))
		return
	}
	c.JSON(http.StatusOK, data)
}

type jCombinedRatingsIndex struct {
	Data            []*jCombined `json:"data"`
	Draw            int          `json:"draw"`
	RecordsTotal    int          `json:"recordsTotal"`
	RecordsFiltered int          `json:"recordsFiltered"`
}

func (r *CurrentRating) String() string {
	return fmt.Sprintf("%.f (%.f : %.f)", r.Low, r.R, r.RD)
}

func singleUser(c *gin.Context, u *user.User, rs, ps CurrentRatings) (table *jCombinedRatingsIndex, err error) {
	log.Debugf("Entering")
	defer log.Debugf("Exiting")

	table = new(jCombinedRatingsIndex)
	l1, l2 := len(rs), len(ps)
	if l1 != l2 {
		err = fmt.Errorf("Length mismatch between ratings and projected ratings l1: %d l2: %d.", l1, l2)
		return
	}

	table.Data = make([]*jCombined, 0)
	for i, r := range rs {
		if p := ps[i]; !p.generated {
			table.Data = append(table.Data, &jCombined{
				Gravatar:  user.Gravatar(u),
				Name:      u.Link(),
				Type:      template.HTML(r.Type.String()),
				Current:   template.HTML(r.String()),
				Projected: template.HTML(p.String()),
			})
		}
	}

	var draw int
	if draw, err = strconv.Atoi(c.PostForm("draw")); err != nil {
		log.Errorf("strconv.Atoi error: %v", err)
		return
	}

	table.Draw = draw
	table.RecordsTotal = l1
	table.RecordsFiltered = l2
	return
}
func toCombined(c *gin.Context, us []*user.User, rs, ps CurrentRatings, o int32, cnt int64) (*jCombinedRatingsIndex, error) {
	table := new(jCombinedRatingsIndex)
	l1, l2 := len(rs), len(ps)
	if l1 != l2 {
		return nil, fmt.Errorf("Length mismatch between ratings and projected ratings l1: %d l2: %d.", l1, l2)
	}
	table.Data = make([]*jCombined, 0)
	for i, r := range rs {
		if !r.generated {
			table.Data = append(table.Data, &jCombined{
				Rank:      i + int(o) + 1,
				Gravatar:  user.Gravatar(us[i]),
				Name:      us[i].Link(),
				Type:      template.HTML(r.Type.String()),
				Current:   template.HTML(r.String()),
				Projected: template.HTML(ps[i].String()),
			})
		}
	}

	if draw, err := strconv.Atoi(c.PostForm("draw")); err != nil {
		return nil, err
	} else {
		table.Draw = draw
	}

	table.RecordsTotal = int(cnt)
	table.RecordsFiltered = int(cnt)
	return table, nil
}

// func (cl Client) IncreaseFor(c *gin.Context, ukey *datastore.Key, t Type, cs []*Contest) (*CurrentRating, *CurrentRating, error) {
// 	log.Debugf("Entering")
// 	defer log.Debugf("Exiting")
//
// 	ucs, err := cl.UnappliedFor(c, ukey, t)
// 	if err != nil {
// 		return nil, nil, err
// 	}
//
// 	r, err := cl.GetRating(c, ukey, t)
// 	if err != nil {
// 		return nil, nil, err
// 	}
//
// 	cr, err := r.Projected(c, ucs)
// 	if err != nil {
// 		return nil, nil, err
// 	}
//
// 	nr, err := r.Projected(c, append(ucs, filterContestsFor(cs, ukey)...))
// 	return cr, nr, err
// }

func filterContestsFor(cs []*Contest, pk *datastore.Key) (fcs []*Contest) {
	for _, c := range cs {
		if c.Key.Parent.Equal(pk) {
			fcs = append(fcs, c)
		}
	}
	return
}

// createTask creates a new task in your App Engine queue.
func createTask(projectID, locationID, queueID string, uid int64, t Type) (*taskspb.Task, error) {
	// Create a new Cloud Tasks client instance.
	// See https://godoc.org/cloud.google.com/go/cloudtasks/apiv2
	ctx := context.Background()
	cl, err := cloudtasks.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("NewClient: %v", err)
	}
	defer cl.Close()

	// Build the Task queue path.
	queuePath := fmt.Sprintf("projects/%s/locations/%s/queues/%s", projectID, locationID, queueID)

	// Build the Task payload.
	// https://godoc.org/google.golang.org/genproto/googleapis/cloud/tasks/v2#CreateTaskRequest
	req := &taskspb.CreateTaskRequest{
		Parent: queuePath,
		Task: &taskspb.Task{
			// https://godoc.org/google.golang.org/genproto/googleapis/cloud/tasks/v2#AppEngineHttpRequest
			MessageType: &taskspb.Task_AppEngineHttpRequest{
				AppEngineHttpRequest: &taskspb.AppEngineHttpRequest{
					HttpMethod:  taskspb.HttpMethod_POST,
					RelativeUri: fmt.Sprintf("/ratings/userUpdate/%d/%s", uid, t.IDString()),
				},
			},
		},
	}

	createdTask, err := cl.CreateTask(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("cloudtasks.CreateTask: %v", err)
	}

	return createdTask, nil
}
