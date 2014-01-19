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
	AuthPath   = "/picasaauth"
	ConfigPath = "/picasaconfig"
)

type picasaConfig struct {
	ClientId     string
	ClientSecret string
	RedirectURL  string
}

type userConfig struct {
	context  appengine.Context
	rootUser string
}

type Token struct {
	AccessToken  string
	RefreshToken string
	Expiry       time.Time
}

func (uc userConfig) rootUserKey() *datastore.Key {
	return datastore.NewKey(uc.context, "string", uc.rootUser, 0, nil)
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

func (uc *userConfig) newOauth2ClientConfig() *oauth.Config {
	return &oauth.Config{
		ClientId:     "",
		ClientSecret: "",
		RedirectURL:  "",
		Scope:        "https://picasaweb.google.com/data/",
		AuthURL:      "https://accounts.google.com/o/oauth2/auth",
		TokenURL:     "https://accounts.google.com/o/oauth2/token",
		AccessType:   "offline",
		TokenCache:   uc,
	}
}

func (uc *userConfig) loadOauth2ClientConfig() (*oauth.Config, error) {
	k := datastore.NewKey(uc.context, "picasaConfig",
		"PicasaConfig", 0, nil)
	p := &picasaConfig{}
	if err := datastore.Get(uc.context, k, p); err != nil {
		uc.context.Errorf("Picasa config load error: %s", err)
		return nil, err
	}
	c := uc.newOauth2ClientConfig()
	c.ClientId = p.ClientId
	c.ClientSecret = p.ClientSecret
	c.RedirectURL = p.RedirectURL
	return c, nil
}

func (uc *userConfig) saveOauth2ClientConfig(c *oauth.Config) error {
	k := datastore.NewKey(uc.context, "picasaConfig",
		"PicasaConfig", 0, nil)
	p := &picasaConfig{
		ClientId:     c.ClientId,
		ClientSecret: c.ClientSecret,
		RedirectURL:  c.RedirectURL,
	}
	if _, err := datastore.Put(uc.context, k, p); err != nil {
		return err
	}
	return nil
}

// Note that this may force token renewal if expired.
func MaybeGetAuth(c appengine.Context, u string) *userConfig {
	uc := &userConfig{context: c, rootUser: u}
	token, err := uc.Token()
	if err != nil {
		return nil
	}
	config, err := uc.loadOauth2ClientConfig()
	if err != nil {
		return nil
	}
	if token.Expired() {
		transport := &oauth.Transport{
			Config:    config,
			Token:     token,
			Transport: &urlfetch.Transport{Context: c},
		}
		if err = transport.Refresh(); err != nil {
			c.Errorf("Error refreshing: %s", err)
			return nil
		}
	}
	return uc
}

// Must execute in the context of a live user request.
func AuthorizeHandler(w http.ResponseWriter, r *http.Request) {
	code := r.FormValue("code")
	form_err := r.FormValue("error")
	c := appengine.NewContext(r)
	uc := &userConfig{context: c, rootUser: user.Current(c).String()}
	config, err := uc.loadOauth2ClientConfig()
	if err != nil {
		return
	}
	switch {
	case form_err != "":
		http.Error(w, form_err, http.StatusInternalServerError)
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

func ConfigHandler(w http.ResponseWriter, r *http.Request) {
	clientID := r.FormValue("client")
	clientSecret := r.FormValue("secret")
	redirectURL := r.FormValue("redirect")
	c := appengine.NewContext(r)
	uc := &userConfig{context: c, rootUser: user.Current(c).String()}
	if !user.IsAdmin(c) {
		uc.context.Errorf("Attempt by non-admin user (%s) to set config",
			user.Current(c).String())
		return
	}
	config := uc.newOauth2ClientConfig()
	config.ClientId = clientID
	config.ClientSecret = clientSecret
	config.RedirectURL = redirectURL
	uc.context.Infof("Saving Picasa config: %#v", config)
	if err := uc.saveOauth2ClientConfig(config); err != nil {
		uc.context.Errorf("Error saving client config", err)
		return
	}
}
