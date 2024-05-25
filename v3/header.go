package sn

import (
	"fmt"
	"time"

	"github.com/elliotchance/pie/v2"
)

// Header provides fields common to all games.
type Header struct {
	ID                        string `firestore:"-"`
	Type                      Type
	Title                     string
	Turn                      int
	Phase                     Phase
	Round                     int
	NumPlayers                int
	CreatorID                 UID
	CreatorName               string
	CreatorEmail              string
	CreatorEmailNotifications bool
	CreatorEmailHash          string
	CreatorGravType           string
	UserIDS                   []UID
	UserNames                 []string
	UserEmails                []string
	UserEmailHashes           []string
	UserEmailNotifications    []bool
	UserGravTypes             []string
	OrderIDS                  []PID
	CPIDS                     []PID
	WinnerIDS                 []UID
	Status                    Status
	Undo                      Stack
	OptString                 string
	StartedAt                 time.Time
	EndedAt                   time.Time
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
	Private                   bool
}

func (h *Header) getHeader() *Header {
	return h
}

func (h Header) Rev() int {
	return h.Undo.Current
}

func (h Header) Committed() int {
	return h.Undo.Committed
}

// func (h *Header) Load(ps []datastore.Property) error {
// 	return datastore.LoadStruct(h, ps)
// }
//
// func (h *Header) Save() ([]datastore.Property, error) {
// 	t := time.Now()
// 	if h.CreatedAt.IsZero() {
// 		h.CreatedAt = t
// 	}
// 	h.UpdatedAt = t
// 	return datastore.SaveStruct(h)
// }
//
// func (h *Header) LoadKey(k *datastore.Key) error {
// 	h.Key = k
// 	return nil
// }

// func (h *Header) CTX() *gin.Context {
// 	return h.c
// }
//
// func (h *Header) SetCTX(c *gin.Context) {
// 	h.c = c
// }

// type headerer interface {
// 	GetHeader() *Header
// 	GetAcceptDialog() bool
// 	AcceptedPlayers() int
// 	PlayererByPID(PID) Playerer
// 	PlayererByUserID(int64) Playerer
// 	PlayererByIndex(int) Playerer
// 	Winnerers() Playerers
// 	User(UIndex) User
// 	CurrentPlayerers() []Playerer
// 	NextPlayerer(...Playerer) Playerer
// 	DefaultColorMap() []Color
// 	UserLinks() template.HTML
// 	Private() bool
// 	CanAdd(User) bool
// 	CanDropout(User) bool
// 	Stub() string
// 	CTX() *gin.Context
// 	Accept(*gin.Context, User) (bool, error)
// 	Drop(User) error
// 	IsCurrentPlayer(User) bool
// }
//
// func (h Header) ID() int64 {
// 	if h.Key == nil {
// 		return 0
// 	}
// 	return h.Key.ID
// }

// func (h *Header) GetHeader() *Header {
// 	return h
// }

// type UserIndices []int
//
// func (uis *UserIndices) Append(indices ...int)             { *uis = uis.AppendS(indices...) }
// func (uis UserIndices) AppendS(indices ...int) UserIndices { return append(uis, indices...) }
//
// func (uis UserIndices) Include(index int) bool {
// 	for _, i := range uis {
// 		if i == index {
// 			return true
// 		}
// 	}
// 	return false
// }
//
// func HasUIndex(uis []UIndex, i UIndex) bool {
// 	for _, ui := range uis {
// 		if ui == i {
// 			return true
// 		}
// 	}
// 	return false
// }
//
// func (uis UserIndices) RemoveS(indices ...int) UserIndices {
// 	for _, index := range indices {
// 		uis = uis.remove(index)
// 	}
// 	return uis
// }
//
// func (uis UserIndices) remove(index int) UserIndices {
// 	for i, indx := range uis {
// 		if indx == index {
// 			return uis.removeAt(i)
// 		}
// 	}
// 	return uis
// }
//
// func (uis UserIndices) removeAt(i int) UserIndices { return append(uis[:i], uis[i+1:]...) }

