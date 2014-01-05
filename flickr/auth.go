package flickr

import (
        "net/http"

        "appengine"
        "appengine/urlfetch"

// Uses the legacy authentication mechanism
//        "code.google.com/p/flickgo"
        "github.com/mrjones/oauth"
)

var (
        AuthPath = "/flickrauth"
        FlickrProvider = oauth.ServiceProvider{
                RequestTokenUrl: "http://www.flickr.com/services/oauth/request_token",
                AuthorizeTokenUrl: "http://www.flickr.com/services/oauth/authorize",
                AccessTokenUrl: "http://www.flickr.com/services/oauth/access_token",
        }
        RedirectURL = "http://localhost:8080" + AuthPath
)

func AuthorizeHandler(w http.ResponseWriter, r *http.Request) {
        c := appengine.NewContext(r)
        // Get the key and secret at http://www.flickr.com/services/apps/by/me
        consumer := oauth.NewConsumer("--API KEY--",
                                      "--SECRET--", FlickrProvider)
        consumer.HttpClient = urlfetch.Client(c)
        rtoken, loginUrl, err := consumer.GetRequestTokenAndUrl(RedirectURL)
        if err != nil {
                c.Errorf("Error: %v", err)
                return
        }
        c.Infof("oauth.OAUTH_VERSION: %s", oauth.OAUTH_VERSION)
        c.Infof("rtoken, loginUrl: %#v, %v", rtoken, loginUrl)
}
