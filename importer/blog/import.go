package blog

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/PuerkitoBio/purell"
	log "github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/robbiet480/go-wordpress"
	"github.com/tomnomnom/linkheader"
	"github.com/volatiletech/sqlboiler/queries/qm"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/Bnei-Baruch/mdb/models"
	"github.com/Bnei-Baruch/mdb/utils"
	"github.com/Bnei-Baruch/mdb/version"
)

var sneakyBustards = make([]string, 0)

func Import() {
	clock, _ := Init()

	//for _, v := range allBlogs {
	//	currentBlog = v
	//	utils.Must(doImport())
	//}
	utils.Must(cleanAllPosts())

	log.Infof("%d sneaky bustards", len(sneakyBustards))
	sort.Slice(sneakyBustards, func(i, j int) bool {
		return sneakyBustards[i] < sneakyBustards[j]
	})
	for i := range sneakyBustards {
		log.Info(sneakyBustards[i])
	}

	Shutdown()
	log.Info("Success")
	log.Infof("Total run time: %s", time.Now().Sub(clock).String())
}

func doImport() error {
	log.Infof("doImport: %s", currentBlog.Name)
	postFilter := getBlogPostFilter(currentBlog.ID)

	walkFn := func(path string, info os.FileInfo, err error) error {
		f, err := os.Open(path)
		if err != nil {
			return errors.Wrapf(err, "os.Open %s", path)
		}

		var post wordpress.Post
		err = json.NewDecoder(f).Decode(&post)
		if err != nil {
			return errors.Wrapf(err, "json.Decode %s", path)
		}

		blogPost, err := prepare(&post)
		if err != nil {
			return errors.Wrapf(err, "prepare %s", path)
		}

		blogPost.BlogID = currentBlog.ID
		blogPost.Filtered = !postFilter.IsPass(&post)
		err = blogPost.Insert(mdb)
		if err != nil {
			return errors.Wrapf(err, "blogPost.Insert %s", path)
		}

		return nil
	}

	err := traverse(walkFn)
	if err != nil {
		fmt.Printf("traverse error: %v\n", err)
	}

	return nil
}

func prepare(post *wordpress.Post) (*models.BlogPost, error) {
	ctxNode := html.Node{
		Type:     html.ElementNode,
		DataAtom: atom.Body,
		Data:     "body",
	}
	var sb strings.Builder

	content := post.Content.Rendered
	if strings.Contains(content, "http://youtube") {
		content = strings.ReplaceAll(content, "http://youtube", "https://youtube")
	}
	nodes, err := html.ParseFragment(strings.NewReader(content), &ctxNode)
	if err != nil {
		return nil, errors.Wrapf(err, "html.Parse %d", post.ID)
	}
	for i := range nodes {
		node := nodes[i]

		if skipNode(node) {
			continue
		}

		if stopNode(node) {
			break
		}

		cleanNode(node)
		html.Render(&sb, node)
	}

	bp := &models.BlogPost{
		WPID:     int64(post.ID),
		PostedAt: post.DateGMT.Time,
		Title:    post.Title.Rendered,
		Content:  sb.String(),
		Link:     post.Link,
	}

	return bp, nil
}

func isShareDaddy(node *html.Node) bool {
	if node.DataAtom != atom.Div {
		return false
	}

	for _, a := range node.Attr {
		if a.Key == "class" {
			classes := strings.Fields(strings.TrimSpace(a.Val))
			for i := range classes {
				if classes[i] == "sharedaddy" {
					return true
				}
			}
			break
		}
	}

	return false
}

func isPostMetadata(node *html.Node) bool {
	if node.DataAtom != atom.P {
		return false
	}

	for _, a := range node.Attr {
		if a.Key == "class" {
			classes := strings.Fields(strings.TrimSpace(a.Val))
			for i := range classes {
				if classes[i] == "postmetadata" || classes[i] == "postmetadata_links" {
					return true
				}
			}
			break
		}
	}

	return false
}

func isScript(node *html.Node) bool {
	return node.DataAtom == atom.Script
}

func cleanNode(node *html.Node) {
	// clean children
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		if isScript(c) {
			node.RemoveChild(c)
		}
	}
}

func skipNode(node *html.Node) bool {
	return isScript(node)
}

func stopNode(node *html.Node) bool {
	return isShareDaddy(node) || isPostMetadata(node)
}

func stdLink(link string) string {
	ltLink := strings.TrimSpace(link)
	pUrl, err := url.Parse(ltLink)
	if err != nil {
		log.Warnf("stdLink url.Parse: %s", err.Error())
		return strings.ToLower(ltLink)
	}

	return purell.NormalizeURL(pUrl, purell.FlagsUnsafeGreedy^purell.FlagRemoveFragment)
}

