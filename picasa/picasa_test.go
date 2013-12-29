package picasa

import (
	"io/ioutil"
	"log"
	"testing"
)

func TestParseFeed(t *testing.T) {
	xml_fname := "testdata/tag-search.xml"
	text, err := ioutil.ReadFile(xml_fname)
	if err != nil {
		log.Fatalf("Failed to read %s: %s", xml_fname, err)
	}
	feed := parseFeed(text)
	if len(feed.Photos) != 2 {
		t.Errorf("Feed had > 2 items: %d", len(feed.Photos))
	}
	p0 := Photo{
		Title: "IMG_20131210_125726.jpg",
		Id:    "5955929363163475490",
		Contents: Content{
			Src: "https://lh4.googleusercontent.com/-JPlXi5pFopE/Uqe2My6tbiI/AAAAAAAAHbI/urCIvjBmLv0/IMG_20131210_125726.jpg"},
		Version: "8561",
		Album:   "12/10/13",
	}
	p1 := Photo{
		Title: "IMG_20131210_120040.jpg",
		Id:    "5955929366130897234",
		Album: "12/10/13",
		Contents: Content{
			Src: "https://lh3.googleusercontent.com/-YR8R3keyKJM/Uqe2M9-MsVI/AAAAAAAAHbE/00IfRoh6Eso/IMG_20131210_120040.jpg"},
		Version: "8560",
	}
	if p0 != feed.Photos[0] {
		t.Errorf("%s != %s", p0, feed.Photos[0])
	}
	if p1 != feed.Photos[1] {
		t.Errorf("%s != %s", p1, feed.Photos[1])
	}
}
