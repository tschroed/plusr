package flickr

import (
	"net/http"

	"appengine"
	"appengine/datastore"
	"appengine/urlfetch"
	"appengine/user"

	"code.google.com/p/flickgo"
	"github.com/mrjones/oauth"
)

var (
	AuthPath       = "/flickrauth"
	ConfigPath     = "/flickrconfig"
	FlickrProvider = oauth.ServiceProvider{
		RequestTokenUrl:   "http://www.flickr.com/services/oauth/request_token",
		AuthorizeTokenUrl: "http://www.flickr.com/services/oauth/authorize",
		AccessTokenUrl:    "http://www.flickr.com/services/oauth/access_token",
	}
)

type flickrConfig struct {
	// Get the key and secret at http://www.flickr.com/services/apps/by/me
        // Use the config handler to save them to the datastore
	APIKey      string
	APISecret   string
	RedirectURL string
}

type keyValue struct {
	Key   string
	Value string
}

type AccessToken struct {
	Token          string
	Secret         string
	AdditionalData []keyValue
}

func copyAccessToken(atoken *oauth.AccessToken) *AccessToken {
	at := &AccessToken{
		Token:  atoken.Token,
		Secret: atoken.Secret,
	}
	at.AdditionalData = make([]keyValue, 0)
	for k, v := range atoken.AdditionalData {
		at.AdditionalData = append(at.AdditionalData, keyValue{k, v})
	}
	return at
}

func (a *AccessToken) toAccessToken() *oauth.AccessToken {
	at := &oauth.AccessToken{
		Token:  a.Token,
		Secret: a.Secret,
	}
	at.AdditionalData = make(map[string]string, 0)
	for _, kv := range a.AdditionalData {
		at.AdditionalData[kv.Key] = kv.Value
	}
	return at
}

type userConfig struct {
	context  appengine.Context
	rootUser string
}

func (uc *userConfig) rootUserKey() *datastore.Key {
	return datastore.NewKey(uc.context, "string", uc.rootUser, 0, nil)
}

func (uc *userConfig) accessToken() (*oauth.AccessToken, error) {
	atoken := &AccessToken{}
	akey := datastore.NewKey(uc.context, "AccessToken",
		"FlickrAccessToken", 0, uc.rootUserKey())
	if err := datastore.Get(uc.context, akey, atoken); err != nil {
		return nil, err
	}
	return atoken.toAccessToken(), nil
}

func (uc *userConfig) Printf(format string, a ...interface{}) (n int, err error) {
	uc.context.Infof(format, a...)
	return 0, nil
}

func (uc *userConfig) newFlickrConfig() *flickrConfig {
	return &flickrConfig{}
}

func (uc *userConfig) loadFlickrConfig() (*flickrConfig, error) {
	k := datastore.NewKey(uc.context, "flickrConfig",
		"FlickrConfig", 0, nil)
	f := &flickrConfig{}
	if err := datastore.Get(uc.context, k, f); err != nil {
		uc.context.Errorf("Flickr config load error: %s", err)
		return nil, err
	}
	return f, nil
}

func (uc *userConfig) saveFlickrConfig(c *flickrConfig) error {
	k := datastore.NewKey(uc.context, "flickrConfig",
		"FlickrConfig", 0, nil)
	if _, err := datastore.Put(uc.context, k, c); err != nil {
		return err
	}
	return nil
}


func MaybeGetAuth(c appengine.Context, u string) *userConfig {
	uc := &userConfig{context: c, rootUser: u}
	tok, _ := uc.accessToken()
	uc.context.Infof("AccessToken: %#v", tok)
	if tok == nil {
		return nil
	}
	return uc
}

func (uc *userConfig) oauthConsumer() *oauth.Consumer {
	c, err := uc.loadFlickrConfig()
	if err != nil {
		uc.context.Errorf("loadFlickrConfig: %s", err)
		return nil
	}
	consumer := oauth.NewConsumer(c.APIKey, c.APISecret, FlickrProvider)
	consumer.HttpClient = urlfetch.Client(uc.context)
	consumer.Logger = uc
	//	consumer.Debug(true)
	return consumer
}

func AuthorizeHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	uc := userConfig{context: c, rootUser: user.Current(c).String()}
	oauth_token := r.FormValue("oauth_token")
	oauth_verifier := r.FormValue("oauth_verifier")
	rkey := datastore.NewKey(uc.context, "oauth.RequestToken",
		"FlickrRequestToken", 0, uc.rootUserKey())
	consumer := uc.oauthConsumer()
	cfg, _ := uc.loadFlickrConfig()
	switch {
	case oauth_token == "":
		consumer.AdditionalAuthorizationUrlParams["perms"] = flickgo.WritePerm
		rtoken, loginUrl, err := consumer.GetRequestTokenAndUrl(cfg.RedirectURL)
		if err != nil {
			c.Errorf("Error: %v", err)
			return
		}
		uc.context.Infof("oauth.OAUTH_VERSION: %s", oauth.OAUTH_VERSION)
		uc.context.Infof("rtoken, loginUrl: %#v, %v", rtoken, loginUrl)
		if _, err := datastore.Put(uc.context, rkey, rtoken); err != nil {
			uc.context.Errorf("Datastore error: %v", err)
			return
		}
		http.Redirect(w, r, loginUrl, http.StatusFound)
	default:
		rtoken := new(oauth.RequestToken)
		if err := datastore.Get(uc.context, rkey, rtoken); err != nil {
			uc.context.Errorf("Datastore error: %v", err)
			return
		}
		uc.context.Infof("rtoken, oauth_token, oauth_verifier: %#v, %v, %v",
			rtoken, oauth_token, oauth_verifier)
		atoken, err := consumer.AuthorizeToken(rtoken, oauth_verifier)
		uc.context.Infof("atoken: %#v", atoken)
		if err != nil {
			uc.context.Errorf("AuthorizeToken error: %v", err)
			return
		}
		myAToken := copyAccessToken(atoken)
		uc.context.Infof("myAToken: %#v", myAToken)
		akey := datastore.NewKey(uc.context, "AccessToken",
			"FlickrAccessToken", 0, uc.rootUserKey())
		if _, err := datastore.Put(uc.context, akey, myAToken); err != nil {
			uc.context.Errorf("Datastore error: %v", err)
			return
		}
		if err := datastore.Delete(uc.context, rkey); err != nil {
			uc.context.Errorf("Datastore error: %v", err)
			return
		}
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
	config := uc.newFlickrConfig()
	config.APIKey = clientID
	config.APISecret = clientSecret
	config.RedirectURL = redirectURL
	uc.context.Infof("Saving Flickr config: %#v", config)
	if err := uc.saveFlickrConfig(config); err != nil {
		uc.context.Errorf("Error saving client config", err)
		return
	}
}