// func NewHeader(c *gin.Context, g Gamer, id int64) *Header {
// 	return &Header{
// 		c:     c,
// 		gamer: g,
// 		Key:   datastore.IDKey("Game", id, GamesRoot(c)),
// 	}
// }

type Strings []string

type ColorMaps map[Type][]Color

var defaultColorMaps = ColorMaps{
	Confucius: []Color{Yellow, Purple, Green, White, Black},
	Tammany:   []Color{Red, Yellow, Purple, Black, Brown},
	ATF:       []Color{Red, Green, Purple},
	GOT:       []Color{Yellow, Purple, Green, Black},
	Indonesia: []Color{White, Black, Green, Purple, Orange},
}

// func (h *Header) DefaultColorMap() []Color {
// 	return defaultColorMaps[h.Type]
// }
//
// func (h *Header) ColorMapFor(u User) ColorMap {
// 	cm := h.DefaultColorMap()
// 	if u != nil {
// 		if p := h.PlayererByUserID(u.ID()); p != nil {
// 			cm = p.ColorMap()
// 		}
// 	}
// 	cMap := make(ColorMap, len(h.UserIDS))
// 	for i, uid := range h.UserIDS {
// 		cMap[int(uid)] = cm[i]
// 	}
// 	return cMap
// }
//
// func (ss Strings) Include(s string) bool {
// 	for _, value := range ss {
// 		if s == value {
// 			return true
// 		}
// 	}
// 	return false
// }

// func actionPath(r *http.Request) string {
// 	s := strings.Split(r.URL.String(), "/")
// 	return s[len(s)-1]
// }

// func (h *Header) FromParams(c *gin.Context, cu User, t Type) error {
// 	Debugf(msgEnter)
// 	defer Debugf(msgExit)
//
// 	h.Title = cu.Name + "'s Game"
// 	h.Status = Recruiting
// 	h.Type = t
// 	return nil
// }

// func (h *Header) FromForm(c *gin.Context, cu User, t Type) error {
// 	Debugf(msgEnter)
// 	defer Debugf(msgExit)
//
// 	obj := struct {
// 		Title      string `form:"title"`
// 		NumPlayers int    `form:"num-players" binding"min=0,max=5"`
// 		Password   string `form:"password"`
// 	}{}
//
// 	err := c.ShouldBind(&obj)
// 	if err != nil {
// 		return err
// 	}
//
// 	h.Title = cu.Name + "'s Game"
// 	if obj.Title != "" {
// 		h.Title = obj.Title
// 	}
//
// 	h.NumPlayers = 4
// 	if obj.NumPlayers >= 1 && obj.NumPlayers <= 5 {
// 		h.NumPlayers = obj.NumPlayers
// 	}
//
// 	h.Password = obj.Password
// 	h.AddCreator(cu)
// 	h.AddUser(cu)
// 	h.Status = Recruiting
// 	h.Type = t
// 	return nil
// }

// func (h *Header) User(index UIndex) User {
// 	i := int(index)
// 	l := len(h.UserIDS)
// 	if l > 0 {
// 		i = i % l
// 	}
// 	return h.Users[i]
// }

// func (client *Client) AfterLoad(c *gin.Context, h *Header) error {
// 	h.AfterLoad()
// 	return nil
// }
//
// func (h *Header) AfterLoad() {
// 	h.Users = make(Users, len(h.UserIDS))
// 	for i, id := range h.UserIDS {
// 		h.Users[i] = NewUser(id)
// 		if i >= 0 && i < len(h.UserNames) {
// 			h.Users[i].Name = h.UserNames[i]
// 		}
// 		if i >= 0 && i < len(h.UserEmails) {
// 			h.Users[i].Email = h.UserEmails[i]
// 		}
// 	}
//
// 	h.Creator = NewUser(h.CreatorID)
// 	h.Creator.Name = h.CreatorName
// 	h.Creator.Email = h.CreatorEmail
// }