var linkPostMap map[string]*models.BlogPost

func loadLinkMap() error {
	posts, err := models.BlogPosts(mdb, qm.Select("id", "link", "wp_id", "blog_id")).All()
	if err != nil {
		return errors.Wrap(err, "Load posts from DB")
	}

	linkPostMap = make(map[string]*models.BlogPost, len(posts))
	for i := range posts {
		linkPostMap[stdLink(posts[i].Link)] = posts[i]
	}
	log.Infof("LinkMap: %d posts %d links [%t]", len(posts), len(linkPostMap), len(posts) == len(linkPostMap))

	return nil
}

func cleanAllPosts() error {
	err := loadLinkMap()
	if err != nil {
		return errors.Wrap(err, "loadLinkMap")
	}

	posts, err := models.BlogPosts(mdb).All()
	if err != nil {
		return errors.Wrap(err, "Load posts from DB")
	}

	for i := range posts {
		err := cleanPost(posts[i])
		if err != nil {
			return errors.Wrapf(err, "cleanPost [%d]", posts[i].ID)
		}
	}
	return nil
}

func cleanPost(post *models.BlogPost) error {
	ctxNode := html.Node{
		Type:     html.ElementNode,
		DataAtom: atom.Body,
		Data:     "body",
	}
	content := post.Content
	if strings.Contains(content, "http://youtube") {
		content = strings.ReplaceAll(content, "http://youtube", "https://youtube")
	}

	nodes, err := html.ParseFragment(strings.NewReader(content), &ctxNode)
	if err != nil {
		return errors.Wrapf(err, "html.ParseFragment %d", post.ID)
	}

	var sb strings.Builder
	for i := range nodes {
		traverseHtmlNode(nodes[i], makeNodeRelativeLinks)
		traverseHtmlNode(nodes[i], makeNodeSecureDomains)
		html.Render(&sb, nodes[i])
	}
	post.Content = sb.String()

	err = post.Update(mdb, "content")
	if err != nil {
		return errors.Wrapf(err, "post.Update %d", post.ID)
	}

	return nil
}

func traverseHtmlNode(node *html.Node, fn func(node *html.Node)) {
	fn(node)
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		traverseHtmlNode(c, fn)
	}
}

var HOST_RE = regexp.MustCompile(`laitman\.(com|es|ru|co\.il)`)
var MEDIA_RE = regexp.MustCompile(`\.(jpg|jpeg|png|gif|bmp|mp3|mp4|wmv|flv|pdf|zip)`)
var TERM_RE = regexp.MustCompile(`/(category|topics|tag)`)
var SHORTNER_RE = regexp.MustCompile(`(?i)^/[a-z0-9]{5,8}$`)
var POST_ID_HTML_RE = regexp.MustCompile(`(?i)^/(.*)/([0-9]+)\.html$`)

var REL_FMT_INT = "/publications/blog/%s/%d"
var REL_FMT_STR = "/publications/blog/%s/%s"

func hostToBlogName(host string) string {
	return strings.Replace(host, ".", "-", -1)
}

