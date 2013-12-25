package picasa

// TODO(tschroed):
// - Make it so that client id and secret are looked up from storage
// - Implement token caching
// - Tests, sucker!

import (
	"errors"
	"net/http"

	"appengine"
	"appengine/urlfetch"

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

type userConfig struct {
	Context appengine.Context
}

func (uc userConfig) Token() (*oauth.Token, error) {
	return nil, errors.New("Not implemented")
}

func (uc userConfig) PutToken(*oauth.Token) error {
	return errors.New("Not implemented")
}


func IsAuthorized(c appengine.Context) bool {
        return false
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
		redirectURL:  "http://lungworm.zweknu.org:8080/picasaauth",
		scope:        "https://picasaweb.google.com/data/",
		requestURL:   "https://www.googleapis.com/oauth2/v1/userinfo",
		authURL:      "https://accounts.google.com/o/oauth2/auth",
		tokenURL:     "https://accounts.google.com/o/oauth2/token",
	}
}

func Authorize(w http.ResponseWriter, r *http.Request) {
	code := r.FormValue("code")
	err := r.FormValue("error")
	c := appengine.NewContext(r)
	uc := &userConfig{Context: c}
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