// func include(ints []int64, i int64) bool {
// 	for _, v := range ints {
// 		if v == i {
// 			return true
// 		}
// 	}
// 	return false
// }

// func remove(ints []int64, i int64) []int64 {
// 	for index, j := range ints {
// 		if j == i {
// 			return append(ints[:index], ints[index+1:]...)
// 		}
// 	}
// 	return ints
// }

func (h *Header) CanAdd(u User) bool {
	return len(h.UserIDS) < h.NumPlayers && !pie.Contains(h.UserIDS, u.ID)
}

func (h *Header) CanDropout(u *User) bool {
	return h.Status == Recruiting && pie.Contains(h.UserIDS, u.ID)
}

func (h *Header) Stub() string {
	return string(h.Type)
}

// func (h *Header) Private() bool {
// 	return h.Password != ""
// }

func (h *Header) HasUser(u *User) bool {
	return pie.Contains(h.UserIDS, u.ID)
}

func (h *Header) RemoveUser(u2 *User) {
	i := h.IndexFor(u2.ID)
	if i == UIndexNotFound {
		return
	}

	if i >= 0 && i < UIndex(len(h.UserIDS)) {
		h.UserIDS = append(h.UserIDS[:i], h.UserIDS[i+1:]...)
	}
	// if i >= 0 && i < UIndex(len(h.UserKeys)) {
	// 	h.UserKeys = append(h.UserKeys[:i], h.UserKeys[i+1:]...)
	// }
	if i >= 0 && i < UIndex(len(h.UserNames)) {
		h.UserNames = append(h.UserNames[:i], h.UserNames[i+1:]...)
	}
	if i >= 0 && i < UIndex(len(h.UserEmails)) {
		h.UserEmails = append(h.UserEmails[:i], h.UserEmails[i+1:]...)
	}
	if i >= 0 && i < UIndex(len(h.UserEmailHashes)) {
		h.UserEmailHashes = append(h.UserEmailHashes[:i], h.UserEmailHashes[i+1:]...)
	}
	if i >= 0 && i < UIndex(len(h.UserEmailNotifications)) {
		h.UserEmailNotifications = append(h.UserEmailNotifications[:i], h.UserEmailNotifications[i+1:]...)
	}
	if i >= 0 && i < UIndex(len(h.UserGravTypes)) {
		h.UserGravTypes = append(h.UserGravTypes[:i], h.UserGravTypes[i+1:]...)
	}
}
func (h *Header) AddUser(u *User) {
	h.UserIDS = append(h.UserIDS, u.ID)
	// h.UserKeys = append(h.UserKeys, u.Key)
	h.UserNames = append(h.UserNames, u.Name)
	h.UserEmails = append(h.UserEmails, u.Email)
	h.UserEmailHashes = append(h.UserEmailHashes, u.EmailHash)
	h.UserEmailNotifications = append(h.UserEmailNotifications, u.EmailNotifications)
	h.UserGravTypes = append(h.UserGravTypes, u.GravType)
}

func (h *Header) AddCreator(u *User) {
	// h.Creator = u
	h.CreatorID = u.ID
	// h.CreatorKey = u.Key
	h.CreatorName = u.Name
	h.CreatorEmail = u.Email
	h.CreatorEmailHash = u.EmailHash
	h.CreatorEmailNotifications = u.EmailNotifications
	h.CreatorGravType = u.GravType
}

func (h *Header) AddUsers(us ...*User) {
	for _, u := range us {
		h.AddUser(u)
	}
}

