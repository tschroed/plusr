// Once deployed:
//   / to auth as the root user (if not already done)
//   /flickrconfig to setup a null flickr config (use cloud console
//       datastore editor to change)
//   /picasaconfig to setup a null picasa config (use cloud console
//       datastore editor to change)
//   / to rerun flickr and picasa authorization flows
//   /readfeed to sync photos
//
//  Edit cron.yaml with /readfeed?user=<user> to automate.
package plusr

import (
	"html/template"
	"net/http"
	"time"

	"appengine"
	"appengine/datastore"
	"appengine/user"

	"flickr"
	"picasa"
)

type Greeting struct {
	Author  string
	Content string
	Date    time.Time
}

func init() {
	http.HandleFunc("/", root)
	http.HandleFunc(picasa.AuthPath, picasa.AuthorizeHandler)
	http.HandleFunc(picasa.ConfigPath, picasa.ConfigHandler)
	http.HandleFunc(flickr.AuthPath, flickr.AuthorizeHandler)
	http.HandleFunc(flickr.ConfigPath, flickr.ConfigHandler)
	http.HandleFunc("/garbage", flickr.UploadGarbage)
	http.HandleFunc("/logout", logout)
	http.HandleFunc("/readfeed", photoFeedHandler)
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

func logout(w http.ResponseWriter, r *http.Request) {
	var url string
	c := appengine.NewContext(r)
	if u := user.Current(c); u != nil {
		var err error
		if url, err = user.LogoutURL(c, r.URL.String()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		url = "/"
	}
	http.Redirect(w, r, url, http.StatusFound)
	return
}

func photoFeedHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	username := r.FormValue("user")
	if username == "" {
		username = user.Current(c).String()
	}
	pAuth := picasa.MaybeGetAuth(c, username)
	if pAuth == nil {
		c.Errorf("Unable to get authentication blob.")
		return
	}
	phIn := make(chan *picasa.Photo)
	doneIn := make(chan bool)
	source := picasa.NewPhotoSource(pAuth, phIn, doneIn)
	fAuth := flickr.MaybeGetAuth(c, username)
	if pAuth == nil {
		c.Errorf("Unable to get authentication blob.")
		return
	}
	phOut := make(chan flickr.Photo)
	doneOut := make(chan bool)
	sink := flickr.NewPhotoSink(fAuth, phOut, doneOut)
	flickr.NewPhotoSink(fAuth, phOut, doneOut)
	go source.Loop()
	go sink.Loop()
	for {
		select {
		case p := <-phIn:
			c.Infof("Photo: %s (%s)", p.Title, p.Contents.Src)
			phOut <- p
		case <-doneIn:
			c.Infof("Those are all the photos.")
			doneOut <- true
			return
		}
	}
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
		http.Redirect(w, r, url, http.StatusFound)
		return
	}
	if picasa.MaybeGetAuth(c, u.String()) == nil {
		c.Infof("Picasa is not authorized.")
		http.Redirect(w, r, picasa.AuthPath, http.StatusFound)
		return
	}
	if flickr.MaybeGetAuth(c, u.String()) == nil {
		c.Infof("Flickr is not authorized.")
		http.Redirect(w, r, flickr.AuthPath, http.StatusFound)
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
