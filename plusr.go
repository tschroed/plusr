package plusr

// TODO(tschroed):
// - Make it so that client id and secret are looked up from storage
// - Factor out oauth stuff to be less mainline-y
// - Implement token caching
// - Tests, sucker!

import (
	"errors"
	"html/template"
	"net/http"
	"time"

	"appengine"
	"appengine/datastore"
	"appengine/urlfetch"
	"appengine/user"

	"code.google.com/p/goauth2/oauth"
)

type oauth2ClientConfig struct {
	clientId     string
	clientSecret string
	redirectURL  string
	scope        string
	requestURL   string
	authURL      string
	tokenURL     string
}

// TODO(tschroed): Ideally what we'd do here is provide a handler
// which would allow admin users to pass new id, secret, and
// redirect URL and then save that value in data store.
//
// The rest of the time it would be looked up.
func newOauth2ClientConfig() *oauth2ClientConfig {
	return &oauth2ClientConfig{
		clientId:     "",
		clientSecret: "",
		redirectURL:  "http://lungworm.zweknu.org:8080/oauth2callback",
		scope:        "https://picasaweb.google.com/data/",
		requestURL:   "https://www.googleapis.com/oauth2/v1/userinfo",
		authURL:      "https://accounts.google.com/o/oauth2/auth",
		tokenURL:     "https://accounts.google.com/o/oauth2/token",
	}
}

type Greeting struct {
	Author  string
	Content string
	Date    time.Time
}

type UserConfig struct {
	Context appengine.Context
}

func (uc UserConfig) Token() (*oauth.Token, error) {
	return nil, errors.New("Not implemented")
}

func (uc UserConfig) PutToken(*oauth.Token) error {
	return errors.New("Not implemented")
}

func init() {
	http.HandleFunc("/", root)
	http.HandleFunc("/oauth2callback", fetchToken)
	http.HandleFunc("/logout", logout)
	http.HandleFunc("/sign", sign)
}

// guestbookKey returns the key used for all guestbook entries.
func guestbookKey(c appengine.Context) *datastore.Key {
	// The string "default_guestbook" here could be varied to have multiple guestbooks.
	return datastore.NewKey(c, "Guestbook", "default_guestbook", 0, nil)
}

func userKey(c appengine.Context) *datastore.Key {
	return datastore.NewKey(c, "Plusr", user.Current(c).String(), 0, nil)
}

func fetchToken(w http.ResponseWriter, r *http.Request) {
	code := r.FormValue("code")
	err := r.FormValue("error")
	c := appengine.NewContext(r)
	uc := &UserConfig{Context: c}
	oa2c := newOauth2ClientConfig()
	config := &oauth.Config{
		ClientId:     oa2c.clientId,
		ClientSecret: oa2c.clientSecret,
		RedirectURL:  oa2c.redirectURL,
		Scope:        oa2c.scope,
		AuthURL:      oa2c.authURL,
		TokenURL:     oa2c.tokenURL,
		TokenCache:   uc,
	}
	uc.Context.Infof("%s", config)
	if err != "" {
		http.Error(w, err, http.StatusInternalServerError)
		return
	} else if code == "" {
		url := config.AuthCodeURL("")
		c.Infof("url = %s", url)
		w.Header().Set("Location", url)
		w.WriteHeader(http.StatusFound)
		return
	} else {
		transport := &oauth.Transport{
			Config:    config,
			Transport: &urlfetch.Transport{Context: c},
		}
		token, err := transport.Exchange(code)
		c.Infof("token, err = %s, %s", token, err)
	}
}

func logout(w http.ResponseWriter, r *http.Request) {
	var url string
	c := appengine.NewContext(r)
	u := user.Current(c)
	if u != nil {
		var err error
		url, err = user.LogoutURL(c, r.URL.String())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		url = "/"
	}
	w.Header().Set("Location", url)
	w.WriteHeader(http.StatusFound)
	return
}

func root(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	u := user.Current(c)
	if u == nil {
		url, err := user.LoginURL(c, r.URL.String())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Location", url)
		w.WriteHeader(http.StatusFound)
		return
	}
	uc := &UserConfig{Context: c}
	_, err := uc.Token()
	if err != nil {
		c.Infof("Missing Picasa OAuth2 token: %s", err)
		w.Header().Set("Location", "/oauth2callback")
		w.WriteHeader(http.StatusFound)
		return
	}

	// Ancestor queries, as shown here, are strongly consistent with the High
	// Replication Datastore. Queries that span entity groups are eventually
	// consistent. If we omitted the .Ancestor from this query there would be
	// a slight chance that Greeting that had just been written would not
	// show up in a query.
	q := datastore.NewQuery("Greeting").Ancestor(guestbookKey(c)).Order("-Date").Limit(10)
	greetings := make([]Greeting, 0, 10)
	if _, err := q.GetAll(c, &greetings); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := guestbookTemplate.Execute(w, greetings); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

var guestbookTemplate = template.Must(template.New("book").Parse(guestbookTemplateHTML))

const guestbookTemplateHTML = `
<html>
  <body>
    {{range .}}
      {{with .Author}}
        <p><b>{{.}}</b> wrote:</p>
      {{else}}
        <p>An anonymous person wrote:</p>
      {{end}}
      <pre>{{.Content}}</pre>
    {{end}}
    <form action="/sign" method="post">
      <div><textarea name="content" rows="3" cols="60"></textarea></div>
      <div><input type="submit" value="Sign Guestbook"></div>
    </form>
    <form action="/logout" method="post">
      <div><input type="submit" value="Logout"></div>
    </form>
  </body>
</html>
`

func sign(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	u := user.Current(c)
	g := Greeting{
		Content: r.FormValue("content"),
		Date:    time.Now(),
	}
	g.Author = u.String()
	// We set the same parent key on every Greeting entity to ensure each Greeting
	// is in the same entity group. Queries across the single entity group
	// will be consistent. However, the write rate to a single entity group
	// should be limited to ~1/second.
	key := datastore.NewIncompleteKey(c, "Greeting", guestbookKey(c))
	_, err := datastore.Put(c, key, &g)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
}