// func (h *Header) CurrentPlayerer() Playerer {
// 	if len(h.CurrentPlayerers()) == 1 {
// 		return h.CurrentPlayerers()[0]
// 	}
// 	return nil
// }
//
// // CurrentPlayererFrom provides the first current player from players ps.
// func (h *Header) CurrentPlayerFrom(ps Playerers) (cp Playerer) {
// 	if cps := h.CurrentPlayersFrom(ps); len(cps) > 0 {
// 		cp = cps[0]
// 	}
// 	return
// }
//
// func (h *Header) CurrentUserPlayerer(cu User) Playerer {
// 	switch cps := h.CurrentUserPlayerers(cu); len(cps) {
// 	case 0:
// 		return nil
// 	case 1:
// 		return cps[0]
// 	default:
// 		Warningf("CurrentUserPlayerer found %d current user players.  Returned only the first.")
// 		return cps[0]
// 	}
// }
//
// func isAdmin(u User) bool {
// 	if u == nil {
// 		return false
// 	}
// 	return u.Admin
// }

// func (h *Header) CurrentUserPlayerers(cu User) Playerers {
// 	if cu == nil {
// 		return nil
// 	}
//
// 	for _, cp := range h.CurrentPlayerers() {
// 		if isAdmin(cu) || cp.User().Equal(cu) {
// 			return Playerers{cp}
// 		}
// 	}
// 	return nil
// }
//
// // CurrentPlayererFor returns the current player from players ps associated with the user u.
// // If no player is associated with the user, but user is admin, then returns default current player.
// func (h *Header) CurrentPlayerFor(ps Playerers, u User) (cp Playerer) {
// 	if u == nil {
// 		return
// 	}
//
// 	for _, p := range h.CurrentPlayersFrom(ps) {
// 		if p.User().ID() == u.ID() {
// 			cp = p
// 			return
// 		}
// 	}
//
// 	if isAdmin(u) {
// 		cp = h.CurrentPlayerFrom(ps)
// 	}
// 	return
// }
//
// func (h *Header) CurrentPlayerers() []Playerer {
// 	if h.Status == Completed {
// 		return nil
// 	}
//
// 	l := len(h.CPUserIndices)
// 	if l == 0 {
// 		return nil
// 	}
//
// 	ps := make([]Playerer, l)
// 	for i, index := range h.CPUserIndices {
// 		ps[i] = h.PlayerByUserIndex(index)
// 	}
// 	return ps
// }
//
// // CurrentPlayerers returns the current players in players.
// func (h *Header) CurrentPlayersFrom(players Playerers) (ps Playerers) {
// 	if h.Status != Completed {
// 		for _, index := range h.CPUserIndices {
// 			ps = append(ps, PlayerByUserIndex(players, index))
// 		}
// 	}
// 	return
// }
//
// func (h *Header) NextPlayerer(p Playerer) Playerer {
// 	Debugf(msgEnter)
// 	defer Debugf(msgExit)
//
// 	if p == nil {
// 		return nil
// 	}
//
// 	for i, pid := range h.OrderIDS {
// 		if pid == p.ID() {
// 			return h.PlayererByIndex(i + 1)
// 		}
// 	}
//
// 	return nil
// }
//
// func (h *Header) PreviousPlayerer(p Playerer) Playerer {
// 	Debugf(msgEnter)
// 	defer Debugf(msgExit)
//
// 	if p == nil {
// 		return nil
// 	}
//
// 	for i, pid := range h.OrderIDS {
// 		if pid == p.ID() {
// 			return h.PlayererByIndex(i - 1)
// 		}
// 	}
//
// 	return nil
// }
//
// func (h *Header) Winnerers() Playerers {
// 	if len(h.WinnerIDS) == 0 || h.WinnerIDS[0] == -1 {
// 		return nil
// 	}
//
// 	var playerers Playerers
// 	for _, index := range h.WinnerIDS {
// 		playerers = append(playerers, h.PlayerByUserIndex(index))
// 	}
// 	return playerers
// }

