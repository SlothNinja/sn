package sn

import (
	"fmt"
	"sort"

	"cloud.google.com/go/datastore"
	"cloud.google.com/go/firestore"
	"github.com/elliotchance/pie/v2"
	"github.com/gin-gonic/gin"
	elogo "github.com/kortemy/elo-go"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const eloKind = "Elo"
const historyKind = "History"
const defaultRating = 1500

type Elo struct {
	ID     UID
	Rating int
}

func newEloDefault(uid UID) Elo {
	return Elo{ID: uid, Rating: defaultRating}
}

func (cl Client) EloDocRef(uid UID) *firestore.DocumentRef {
	return cl.eloCollectionRef().Doc(fmt.Sprintf("%d", uid))
}

func (cl Client) eloCollectionRef() *firestore.CollectionRef {
	return cl.FS.Collection(eloKind)
}

func (cl Client) EloHistoryRef(uid UID) *firestore.CollectionRef {
	return cl.EloDocRef(uid).Collection(historyKind)
}

// func (e Elo) IncompleteKey() *datastore.Key {
// 	if e.Key == nil || e.Key.Parent == nil {
// 		return nil
// 	}
// 	return datastore.IncompleteKey(eloKind, e.Key.Parent)
// }

func eloCopy(elo Elo) *Elo {
	return &elo
}

type EloClient struct {
	*Client
}

// func NewEloClient(snClient *Client, prefix string) *EloClient {
// 	client := &EloClient{
// 		Client: snClient,
// 	}
// 	return client
// 	// return client.addRoutes(prefix)
// }
//
// func (cl *EloClient) GetMulti(c *gin.Context, uids []UID) ([]*Elo, error) {
// 	ks := pie.Map(uids, func(uid UID) *datastore.Key { return newCurrentEloKey(uid) })
// 	elos := make([]*Elo, len(uids))
// 	err := cl.DS.GetMulti(c, ks, elos)
// 	return filterNoSuchEntity(elos, uids, err)
// }
//
// func (cl *EloClient) Get(c *gin.Context, uid UID) (*Elo, error) {
// 	elos, err := cl.GetMulti(c, []UID{uid})
// 	elos, err = filterNoSuchEntity(elos, []UID{uid}, err)
// 	if err != nil {
// 		return nil, err
// 	}
// 	if len(elos) != 1 {
// 		return nil, fmt.Errorf("incorrect number of ratings returned")
// 	}
// 	return pie.First(elos), nil
// }

type Results map[int][]*datastore.Key

// range through may has not guaranteed order
// places provides ordered list of map keys so map may be tranversed in order
func (rs Results) places() []int {
	var places []int
	for place := range rs {
		places = append(places, place)
	}
	sort.Slice(places, func(p1, p2 int) bool { return p1 < p2 })
	return places
}

func merrFilter(err error, ignore ...error) error {
	if err == nil {
		return err
	}

	merr, ok := err.(datastore.MultiError)
	if !ok {
		return err
	}

	for _, err1 := range merr {
		if !pie.Any(ignore, func(err2 error) bool { return err1 == err2 }) {
			return err
		}
	}
	return nil
}

type eloMap map[UID]Elo
type PlacesMap map[UID]int

const notFound = 0

func updateEloFor(uid1 UID, elos eloMap, places PlacesMap) int {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	var delta int
	elo := elogo.NewElo()
	for uid2 := range elos {
		if uid1 == uid2 {
			continue
		}
		score := 0.0
		if places[uid1] == places[uid2] {
			score = 0.5
		}
		// places are essentially first, second, third, etc.
		// thus, a lower place indicates better performance
		if places[uid1] < places[uid2] {
			score = 1
		}
		delta += elo.RatingDelta(elos[uid1].Rating, elos[uid2].Rating, score)
	}
	return elos[uid1].Rating + delta
}

func (cl Client) SaveElosIn(tx *firestore.Transaction, elos []Elo) error {
	for _, elo := range elos {
		if err := tx.Set(cl.EloDocRef(elo.ID), elo); err != nil {
			return err
		}
		if err := tx.Create(cl.EloHistoryRef(elo.ID).NewDoc(), elo); err != nil {
			return err
		}
	}
	return nil
}

// // if merely missing from datastore, provide default elo entity
// func filterNoSuchEntity(elos []*Elo, uids []UID, err error) ([]*Elo, error) {
// 	if err == nil {
// 		return elos, err
// 	}
//
// 	// if no Elo entity associated with user, create initial Elo entity for user
//
// 	if l1, l2 := len(elos), len(uids); l1 != l2 {
// 		return elos, fmt.Errorf("len(elos):%d len(uids):%d: must be same length", l1, l2)
// 	}
//
// 	me, ok := err.(datastore.MultiError)
// 	if !ok {
// 		return nil, err
// 	}
//
// 	for i, e := range me {
//
// 		if e == datastore.ErrNoSuchEntity {
// 			elos[i] = newEloDefault(uids[i])
// 		} else if e != nil {
// 			return nil, err
// 		}
// 	}
// 	return elos, nil
// }

// // Update pulls current Elo from db and provides rating updates and deltas per results for users associated with uids.
// // Returns ratings, updates, and current Elo (not updated) in same order as supplied uids
// func (cl *EloClient) Update(c *gin.Context, uids []UID, places PlacesMap) ([]*Elo, []*Elo, error) {
//
// 	oldElos, err := cl.GetMulti(c, uids)
//
// 	oldElos, err = filterNoSuchEntity(oldElos, uids, err)
//
// 	if err != nil {
// 		return nil, nil, err
// 	}
//
// 	newElos := make([]*Elo, len(uids))
// 	eloMap := make(eloMap, len(oldElos))
// 	for i, elo := range oldElos {
// 		eloMap[uids[i]] = elo
// 		newElos[i] = eloCopy(*elo)
// 	}
//
// 	for i, uid := range uids {
// 		newElos[i].Rating = updateEloFor(uid, eloMap, places)
// 	}
//
// 	return oldElos, newElos, nil
// }

// Update pulls current Elo from db and provides rating updates and deltas per results for users associated with uids.
// Returns ratings, updates, and current Elo (not updated) in same order as supplied uids
func (cl Client) UpdateElo(c *gin.Context, uids []UID, places PlacesMap) ([]Elo, []Elo, error) {
	Debugf(msgEnter)
	defer Debugf(msgExit)

	oldElos := make([]Elo, len(uids))
	for i, uid := range uids {
		snap, err := cl.EloDocRef(uid).Get(c)
		if status.Code(err) == codes.NotFound {
			oldElos[i] = newEloDefault(uid)
			continue
		}
		if err != nil {
			return nil, nil, err
		}
		var elo Elo
		if err := snap.DataTo(&elo); err != nil {
			return nil, nil, err
		}
		oldElos[i] = elo
	}

	Debugf("oldElos: %#v", oldElos)

	newElos := make([]Elo, len(uids))
	eloMap := make(eloMap, len(oldElos))
	for i, elo := range oldElos {
		elo2 := elo
		eloMap[uids[i]] = elo2
		newElos[i] = elo
	}

	for i, uid := range uids {
		newElos[i].Rating = updateEloFor(uid, eloMap, places)
		Debugf("newElos: %#v", newElos)
		Debugf("eloMap: %#v", eloMap)
	}

	return oldElos, newElos, nil
}

// const (
// 	currentRatingsKey = "CurrentRatings"
// 	projectedKey      = "Projected"
// 	gravSize          = "80"
// )
//
// func CurrentRatingsFrom(c *gin.Context) (rs CurrentRatings) {
// 	rs, _ = c.Value(currentRatingsKey).(CurrentRatings)
// 	return
// }
//
// func ProjectedFrom(c *gin.Context) (r *Rating) {
// 	r, _ = c.Value(projectedKey).(*Rating)
// 	return
// }
//
// func (client *RatingClient) addRoutes(prefix string) *RatingClient {
// 	g1 := client.Router.Group(prefix + "s")
// 	g1.POST("/userUpdate/:uid/:type", client.updateUser)
//
// 	g1.GET("/update/:type", client.Update)
//
// 	g1.GET("/show/:type", client.RatingsIndex)
//
// 	g1.POST("/show/:type/json", client.JSONFilteredAction)
// 	return client
// }
//
// // Ratings
// type Ratings []*Rating
// type Rating struct {
// 	Key *datastore.Key `datastore:"__key__"`
// 	Common
// }
//
// func (r *Rating) Load(ps []datastore.Property) error {
// 	return datastore.LoadStruct(r, ps)
// }
//
// func (r *Rating) Save() ([]datastore.Property, error) {
// 	t := time.Now()
// 	if r.CreatedAt.IsZero() {
// 		r.CreatedAt = t
// 	}
// 	r.UpdatedAt = t
// 	return datastore.SaveStruct(r)
// }
//
// func (r *Rating) LoadKey(k *datastore.Key) error {
// 	r.Key = k
// 	return nil
// }
//
// type CurrentRatings []*CurrentRating
// type CurrentRating struct {
// 	Key *datastore.Key `datastore:"__key__"`
// 	Common
// }
//
// func (r *CurrentRating) Load(ps []datastore.Property) error {
// 	return datastore.LoadStruct(r, ps)
// }
//
// func (r *CurrentRating) Save() ([]datastore.Property, error) {
// 	t := time.Now()
// 	if r.CreatedAt.IsZero() {
// 		r.CreatedAt = t
// 	}
// 	r.UpdatedAt = t
// 	return datastore.SaveStruct(r)
// }
//
// func (r *CurrentRating) LoadKey(k *datastore.Key) error {
// 	r.Key = k
// 	return nil
// }
//
// type Common struct {
// 	generated bool
// 	Type      Type      `json:"type"`
// 	R         float64   `json:"r"`
// 	RD        float64   `json:"rd"`
// 	Low       float64   `json:"low"`
// 	High      float64   `json:"high"`
// 	Leader    bool      `json:"leader"`
// 	CreatedAt time.Time `json;"createdAt"`
// 	UpdatedAt time.Time `json:"updatedAt"`
// }
//
// func (r *CurrentRating) Rank() *glicko.Rank {
// 	return &glicko.Rank{
// 		R:  r.R,
// 		RD: r.RD,
// 	}
// }
//
// func NewRating(c *gin.Context, id int64, pk *datastore.Key, t Type, params ...float64) *Rating {
// 	r, rd := defaultR, defaultRD
// 	if len(params) == 2 {
// 		r, rd = params[0], params[1]
// 	}
//
// 	rating := new(Rating)
// 	rating.Key = datastore.IDKey(rKind, id, pk)
// 	rating.R = r
// 	rating.RD = rd
// 	rating.Low = r - (2.0 * rd)
// 	rating.High = r + (2.0 * rd)
// 	rating.Type = t
// 	return rating
// }
//
// func NewCurrent(pk *datastore.Key, t Type, params ...float64) *CurrentRating {
// 	r, rd := defaultR, defaultRD
// 	if len(params) == 2 {
// 		r, rd = params[0], params[1]
// 	}
//
// 	rating := new(CurrentRating)
// 	rating.Key = datastore.NameKey(crKind, string(t), pk)
// 	rating.R = r
// 	rating.RD = rd
// 	rating.Low = r - (2.0 * rd)
// 	rating.High = r + (2.0 * rd)
// 	rating.Type = t
// 	return rating
// }
//
// const (
// 	defaultR  float64 = 1500
// 	defaultRD float64 = 350
// )
//
// const (
// 	rKind  = "Rating"
// 	crKind = "CurrentRating"
// )
//
// func singleError(e error) error {
// 	if e == nil {
// 		return e
// 	}
// 	if me, ok := e.(datastore.MultiError); ok {
// 		return me[0]
// 	}
// 	return e
// }
//
// // Get Current Rating for Type and user associated with uKey
// func (client *RatingClient) Get(c *gin.Context, uKey *datastore.Key, t Type) (*CurrentRating, error) {
// 	ratings, err := client.GetMulti(c, []*datastore.Key{uKey}, t)
// 	return ratings[0], singleError(err)
// }
//
// func (client *RatingClient) GetMulti(c *gin.Context, uKeys []*datastore.Key, t Type) (CurrentRatings, error) {
// 	l := len(uKeys)
// 	ratings := make(CurrentRatings, l)
// 	ks := make([]*datastore.Key, l)
// 	for i, uKey := range uKeys {
// 		ratings[i] = NewCurrent(uKey, t)
// 		ks[i] = ratings[i].Key
// 	}
//
// 	err := client.DS.GetMulti(c, ks, ratings)
// 	if err == nil {
// 		return ratings, nil
// 	}
//
// 	me := err.(datastore.MultiError)
// 	isNil := true
// 	for i := range uKeys {
// 		if me[i] == datastore.ErrNoSuchEntity {
// 			ratings[i].generated = true
// 			me[i] = nil
// 		} else {
// 			isNil = false
// 		}
// 	}
// 	if isNil {
// 		return ratings, nil
// 	} else {
// 		return ratings, me
// 	}
// }
//
// func (client *RatingClient) GetAll(c *gin.Context, uKey *datastore.Key) (CurrentRatings, error) {
// 	l := len(types())
// 	rs := make(CurrentRatings, l)
// 	ks := make([]*datastore.Key, l)
// 	for i, t := range types() {
// 		rs[i] = NewCurrent(uKey, t)
// 		ks[i] = rs[i].Key
// 	}
//
// 	err := client.DS.GetMulti(c, ks, rs)
// 	if err == nil {
// 		return nil, err
// 	}
//
// 	merr, ok := err.(datastore.MultiError)
// 	if !ok {
// 		return nil, err
// 	}
//
// 	enil := true
// 	for i, e := range merr {
// 		if e == datastore.ErrNoSuchEntity {
// 			rs[i].generated = true
// 			merr[i] = nil
// 		} else if e != nil {
// 			enil = false
// 		}
// 	}
//
// 	if enil {
// 		return rs, nil
// 	}
// 	return nil, merr
// }
//
// func (client *RatingClient) GetFor(c *gin.Context, t Type) (CurrentRatings, error) {
// 	q := datastore.NewQuery(crKind).
// 		Ancestor(user.RootKey()).
// 		FilterField("Type", "=", string(t)).
// 		Order("-Low")
//
// 	var rs CurrentRatings
// 	_, err := client.DS.GetAll(c, q, &rs)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return rs, nil
// }
//
// func (rs CurrentRatings) Projected(c *gin.Context, cm ContestMap) (CurrentRatings, error) {
// 	ratings := make(CurrentRatings, len(rs))
// 	for i, r := range rs {
// 		var err error
// 		if ratings[i], err = r.Projected(cm[r.Type]); err != nil {
// 			return nil, err
// 		}
// 	}
// 	return ratings, nil
// }
//
// func (r *CurrentRating) Projected(cs []*Contest) (*CurrentRating, error) {
// 	log.Debugf(msgEnter)
// 	defer log.Debugf(msgExit)
//
// 	l := len(cs)
// 	if l == 0 && r.generated {
// 		return r, nil
// 	}
//
// 	gcs := make(glicko.Contests, l)
// 	for i, c := range cs {
// 		gcs[i] = glicko.NewContest(c.Outcome, c.R, c.RD)
// 	}
//
// 	rating, err := glicko.UpdateRating(r.Rank(), gcs)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	return NewCurrent(r.Key.Parent, r.Type, rating.R, rating.RD), nil
// }
//
// func (r *CurrentRating) Generated() bool {
// 	return r.generated
// }
//
// func (client *RatingClient) RatingsIndex(c *gin.Context) {
// 	client.Log.Debugf(msgEnter)
// 	defer client.Log.Debugf(msgExit)
//
// 	cu, err := client.User.Current(c)
// 	if err != nil {
// 		client.Log.Debugf(err.Error())
// 	}
// 	t := ToType(c.Param("type"))
// 	c.HTML(http.StatusOK, "rating/index", gin.H{
// 		"Type":      t,
// 		"Heading":   "Ratings: " + t.String(),
// 		"Types":     types(),
// 		"Context":   c,
// 		"VersionID": VersionID(),
// 		"CUser":     cu,
// 	})
// }
//
// func getAllCurrentRatingsQuery(c *gin.Context) *datastore.Query {
// 	return datastore.NewQuery(crKind).Ancestor(user.RootKey())
// }
//
// func (client *RatingClient) getCurrentRatingsFiltered(c *gin.Context, t Type, leader bool, offset, limit int32) (CurrentRatings, int64, error) {
// 	q := getAllCurrentRatingsQuery(c)
//
// 	if leader {
// 		q = q.FilterField("Leader", "=", true)
// 	}
//
// 	if t != NoType {
// 		q = q.FilterField("Type", "=", string(t))
// 	}
//
// 	var cnt int64
// 	count, err := client.DS.Count(c, q)
// 	if err != nil {
// 		return nil, 0, err
// 	}
// 	cnt = int64(count)
//
// 	q = q.Offset(int(offset)).
// 		Limit(int(limit)).
// 		Order("-Low")
//
// 	var rs CurrentRatings
// 	_, err = client.DS.GetAll(c, q, &rs)
// 	if err != nil {
// 		return nil, 0, err
// 	}
//
// 	return rs, cnt, err
// }
//
// func (client *RatingClient) getUsers(c *gin.Context, rs CurrentRatings) (user.Users, error) {
// 	client.Log.Debugf(msgEnter)
// 	defer client.Log.Debugf(msgExit)
//
// 	l := len(rs)
// 	if l == 0 {
// 		return nil, nil
// 	}
//
// 	ids := make([]int64, len(rs))
// 	for i := range rs {
// 		ids[i] = rs[i].Key.Parent.ID
// 	}
//
// 	us, err := client.User.GetMulti(c, ids)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return us, nil
// }
//
// func (client *RatingClient) getProjected(c *gin.Context, rs CurrentRatings) (CurrentRatings, error) {
// 	client.Log.Debugf(msgEnter)
// 	defer client.Log.Debugf(msgExit)
//
// 	ps := make(CurrentRatings, len(rs))
// 	for i, r := range rs {
// 		uKey := r.Key.Parent
//
// 		cs, err := client.Contest.UnappliedFor(c, uKey, r.Type)
// 		if err != nil {
// 			return nil, err
// 		}
//
// 		ps[i], err = r.Projected(cs)
// 		if err != nil {
// 			return nil, err
// 		}
//
// 		if r.generated && len(cs) == 0 {
// 			ps[i].generated = true
// 		}
// 	}
// 	return ps, nil
// }
//
// func (client *RatingClient) projected(c *gin.Context, ukey *datastore.Key, t Type, cs ...*Contest) (*CurrentRating, error) {
// 	client.Log.Debugf(msgEnter)
// 	defer client.Log.Debugf(msgExit)
//
// 	ucs, err := client.Contest.UnappliedFor(c, ukey, t)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	ucs = append(ucs, cs...)
//
// 	r, err := client.Get(c, ukey, t)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	if r.generated && len(ucs) == 0 {
// 		return r, nil
// 	}
//
// 	return r.Projected(ucs)
// }
//
// func (client *RatingClient) GetProjected(c *gin.Context, ukey *datastore.Key, t Type) (*CurrentRating, error) {
// 	client.Log.Debugf(msgEnter)
// 	defer client.Log.Debugf(msgExit)
//
// 	return client.projected(c, ukey, t)
// }
//
// func (client *RatingClient) GetProjectedWith(c *gin.Context, ukey *datastore.Key, t Type, cs []*Contest) (*CurrentRating, error) {
// 	client.Log.Debugf(msgEnter)
// 	defer client.Log.Debugf(msgExit)
//
// 	return client.projected(c, ukey, t, cs...)
// }
//
// func (client *RatingClient) For(c *gin.Context, u *user.User, t Type) (*CurrentRating, error) {
// 	return client.Get(c, u.Key, t)
// }
//
// func (client *RatingClient) MultiFor(c *gin.Context, u *user.User) (CurrentRatings, error) {
// 	return client.GetAll(c, u.Key)
// }
//
// func (client *RatingClient) getLocationID() string {
// 	locationID := os.Getenv("LOCATION_ID")
// 	if locationID != "" {
// 		return locationID
// 	}
// 	client.Log.Warningf("LOCATION_ID environment variable not set -- using default")
// 	return "us-central1"
// }
//
// func (client *RatingClient) Update(c *gin.Context) {
// 	client.Log.Debugf(msgEnter)
// 	defer client.Log.Debugf(msgExit)
//
// 	t := ToType(c.Param("type"))
// 	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
// 	locationID := client.getLocationID()
// 	queueID := "default"
//
// 	q := user.AllQuery(c).
// 		KeysOnly()
// 	it := client.User.DS.Run(c, q)
//
// 	for {
// 		k, err := it.Next(nil)
// 		if err == iterator.Done {
// 			break
// 		}
//
// 		if err != nil {
// 			client.Log.Errorf(err.Error())
// 			c.AbortWithStatus(http.StatusInternalServerError)
// 		}
//
// 		if k.ID == 0 {
// 			continue
// 		}
//
// 		task, err := createTask(projectID, locationID, queueID, k.ID, t)
// 		if err != nil {
// 			client.Log.Errorf("Task: %#v\nError: %v", task, err)
// 		}
// 	}
// }
//
// func (client *RatingClient) updateUser(c *gin.Context) {
// 	client.Log.Debugf(msgEnter)
// 	defer client.Log.Debugf(msgExit)
//
// 	uid, err := strconv.ParseInt(c.Param("uid"), 10, 64)
// 	if err != nil {
// 		client.Log.Errorf(err.Error())
// 		c.AbortWithStatus(http.StatusNotFound)
// 		return
// 	}
//
// 	t := ToType(c.Param("type"))
//
// 	u, err := client.User.Get(c, uid)
// 	if err != nil {
// 		client.Log.Errorf(err.Error())
// 		c.AbortWithStatus(http.StatusNotFound)
// 		return
// 	}
//
// 	r, err := client.For(c, u, t)
// 	if err != nil {
// 		client.Log.Errorf(err.Error())
// 		c.AbortWithStatus(http.StatusNotFound)
// 		return
// 	}
//
// 	cs, err := client.Contest.UnappliedFor(c, u.Key, t)
// 	if err != nil {
// 		client.Log.Errorf(err.Error())
// 		c.AbortWithStatus(http.StatusInternalServerError)
// 		return
//
// 	}
//
// 	p, err := r.Projected(cs)
// 	if err != nil {
// 		client.Log.Errorf(err.Error())
// 		c.AbortWithStatus(http.StatusInternalServerError)
// 		return
// 	}
//
// 	if time.Since(time.Time(r.UpdatedAt)) < 504*time.Hour {
// 		client.Log.Warningf("Did not update rating for user ID: %v", u.ID())
// 		client.Log.Warningf("Rating updated %s ago.", time.Since(time.Time(r.UpdatedAt)))
// 		return
// 	}
//
// 	const threshold float64 = 200.0
// 	// Update leader value to indicate whether present on leader boards
// 	p.Leader = p.RD < threshold
//
// 	var (
// 		es []interface{}
// 		ks []*datastore.Key
// 	)
//
// 	if !p.Generated() {
// 		r := NewRating(c, 0, p.Key.Parent, p.Type, p.R, p.RD)
// 		es = append(es, p, r)
// 		ks = append(ks, p.Key, r.Key)
//
// 	}
//
// 	for _, c := range cs {
// 		c.Applied = true
// 		es = append(es, c)
// 		ks = append(ks, c.Key)
// 	}
//
// 	_, err = client.DS.RunInTransaction(c, func(tx *datastore.Transaction) error {
// 		_, err := tx.PutMulti(ks, es)
// 		return err
// 	})
// 	if err != nil {
// 		client.Log.Errorf(err.Error())
// 		c.AbortWithStatus(http.StatusInternalServerError)
// 	}
// }
//
// // func (client *RatingClient) Fetch(c *gin.Context) {
// // 	if CurrentRatingsFrom(c) != nil {
// // 		return
// // 	}
// //
// // 	u := user.Fetched(c)
// // 	if u == nil {
// // 		// AddErrorf(c, "Unable to get ratings.")
// // 		c.Redirect(http.StatusSeeOther, homePath)
// // 		return
// // 	}
// //
// // 	rs, err := client.MultiFor(c, u)
// // 	if err != nil {
// // 		// AddErrorf(c, err.Error())
// // 		c.Redirect(http.StatusSeeOther, homePath)
// // 		return
// // 	}
// // 	c.Set(currentRatingsKey, rs)
// // }
//
// // func Fetched(c *gin.Context) CurrentRatings {
// // 	return CurrentRatingsFrom(c)
// // }
// //
// // func (client *RatingClient) FetchProjected(c *gin.Context) {
// // 	if ProjectedFrom(c) != nil {
// // 		return
// // 	}
// //
// // 	rs := Fetched(c)
// // 	if rs == nil {
// // 		AddErrorf(c, "Unable to get projected ratings")
// // 		c.Redirect(http.StatusSeeOther, homePath)
// // 		return
// // 	}
// //
// // 	cm, err := client.Contest.Unapplied(c, user.Fetched(c).Key)
// // 	if err != nil {
// // 		AddErrorf(c, err.Error())
// // 		c.Redirect(http.StatusSeeOther, homePath)
// // 		return
// // 	}
// //
// // 	if pr, err := rs.Projected(c, cm); err != nil {
// // 		AddErrorf(c, err.Error())
// // 		c.Redirect(http.StatusSeeOther, homePath)
// // 	} else {
// // 		c.Set(projectedKey, pr)
// // 	}
// // }
//
// func Projected(c *gin.Context) (pr Ratings) {
// 	pr, _ = c.Value("Projected").(Ratings)
// 	return
// }
//
// // type jRating struct {
// // 	Type template.HTML `json:"type"`
// // 	R    float64       `json:"r"`
// // 	RD   float64       `json:"rd"`
// // 	Low  float64       `json:"low"`
// // 	High float64       `json:"high"`
// // }
//
// type jCombined struct {
// 	Rank      int           `json:"rank"`
// 	Gravatar  template.HTML `json:"gravatar"`
// 	Name      template.HTML `json:"name"`
// 	Type      template.HTML `json:"type"`
// 	Current   template.HTML `json:"current"`
// 	Projected template.HTML `json:"projected"`
// }
//
// func (client *RatingClient) JSONRatingsIndexAction(c *gin.Context) {
// 	client.Log.Debugf(msgEnter)
// 	defer client.Log.Debugf(msgExit)
//
// 	uid, err := strconv.ParseInt(c.Param("uid"), 10, 64)
// 	if err != nil {
// 		client.Log.Errorf("rating#JSONIndexAction BySID Error: %s", err)
// 		c.Redirect(http.StatusSeeOther, homePath)
// 		return
// 	}
//
// 	u, err := client.User.Get(c, uid)
// 	if err != nil {
// 		client.Log.Errorf("rating#JSONIndexAction unable to find user for uid: %d", uid)
// 		c.Redirect(http.StatusSeeOther, homePath)
// 		return
// 	}
//
// 	rs, err := client.MultiFor(c, u)
// 	if err != nil {
// 		client.Log.Errorf("rating#JSONIndexAction MultiFor Error: %s", err)
// 		c.Redirect(http.StatusSeeOther, homePath)
// 		return
// 	}
//
// 	ps, err := client.getProjected(c, rs)
// 	if err != nil {
// 		client.Log.Errorf("rating#getProjected Error: %s", err)
// 		c.Redirect(http.StatusSeeOther, homePath)
// 		return
// 	}
//
// 	if data, err := singleUser(c, u, rs, ps); err != nil {
// 		c.JSON(http.StatusOK, fmt.Sprintf("%v", err))
// 	} else {
// 		c.JSON(http.StatusOK, data)
// 	}
// }
//
// func (client *RatingClient) JSONFilteredAction(c *gin.Context) {
// 	client.Log.Debugf(msgEnter)
// 	defer client.Log.Debugf(msgExit)
//
// 	t := ToType(c.Param("type"))
//
// 	var offset, limit int32 = 0, -1
// 	if o, err := strconv.ParseInt(c.PostForm("start"), 10, 64); err == nil && o >= 0 {
// 		offset = int32(o)
// 	}
//
// 	if l, err := strconv.ParseInt(c.PostForm("length"), 10, 64); err == nil {
// 		limit = int32(l)
// 	}
//
// 	rs, cnt, err := client.getCurrentRatingsFiltered(c, t, true, offset, limit)
// 	if err != nil {
// 		client.Log.Errorf("rating#getFiltered Error: %s", err)
// 		return
// 	}
//
// 	us, err := client.getUsers(c, rs)
// 	if err != nil {
// 		client.Log.Errorf("rating#getUsers Error: %s", err)
// 		return
// 	}
//
// 	ps, err := client.getProjected(c, rs)
// 	if err != nil {
// 		client.Log.Errorf("rating#getProjected Error: %s", err)
// 		return
// 	}
//
// 	data, err := toCombined(c, us, rs, ps, offset, cnt)
// 	if err != nil {
// 		client.Log.Errorf("toCombined error: %v", err)
// 		c.JSON(http.StatusOK, fmt.Sprintf("%v", err))
// 		return
// 	}
// 	c.JSON(http.StatusOK, data)
// }
//
// type jCombinedRatingsIndex struct {
// 	Data            []*jCombined `json:"data"`
// 	Draw            int          `json:"draw"`
// 	RecordsTotal    int          `json:"recordsTotal"`
// 	RecordsFiltered int          `json:"recordsFiltered"`
// }
//
// func (r *CurrentRating) String() string {
// 	return fmt.Sprintf("%.f (%.f : %.f)", r.Low, r.R, r.RD)
// }
//
// func singleUser(c *gin.Context, u *user.User, rs, ps CurrentRatings) (table *jCombinedRatingsIndex, err error) {
// 	log.Debugf(msgEnter)
// 	defer log.Debugf(msgExit)
//
// 	table = new(jCombinedRatingsIndex)
// 	l1, l2 := len(rs), len(ps)
// 	if l1 != l2 {
// 		err = fmt.Errorf("length mismatch between ratings and projected ratings l1: %d l2: %d", l1, l2)
// 		return
// 	}
//
// 	table.Data = make([]*jCombined, 0)
// 	for i, r := range rs {
// 		if p := ps[i]; !p.generated {
// 			table.Data = append(table.Data, &jCombined{
// 				Gravatar:  user.Gravatar(u, gravSize),
// 				Name:      u.Link(),
// 				Type:      template.HTML(r.Type.String()),
// 				Current:   template.HTML(r.String()),
// 				Projected: template.HTML(p.String()),
// 			})
// 		}
// 	}
//
// 	var draw int
// 	if draw, err = strconv.Atoi(c.PostForm("draw")); err != nil {
// 		log.Errorf("strconv.Atoi error: %v", err)
// 		return
// 	}
//
// 	table.Draw = draw
// 	table.RecordsTotal = l1
// 	table.RecordsFiltered = l2
// 	return
// }
// func toCombined(c *gin.Context, us user.Users, rs, ps CurrentRatings, o int32, cnt int64) (*jCombinedRatingsIndex, error) {
// 	table := new(jCombinedRatingsIndex)
// 	l1, l2 := len(rs), len(ps)
// 	if l1 != l2 {
// 		return nil, fmt.Errorf("length mismatch between ratings and projected ratings l1: %d l2: %d", l1, l2)
// 	}
// 	table.Data = make([]*jCombined, 0)
// 	for i, r := range rs {
// 		if !r.generated {
// 			table.Data = append(table.Data, &jCombined{
// 				Rank:      i + int(o) + 1,
// 				Gravatar:  user.Gravatar(us[i], gravSize),
// 				Name:      us[i].Link(),
// 				Type:      template.HTML(r.Type.String()),
// 				Current:   template.HTML(r.String()),
// 				Projected: template.HTML(ps[i].String()),
// 			})
// 		}
// 	}
//
// 	if draw, err := strconv.Atoi(c.PostForm("draw")); err != nil {
// 		return nil, err
// 	} else {
// 		table.Draw = draw
// 	}
//
// 	table.RecordsTotal = int(cnt)
// 	table.RecordsFiltered = int(cnt)
// 	return table, nil
// }
//
// func (client *RatingClient) IncreaseFor(c *gin.Context, u *user.User, t Type, cs []*Contest) (*CurrentRating, *CurrentRating, error) {
// 	client.Log.Debugf(msgEnter)
// 	defer client.Log.Debugf(msgExit)
//
// 	k := u.Key
// 	ucs, err := client.Contest.UnappliedFor(c, k, t)
// 	if err != nil {
// 		return nil, nil, err
// 	}
//
// 	r, err := client.For(c, u, t)
// 	if err != nil {
// 		return nil, nil, err
// 	}
//
// 	cr, err := r.Projected(ucs)
// 	if err != nil {
// 		return nil, nil, err
// 	}
//
// 	nr, err := r.Projected(append(ucs, filterContestsFor(cs, k)...))
// 	return cr, nr, err
// }
//
// func filterContestsFor(cs []*Contest, pk *datastore.Key) (fcs []*Contest) {
// 	for _, c := range cs {
// 		if c.Key.Parent.Equal(pk) {
// 			fcs = append(fcs, c)
// 		}
// 	}
// 	return
// }
//
// // createTask creates a new task in your App Engine queue.
// func createTask(projectID, locationID, queueID string, uid int64, t Type) (*taskspb.Task, error) {
// 	// Create a new Cloud Tasks client instance.
// 	// See https://godoc.org/cloud.google.com/go/cloudtasks/apiv2
// 	ctx := context.Background()
// 	client, err := cloudtasks.NewClient(ctx)
// 	if err != nil {
// 		return nil, fmt.Errorf("NewClient: %v", err)
// 	}
// 	defer client.Close()
//
// 	// Build the Task queue path.
// 	queuePath := fmt.Sprintf("projects/%s/locations/%s/queues/%s", projectID, locationID, queueID)
//
// 	// Build the Task payload.
// 	// https://godoc.org/google.golang.org/genproto/googleapis/cloud/tasks/v2#CreateTaskRequest
// 	req := &taskspb.CreateTaskRequest{
// 		Parent: queuePath,
// 		Task: &taskspb.Task{
// 			// https://godoc.org/google.golang.org/genproto/googleapis/cloud/tasks/v2#AppEngineHttpRequest
// 			MessageType: &taskspb.Task_AppEngineHttpRequest{
// 				AppEngineHttpRequest: &taskspb.AppEngineHttpRequest{
// 					HttpMethod:  taskspb.HttpMethod_POST,
// 					RelativeUri: fmt.Sprintf("/ratings/userUpdate/%d/%s", uid, t),
// 				},
// 			},
// 		},
// 	}
//
// 	createdTask, err := client.CreateTask(ctx, req)
// 	if err != nil {
// 		return nil, fmt.Errorf("cloudtasks.CreateTask: %v", err)
// 	}
//
// 	return createdTask, nil
// }
