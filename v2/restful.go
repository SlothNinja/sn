package sn

// func init() {
// 	gob.Register(new(template.HTML))
// }
//
// const (
// 	contextKey      = "Context"
// 	prefixKey       = "Prefix"
// 	requestKey      = 0
// 	ginKey          = -1
// 	tmplKey         = "Templates"
// 	initialPoolSize = 32
// )
//
// func withTemplates(c *gin.Context, ts map[string]*template.Template) *gin.Context {
// 	c.Set(tmplKey, ts)
// 	return c
// }
//
// func TemplatesFrom(c *gin.Context) (ts map[string]*template.Template) {
// 	ts, _ = c.Value(tmplKey).(map[string]*template.Template)
// 	return
// }
//
// func ParseTemplates(path string, ext ...string) render.Render {
// 	r := render.New()
// 	r.TemplatesDir = path
// 	r.Exts = ext
// 	if gin.IsDebugging() {
// 		r.Debug = true
// 	}
// 	r.TemplateFuncMap = Builtins
// 	return r.Init()
// }
//
// func AddTemplates(tmpls map[string]*template.Template) gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		withTemplates(c, tmpls)
// 	}
// }
//
// func TemplateHandler(engine *gin.Engine) gin.HandlerFunc {
// 	r := render.New()
// 	r.TemplatesDir = "templates/"
// 	r.Exts = []string{".tmpl"}
// 	if gin.IsDebugging() {
// 		r.Debug = true
// 	}
// 	r.TemplateFuncMap = Builtins
// 	r = r.Init()
// 	return func(c *gin.Context) {
// 		engine.HTMLRender = r
// 		withTemplates(c, r.Templates)
// 	}
// }
//
// var Builtins = template.FuncMap{
// 	"today":       today,
// 	"parity":      parity,
// 	"odd":         odd,
// 	"even":        even,
// 	"zero":        zero,
// 	"inc":         inc,
// 	"equal":       equal,
// 	"greater":     greater,
// 	"less":        less,
// 	"ints":        ints,
// 	"round2":      round2,
// 	"LastUpdated": LastUpdated,
// 	"Time":        Time,
// 	"Date":        Date,
// 	"ToLower":     ToLower,
// 	"LoginURL":    LoginURL,
// 	"LogoutURL":   LogoutURL,
// 	"noescape":    noescape,
// 	"comment":     comment,
// 	"data":        data,
// 	"add":         add,
// 	"toSentence":  ToSentence,
// }
//
// func today() string {
// 	return time.Now().UTC().Format("January 2, 2006")
// }
//
// func Time(t time.Time) string {
// 	return t.Format("3:04PM MST")
// }
//
// func Date(t time.Time) string {
// 	return t.Format("Jan 2, 2006")
// }
//
// func parity(i int) string {
// 	if i%2 == 0 {
// 		return "even"
// 	}
// 	return "odd"
// }
//
// func odd(i int) bool {
// 	return i%2 != 0
// }
//
// func even(i int) bool {
// 	return !odd(i)
// }
//
// func zero(v interface{}) (result bool) {
// 	switch v.(type) {
// 	case int:
// 		result = v.(int) == 0
// 	case int64:
// 		result = v.(int64) == 0
// 	case time.Time:
// 		result = v.(time.Time).IsZero()
// 	}
// 	return
// }
//
// func inc(v int) int {
// 	return v + 1
// }
//
// func equal(v1, v2 interface{}) (result bool) {
// 	switch v1.(type) {
// 	case *int:
// 		switch v2.(type) {
// 		case int:
// 			result = *v1.(*int) == v2.(int)
// 		case *int:
// 			result = *v1.(*int) == *v2.(*int)
// 		}
// 	case int:
// 		return v1.(int) == v2.(int)
// 	case int64:
// 		return v1.(int64) == v2.(int64)
// 	case string:
// 		return v1.(string) == v2.(string)
// 	}
// 	return false
// }
//
// func less(v1, v2 interface{}) bool {
// 	switch v1.(type) {
// 	case int:
// 		return v1.(int) < v2.(int)
// 	case int64:
// 		return v1.(int64) < v2.(int64)
// 	}
// 	return false
// }
//
// func greater(v1, v2 interface{}) bool {
// 	switch v1.(type) {
// 	case int:
// 		return v1.(int) > v2.(int)
// 	case int64:
// 		return v1.(int64) > v2.(int64)
// 	}
// 	return false
// }
//
// func ints(start, stop int) []int {
// 	var s []int
// 	for i := start; i <= stop; i++ {
// 		s = append(s, i)
// 	}
// 	return s
// }
//
// func round2(f float64) string {
// 	i := math.Floor(f + 0.005)
// 	d := int(math.Floor(((f + 0.005) - i) * 100))
// 	if d < 10 {
// 		return fmt.Sprintf("%d.0%d", int(i), d)
// 	} else {
// 		return fmt.Sprintf("%d.%d", int(i), d)
// 	}
// }
//
// func ToLower(s string) string {
// 	return strings.ToLower(s)
// }
//
// func LogoutURL(c *gin.Context, redirect, label string) (template.HTML, error) {
// 	return template.HTML(fmt.Sprintf("<a href='/user/logout'>%s</a>", label)), nil
// 	// url, err := user.LogoutURL(c, redirect)
// 	// if err != nil {
// 	// 	return "", err
// 	// }
// 	// return template.HTML(fmt.Sprintf(`<a href=%q>%s</a>`, url, label)), nil
// }
//
// func LoginURL(c *gin.Context, redirect, label string) (tmpl template.HTML, err error) {
// 	return template.HTML(fmt.Sprintf("<a href='/user/login'>%s</a>", label)), nil
// 	// url, err := user.LoginURL(c, redirect)
// 	// if err != nil {
// 	// 	return
// 	// }
// 	// tmpl = template.HTML(fmt.Sprintf(`<a href=%q>%s</a>`, url, label))
// 	// return
// }
//
// func noescape(s string) template.HTML {
// 	return template.HTML(s)
// }
//
// func comment(s string) template.HTML {
// 	return template.HTML("<!-- " + s + " -->")
// }
//
// func data(args ...interface{}) map[string]interface{} {
// 	d := make(map[string]interface{}, len(args)/2)
// 	for i := 0; i < len(args); i += 2 {
// 		d[args[i].(string)] = args[i+1]
// 	}
// 	return d
// }
//
// func add(args ...int) (sum int) {
// 	for i := range args {
// 		sum += args[i]
// 	}
// 	return
// }
//
// const (
// 	nKey = "Notices"
// 	eKey = "Errors"
// )
//
// type Notices []template.HTML
//
// func NoticesFrom(c *gin.Context) (ns Notices) {
// 	ns, _ = c.Value(nKey).(Notices)
// 	return
// }
//
// func withNotices(c *gin.Context, ns Notices) *gin.Context {
// 	c.Set(nKey, ns)
// 	return c
// }
//
// func AddNoticef(c *gin.Context, format string, args ...interface{}) {
// 	withNotices(c, append(NoticesFrom(c), HTML(format, args...)))
// }
//
// type Errors []template.HTML
//
// func ErrorsFrom(c *gin.Context) (es Errors) {
// 	es, _ = c.Value(eKey).(Errors)
// 	return
// }
//
// func withErrors(c *gin.Context, es Errors) *gin.Context {
// 	c.Set(eKey, es)
// 	return c
// }
//
// func AddErrorf(c *gin.Context, format string, args ...interface{}) {
// 	withErrors(c, append(ErrorsFrom(c), HTML(format, args...)))
// }
//
// func HTML(format string, args ...interface{}) template.HTML {
// 	return template.HTML(fmt.Sprintf(format, args...))
// }