//	func (h *Header) SetCurrentPlayerers(players ...Playerer) {
//		switch length := len(players); {
//		case length > 0:
//			h.CPUserIndices = make([]UIndex, length)
//			for i, p := range players {
//				h.CPUserIndices[i] = p.UIndex()
//			}
//		default:
//			h.CPUserIndices = nil
//		}
//	}
//
//	func (h *Header) RemoveCurrentPlayers(ps ...Playerer) {
//		if len(ps) > 0 {
//			players := h.CurrentPlayerers()
//			for _, rp := range ps {
//				for i, p := range players {
//					if p.ID() == rp.ID() {
//						players = append(players[:i], players[i+1:]...)
//						break
//					}
//				}
//			}
//			h.SetCurrentPlayerers(players...)
//		}
//	}
// func (h *Header) isCP(uIndex UIndex) bool {
// 	if len(h.CPUserIndices) == 0 || h.CPUserIndices[0] == -1 || uIndex == -1 {
// 		return false
// 	}
//
// 	for _, cpi := range h.CPUserIndices {
// 		if cpi == uIndex {
// 			return true
// 		}
// 	}
// 	return false
// }

// // IsCurrentPlayer returns true if the specified user is the current player.
// func (h *Header) IsCurrentPlayer(u User) bool {
// 	return u != nil && h.isCP(h.IndexFor(u.ID()))
// }

// // IsCurrentPlayer returns ture if the user is the current player or an admin.
//
//	func (h *Header) IsCurrentPlayerOrAdmin(u User) bool {
//		return u != nil && (isAdmin(u) || h.IsCurrentPlayer(u))
//	}
// func (h *Header) isCurrentPlayerOrAdmin(c *gin.Context, u User) bool {
// 	return u != nil && (isAdmin(u) || h.IsCurrentPlayer(u))
// }
//
// // CurrentUserIsCurrentPlayerOrAdmin returns true if current user is the current player or is an administrator.
// // Deprecated in favor of CUserIsCPlayerOrAdmin
// func (h *Header) CurrentUserIsCurrentPlayerOrAdmin(cu User) bool {
// 	c := h.CTX()
// 	Warningf("CurrentUserIsCurrentPlayerOrAdmin is deprecated in favor of CUserIsCPlayerOrAdmin.")
// 	return h.isCurrentPlayerOrAdmin(c, cu)
// }
//
// func (h *Header) PlayerIsUser(p Playerer, u User) bool {
// 	return p != nil && u != nil && h.UserIDFor(p.ID()) == u.ID()
// }
//
// func (h *Header) IsW(uIndex UIndex) bool {
// 	return HasUIndex(h.WinnerIDS, uIndex)
// }

// func (h *Header) IsWinner(u User) bool {
// 	for _, p := range h.PlayerersByUser(u) {
// 		if HasUIndex(h.WinnerIDS, p.UIndex()) {
// 			return true
// 		}
// 	}
// 	return false
// }

// func (h *Header) UserLinks() template.HTML {
// 	links := make([]string, len(h.UserIDS))
// 	for i, uid := range h.UserIDS {
// 		links[i] = string(h.UserLinkFor(uid))
// 	}
// 	return template.HTML(ToSentence(links))
// }
//
// func (h *Header) UserLinkFor(uid UID) template.HTML {
// 	return LinkFor(uid, h.NameByUID(uid))
// }
//
// func (h *Header) PlayerLinkByPID(cu User, pid PID) template.HTML {
// 	i := pid.ToIndex()
// 	uid := h.UserIDS[pid.ToIndex()]
//
// 	cp := h.isCP(i)
//
// 	var me bool
// 	if cu != nil {
// 		me = cu.ID() == uid
// 	}
//
// 	w := h.IsW(i)
// 	n := h.NameFor(pid)
//
// 	path := PathFor(uid)
// 	result := fmt.Sprintf(`<a href=%q >%s</a>`, path, n)
// 	switch h.Status {
// 	case Running:
// 		switch {
// 		case cp && me:
// 			result = fmt.Sprintf(`<a href=%q class="current-player me">%s</a>`, path, n)
// 		case cp:
// 			result = fmt.Sprintf(`<a href=%q class="current-player">%s</a>`, path, n)
// 		}
// 	case Completed:
// 		switch {
// 		case w && me:
// 			result = fmt.Sprintf(`<a href=%q class="winner me">%s</a>`, path, n)
// 		case w:
// 			result = fmt.Sprintf(`<a href=%q class="winner">%s</a>`, path, n)
// 		}
// 	}
// 	return template.HTML(result)
// }

