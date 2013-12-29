package picasa

// TODO(tschroed):
// - Make it so that client id and secret are looked up from storage
// - Tests, sucker!

import (
	"net/http"
	"time"

	"appengine"
	"appengine/datastore"
	"appengine/urlfetch"
	"appengine/user"

	"code.google.com/p/goauth2/oauth"
)

var (
	AuthPath = "/picasaauth"
	// TODO(tschroed): This should be dynamically calculated...
	// maybe from the referrer?
	RedirectURL = "http://localhost:8080" + AuthPath
)

type userConfig struct {
	context appengine.Context
}

type Token struct {
	AccessToken  string
	RefreshToken string
	Expiry       time.Time
}

func (uc userConfig) rootUserKey() *datastore.Key {
	return datastore.NewKey(uc.context, "string",
		user.Current(uc.context).String(), 0,
		nil)
}

func (uc userConfig) Token() (*oauth.Token, error) {
	k := datastore.NewKey(uc.context, "Token",
		"PicasaToken", 0,
		uc.rootUserKey())
	t := &Token{}
	if err := datastore.Get(uc.context, k, t); err != nil {
		uc.context.Errorf("Token() Error: %s", err)
		return nil, err
	}
	return &oauth.Token{
		AccessToken:  t.AccessToken,
		RefreshToken: t.RefreshToken,
		Expiry:       t.Expiry,
	}, nil
}

func (uc userConfig) PutToken(t *oauth.Token) error {
	k := datastore.NewKey(uc.context, "Token",
		"PicasaToken", 0,
		uc.rootUserKey())
	t0 := &Token{
		AccessToken:  t.AccessToken,
		RefreshToken: t.RefreshToken,
		Expiry:       t.Expiry,
	}
	if _, err := datastore.Put(uc.context, k, t0); err != nil {
		uc.context.Errorf("PutToken() Error: %s", err)
                return err
	}
        return nil
}

// TODO(tschroed): Ideally what we'd do here is provide a handler
// which would allow admin users to pass new id, secret, and
// redirect URL and then save that value in data store.
//
// The rest of the time it would be looked up.
func (uc *userConfig) newOauth2ClientConfig() *oauth.Config {
	return &oauth.Config{
		ClientId:     "864886111002-b0v7qc8f9lenaqo9bs7u3n7mkejvoc55.apps.googleusercontent.com",
		ClientSecret: "BV9-WnnEoByAFuTsGD2xN8PT",
		RedirectURL:  RedirectURL,
		Scope:        "https://picasaweb.google.com/data/",
		AuthURL:      "https://accounts.google.com/o/oauth2/auth",
		TokenURL:     "https://accounts.google.com/o/oauth2/token",
		AccessType:   "offline",
		TokenCache:   uc,
	}
}

// Note that this may force token renewal if expired.
func IsAuthorized(c appengine.Context) bool {
	uc := &userConfig{context: c}
	token, err := uc.Token()
	if err != nil {
		return false
	}
	if token.Expired() {
		transport := &oauth.Transport{
			Config:    uc.newOauth2ClientConfig(),
			Token:     token,
			Transport: &urlfetch.Transport{Context: c},
		}
		if err = transport.Refresh(); err != nil {
			c.Errorf("Error refreshing: %s", err)
			return false
		}
	}
	return true
}

func AuthorizeHandler(w http.ResponseWriter, r *http.Request) {
	code := r.FormValue("code")
	err := r.FormValue("error")
	c := appengine.NewContext(r)
	uc := &userConfig{context: c}
	config := uc.newOauth2ClientConfig()
        switch {
        case err != "":
		http.Error(w, err, http.StatusInternalServerError)
		return
        case code == "":
		url := config.AuthCodeURL("")
		w.Header().Set("Location", url)
		w.WriteHeader(http.StatusFound)
		return
        }
        transport := &oauth.Transport{
                Config:    config,
                Transport: &urlfetch.Transport{Context: c},
        }
        if _, err := transport.Exchange(code); err != nil {
                c.Errorf("Couldn't exchange code: %s", err)
	}
}