func ToSentence(strings []string) (sentence string) {
	switch length := len(strings); length {
	case 0:
	case 1:
		sentence = strings[0]
	case 2:
		sentence = strings[0] + " and " + strings[1]
	default:
		for i, s := range strings {
			switch i {
			case 0:
				sentence += s
			case length - 1:
				sentence += ", and " + s
			default:
				sentence += ", " + s
			}
		}
	}
	return sentence
}

// func Camelize(ss ...string) string {
// 	return strings.Replace(Titlize(ss...), " ", "", -1)
// }
//
// func Titlize(ss ...string) string {
// 	return strings.Title(strings.TrimSpace(combine(ss...)))
// }
//
// func combine(ss ...string) string {
// 	result := ""
// 	for _, s := range ss {
// 		result += " " + strings.TrimSpace(s)
// 	}
// 	return result
// }
//
// func IDize(ss ...string) string {
// 	return strings.Replace(strings.ToLower(combine(ss...)), " ", "-", -1)
// }
//
// func IDString(s string) string {
// 	return strings.Replace(strings.ToLower(s), " ", "-", -1)
// }
//
// func JSONString(ss ...string) string {
// 	switch s := Camelize(ss...); len(s) {
// 	case 0:
// 		return ""
// 	case 1:
// 		return strings.ToLower(s[0:1])
// 	default:
// 		return strings.ToLower(s[0:1]) + s[1:]
// 	}
// }
//
// func Pluralize(label string, value int) string {
// 	if value != 1 {
// 		return inflect.Pluralize(label)
// 	}
// 	return label
// }