// func (h *Header) PlayerLinks(cu User) template.HTML {
// 	if h.Status == Recruiting {
// 		return h.UserLinks()
// 	}
//
// 	links := make([]string, len(h.OrderIDS))
// 	for i, pid := range h.OrderIDS {
// 		links[i] = string(h.PlayerLinkByPID(cu, pid))
// 	}
// 	return template.HTML(ToSentence(links))
// }

// func (h *Header) CurrentPlayerLinks(cu User) template.HTML {
// 	cps := h.CPUserIndices
// 	if len(cps) == 0 || h.Status != Running {
// 		return "None"
// 	}
//
// 	links := make([]string, len(cps))
// 	for j, uIndex := range cps {
// 		links[j] = string(h.PlayerLinkByPID(cu, uIndex.ToPID()))
// 	}
// 	return template.HTML(ToSentence(links))
// }

//	func (h *Header) NoCurrentPlayer() bool {
//		return len(h.CPUserIndices) == 0
//	}
// func (h *Header) CurrentPlayerLabel() string {
// 	if length := len(h.CPUserIndices); length == 1 {
// 		return "Current Player"
// 	}
// 	return "Current Players"
// }
//
// func (h *Header) AcceptedPlayers() int {
// 	return len(h.UserIDS)
// }
//
// // PlayererByPID returns the player having the PID.
// func (h *Header) PlayererByPID(pid PID) (p Playerer) {
// 	return PlayererByPID(h.gamer.(GetPlayerers).GetPlayerers(), pid)
// }
//
// // PlayererByPID returns the player from ps having the PID.
// func PlayererByPID(ps Playerers, pid PID) Playerer {
// 	for _, p2 := range ps {
// 		if p2.ID() == pid {
// 			return p2
// 		}
// 	}
// 	return nil
// }

// func (h *Header) PlayererByColor(c Color) Playerer {
// 	for _, p := range h.gamer.(GetPlayerers).GetPlayerers() {
// 		if p.Color() == c {
// 			return p
// 		}
// 	}
// 	return nil
// }
//
// // PlayerBySID provides the player having the id represented by the string.
// func (h *Header) PlayerBySID(sid string) Playerer {
// 	i, err := strconv.Atoi(sid)
// 	if err == nil {
// 		return h.PlayererByPID(PID(i))
// 	}
// 	return nil
// }
//
// // PlayerBySID provides the player in ps having the id represented by the string.
// func PlayerBySID(ps Playerers, sid string) Playerer {
// 	i, err := strconv.Atoi(sid)
// 	if err == nil {
// 		return PlayererByPID(ps, PID(i))
// 	}
// 	return nil
// }
//
// // PlayererByUserID returns the player associated with the user id
// func (h *Header) PlayererByUserID(id int64) Playerer {
// 	return PlayererByUserID(h.gamer.(GetPlayerers).GetPlayerers(), id)
// }
//
// // PlayererByUserID returns the player from ps associated with the user id
// func PlayererByUserID(ps Playerers, id int64) (p Playerer) {
// 	for _, p2 := range ps {
// 		if p2.User().ID() == id {
// 			p = p2
// 			return
// 		}
// 	}
// 	return
// }
//
// func (h *Header) PlayerersByUser(user User) Playerers {
// 	var ps Playerers
// 	for _, p := range h.gamer.(GetPlayerers).GetPlayerers() {
// 		if p.User().Equal(user) {
// 			ps = append(ps, p)
// 		}
// 	}
// 	return ps
// }
//
// func (h *Header) PlayerByUserIndex(i UIndex) Playerer {
// 	for _, p := range h.gamer.(GetPlayerers).GetPlayerers() {
// 		if p.UIndex() == i {
// 			return p
// 		}
// 	}
// 	return nil
// }