func makeNodeRelativeLinks(node *html.Node) {
	if node.DataAtom != atom.A {
		return
	}

	for i := range node.Attr {
		if node.Attr[i].Key == "href" {
			// normalize url for comparison
			v := stdLink(node.Attr[i].Val)

			// check post links map
			if post, ok := linkPostMap[v]; ok {
				relative := fmt.Sprintf(REL_FMT_INT, allBlogs[post.BlogID].Name, post.WPID)
				log.Infof("makeNodeRelativeLinks %s => %s", node.Attr[i].Val, relative)
				node.Attr[i].Val = relative
				break
			}

			// not in links map. try doing magic
			pUrl, err := url.Parse(v)
			if err != nil {
				log.Warnf("url.Parse: %s", err.Error())
				break
			}

			// ignore:
			// * non post urls
			// * fragments (mostly comments)
			// * empty paths
			if !HOST_RE.MatchString(pUrl.Host) ||
				MEDIA_RE.MatchString(pUrl.Path) ||
				TERM_RE.MatchString(pUrl.Path) ||
				pUrl.Fragment != "" ||
				pUrl.Path == "" {
				break
			}

			// default wordpress short url ?
			if id := pUrl.Query().Get("p"); id != "" {
				relative := fmt.Sprintf(REL_FMT_STR, hostToBlogName(pUrl.Host), id)
				log.Infof("makeNodeRelativeLinks %s => %s", node.Attr[i].Val, relative)
				node.Attr[i].Val = relative
				break
			}

			// is post id in path ?
			// example: https://www.laitman.ru/kabbalah-religion/144454.html
			if m := POST_ID_HTML_RE.FindStringSubmatch(pUrl.Path); len(m) > 0 {
				relative := fmt.Sprintf(REL_FMT_STR, hostToBlogName(pUrl.Host), m[len(m)-1])
				log.Infof("makeNodeRelativeLinks %s => %s", node.Attr[i].Val, relative)
				node.Attr[i].Val = relative
				break
			}

			// Looks like a url shortner?
			if SHORTNER_RE.MatchString(pUrl.Path) {
				// Try to follow redirects..
				destUrl, err := NewUrlUnShortner().Follow(pUrl.String())
				if err != nil {
					log.Warnf("Error following short url %s: %s", pUrl.String(), err.Error())
					break
				}

				pDestUrl, err := url.Parse(destUrl)
				if err != nil {
					log.Warnf("url.Parse [followed]: %s", err.Error())
					break
				}

				// default wordpress short url ?
				if id := pDestUrl.Query().Get("p"); id != "" {
					relative := fmt.Sprintf(REL_FMT_STR, hostToBlogName(pDestUrl.Host), id)
					log.Infof("makeNodeRelativeLinks %s => %s", node.Attr[i].Val, relative)
					node.Attr[i].Val = relative
					break
				}

				// check post links map
				if post, ok := linkPostMap[stdLink(destUrl)]; ok {
					relative := fmt.Sprintf(REL_FMT_INT, allBlogs[post.BlogID].Name, post.WPID)
					log.Infof("makeNodeRelativeLinks %s => %s", node.Attr[i].Val, relative)
					node.Attr[i].Val = relative
					break
				}

				log.Infof("Followed url to nowhere interesting: %s", destUrl)
				break
			}

			// nothing really we can do...
			sneakyBustards = append(sneakyBustards, v)
			//log.Errorf("sneaky bustard: %s", v)

			break
		}
	}
}

type UrlUnShortner struct {
	c *http.Client
}

func NewUrlUnShortner() *UrlUnShortner {
	s := new(UrlUnShortner)
	s.c = &http.Client{
		Timeout: time.Second * 10,
	}
	return s
}

func (s *UrlUnShortner) Follow(url string) (string, error) {
	log.Infof("UrlUnShortner.Follow %s", url)

	// prepare HEAD request
	req, err := http.NewRequest(http.MethodHead, url, nil)
	if err != nil {
		return "", errors.Wrap(err, "UrlUnShortner.Follow http.NewRequest")
	}
	req.Header.Set("User-Agent", fmt.Sprintf("BB Archive (MDB %s)", version.Version))

	// do actual request
	resp, err := s.c.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "UrlUnShortner.Follow http.Client.Do")
	}
	if resp.StatusCode != http.StatusOK {
		return "", errors.Errorf("Bad status %d", resp.StatusCode)
	}

	// try to use Link rel="shortlink" header
	if link, ok := resp.Header["Link"]; ok {
		links := linkheader.ParseMultiple(link)
		links = links.FilterByRel("shortlink")
		if len(links) > 0 && links[0].URL != url {
			return links[0].URL, nil
		}
	}

	// return last redirect hop url
	return resp.Request.URL.String(), nil
}

func makeNodeSecureDomains(node *html.Node) {
	for i := range node.Attr {
		switch node.Attr[i].Key {
		case "src":
			sUrl, err := secureDomain(node.Attr[i].Val)
			if err != nil {
				log.Warnf("secureDomain: %s", err.Error())
			} else {
				node.Attr[i].Val = sUrl
			}
		case "srcset":
			s := strings.Split(node.Attr[i].Val, ",")
			for j := range s {
				sj := strings.SplitN(strings.TrimSpace(s[j]), " ", 2)
				sUrl, err := secureDomain(sj[0])
				if err != nil {
					log.Warnf("secureDomain: %s", err.Error())
				} else {
					sj[0] = sUrl
				}
				s[j] = strings.Join(sj, " ")
			}
			node.Attr[i].Val = strings.Join(s, ", ")
		default:
			continue
		}
	}
}

func secureDomain(urlVal string) (string, error) {
	pUrl, err := url.Parse(strings.TrimSpace(urlVal))
	if err != nil {
		return "", errors.Wrap(err, "url.Parse")
	}

	// ignore:
	// * non our urls
	if !HOST_RE.MatchString(pUrl.Host) {
		return urlVal, nil
	}

	// make sure wer'e secure
	pUrl.Scheme = "https"

	// remove subdomains
	s := strings.Split(pUrl.Host, ".")
	if len(s) > 2 {
		for i := range s {
			if s[i] == "laitman" {
				pUrl.Host = strings.Join(s[i:], ".")
				break
			}
		}
	}

	return pUrl.String(), nil
}