// // PlayerByUserIndex returns the player from players ps having the provided user index.
// func PlayerByUserIndex(ps Playerers, index UIndex) Playerer {
// 	for _, p2 := range ps {
// 		if p2.ID().ToIndex() == index {
// 			return p2
// 		}
// 	}
// 	return nil
// }
//
// // PlayererByIndex returns the player at the index i in the ring of players ps
// // Convenience method that automatically wraps-around based on number of players.
// // TODO: Deprecated
// func (h *Header) PlayererByIndex(i int) Playerer {
// 	return PlayererByIndex(h.gamer.(GetPlayerers).GetPlayerers(), i)
// }
//
// // PlayererByIndex returns the player at the index i in the ring of players ps
// // Wraps-around based on number of players.
// func PlayererByIndex(ps Playerers, i int) Playerer {
// 	l := len(ps)
// 	r := i % l
// 	if r < 0 {
// 		return ps[l+r]
// 	}
// 	return ps[r]
// }

type Phase string

// func (p Phase) Int() int {
// 	return int(p)
// }
//
// type PhaseNameMap map[Phase]string
// type PhaseNameMaps map[Type]PhaseNameMap
//
// func registerPhaseNames(t Type, names PhaseNameMap) {
// 	if phaseNameMaps == nil {
// 		phaseNameMaps = make(PhaseNameMaps, len(types()))
// 	}
// 	phaseNameMaps[t] = names
// }
//
// func registerSubPhaseNames(t Type, names SubPhaseNameMap) {
// 	if subPhaseNameMaps == nil {
// 		subPhaseNameMaps = make(SubPhaseNameMaps, len(types()))
// 	}
// 	subPhaseNameMaps[t] = names
// }
//
// type factoryMap map[Type]Factory
//
// var factories factoryMap

// type Factory func(*gin.Context) Gamer
//
// func Register(t Type, f Factory, p PhaseNameMap, sp SubPhaseNameMap) {
// 	if factories == nil {
// 		factories = make(factoryMap, len(types()))
// 	}
// 	factories[t] = f
// 	registerPhaseNames(t, p)
// 	registerSubPhaseNames(t, sp)
// }
//
// func (h *Header) PhaseName() string {
// 	if phaseNameMaps == nil {
// 		return ""
// 	}
// 	if names, ok := phaseNameMaps[h.Type]; ok {
// 		return names[h.Phase]
// 	}
// 	return ""
// }
//
// type SubPhase int
// type SubPhaseNameMap map[SubPhase]string
// type SubPhaseNameMaps map[Type]SubPhaseNameMap
//
// func (h *Header) SubPhaseName() string {
// 	if subPhaseNameMaps == nil {
// 		return ""
// 	}
// 	if names, ok := subPhaseNameMaps[h.Type]; ok {
// 		return names[h.SubPhase]
// 	}
// 	return ""
// }
//
// func (s SubPhase) Int() int {
// 	return int(s)
// }
//
// var phaseNameMaps PhaseNameMaps
// var subPhaseNameMaps SubPhaseNameMaps
//
// func (h *Header) ValidateHeader() error {
// 	if len(h.UserIDS) > h.NumPlayers {
// 		return fmt.Errorf("UserIDS can't be greater than the number of players")
// 	}
// 	return nil
// }
//
// func (h *Header) notificationFor(c *gin.Context, p Playerer) (mailjet.InfoMessagesV31, error) {
// 	Debugf(msgEnter)
// 	defer Debugf(msgExit)
//
// 	gInfo := inf{GameID: h.ID(), Type: h.Type, Title: h.Title}
// 	buf := new(bytes.Buffer)
//
// 	msg := mailjet.InfoMessagesV31{
// 		From: &mailjet.RecipientV31{
// 			Email: "webmaster@slothninja.com",
// 			Name:  "Webmaster",
// 		},
// 	}
//
// 	tmpl := TemplatesFrom(c)["shared/turn_notification"]
//
// 	msg.Subject = fmt.Sprintf("SlothNinja Games: It's your turn in %s (%d)", gInfo.Title, gInfo.GameID)
//
// 	err := tmpl.Execute(buf, gin.H{"Game": gInfo})
// 	if err != nil {
// 		return msg, err
// 	}
//
// 	msg.HTMLPart = buf.String()
// 	msg.To = &mailjet.RecipientsV31{
// 		mailjet.RecipientV31{
// 			Email: h.EmailFor(p.ID()),
// 			Name:  h.NameFor(p.ID()),
// 		},
// 	}
// 	return msg, nil
// }

// func (h *Header) SendTurnNotificationsTo(c *gin.Context, ps ...Playerer) error {
// 	Debugf(msgEnter)
// 	defer Debugf(msgExit)
//
// 	if h.Type == Indonesia {
// 		return nil
// 	}
//
// 	l := len(ps)
// 	if l == 0 {
// 		return nil
// 	}
//
// 	if l == 1 {
// 		msg, err := h.notificationFor(c, ps[0])
// 		if err != nil {
// 			return err
// 		}
//
// 		_, err = SendMessages(c, msg)
// 		return err
// 	}
//
// 	isNil := true
// 	me := make(datastore.MultiError, l)
// 	for i, p := range ps {
// 		msg, err := h.notificationFor(c, p)
// 		me[i] = err
// 		if err != nil {
// 			isNil = false
// 			continue
// 		}
// 		_, err = SendMessages(c, msg)
// 		me[i] = err
// 		if err != nil {
// 			isNil = false
// 		}
// 	}
//
// 	if isNil {
// 		return nil
// 	}
// 	return me
// }

// type withID struct {
// 	*Header
// }
//
// func (wid *withID) MarshalJSON() ([]byte, error) {
// 	jHeader := struct {
// 		ID          int64  `json:"id"`
// 		LastUpdated string `json:"lastUpdated"`
// 		*Header
// 	}{wid.Key.ID, lastUpdated(wid.UpdatedAt), wid.Header}
// 	return json.Marshal(jHeader)
// }

const (
	minute = time.Minute
	hour   = time.Hour
	day    = 24 * time.Hour
	month  = 30 * day
	year   = 365 * day
)

func lastUpdated(t time.Time) string {
	duration := time.Since(t)
	switch {
	case duration < time.Minute:
		return fmt.Sprintf("%d sec", int(duration.Seconds()))
	case duration < time.Hour:
		return fmt.Sprintf("%d min", int(duration.Minutes()))
	case duration < day:
		return fmt.Sprintf("%d hour", int(duration.Hours()))
	case duration < month:
		return fmt.Sprintf("%d day", int(duration/day))
	case duration < year:
		return fmt.Sprintf("%d month", int(duration/month))
	}
	return fmt.Sprintf("%d year", int(duration/year))
}

func LastUpdated(t time.Time) string {
	return lastUpdated(t)
}

// // GHeader stores game headers with associate game data.
// type GHeader struct {
// 	Key *datastore.Key `datastore:"__key__"`
// 	Header
// }
//
// func (gh GHeader) id() int64 {
// 	if gh.Key == nil {
// 		return 0
// 	}
// 	return gh.Key.ID
// }
//
// // MarshalJSON implements json.Marshaler interface
// func (gh GHeader) MarshalJSON() ([]byte, error) {
// 	h, err := json.Marshal(gh.Header)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	var data map[string]interface{}
// 	err = json.Unmarshal(h, &data)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	data["key"] = gh.Key
// 	data["id"] = gh.id()
// 	data["lastUpdated"] = LastUpdated(gh.UpdatedAt)
// 	data["public"] = len(gh.Password) == 0
//
// 	return json.Marshal(data)
// }
