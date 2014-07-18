package main

import (
	//"encoding/base64"
	"fmt"
	"github.com/lambrospetrou/goencoding/lpenc"
	"html/template"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type FooterStruct struct {
	Year int
}

type HeaderStruct struct {
	Title string
}

type TemplateBundle struct {
	Spit   *Spit
	Footer *FooterStruct
	Header *HeaderStruct
}

type TemplateBundleIndex struct {
	Spits  []*Spit
	Footer *FooterStruct
	Header *HeaderStruct
}

var templates = template.Must(template.ParseFiles(
	"templates/partials/header.html",
	"templates/partials/footer.html",
	"templates/view.html",
	"templates/add.html",
	"templates/add_success.html",
	"templates/index.html"))

func renderTemplate(w http.ResponseWriter, tmpl string, o interface{}) {
	// now we can call the correct template by the basename filename
	err := templates.ExecuteTemplate(w, tmpl+".html", o)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// v: view
// d: delete
var validPath = regexp.MustCompile("^/(v|d)/(.+)$")

// BLOG HANDLERS
func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// test for general format
		m := validPath.FindStringSubmatch(r.URL.Path)
		if m == nil {
			http.NotFound(w, r)
			return
		}
		/*
			// check for base64 valid id
			bytes, err := base64.URLEncoding.DecodeString(m[2])
			if err != nil {
				http.Error(w, "Invalid Spit id.", http.StatusBadRequest)
				return
			}
			_, err = strconv.ParseUint(string(bytes), 10, 64)
			if err != nil {
				http.Error(w, "Invalid Spit id.", http.StatusBadRequest)
				return
			}
		*/
		n, err := lpenc.Base62Encoding.Decode(m[2])
		fmt.Println(m[2], n, err)
		if err != nil {
			http.Error(w, "Invalid Spit id.", http.StatusBadRequest)
			return
		}
		fn(w, r, m[2])
	}
}

func viewHandler(w http.ResponseWriter, r *http.Request, id string) {
	// fetch the Spit with the requested id
	spit, err := LoadSpit(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	fmt.Println("ids: ", spit.IdRaw, spit.Id)
	// check if the Spit content is a URL

	if spit.IsURL {
		fmt.Println("valid url: ", spit.Content)
		http.Redirect(w, r, spit.Content, http.StatusMovedPermanently)
		return
	}

	// display the Spit
	bundle := &TemplateBundle{
		Spit:   spit,
		Footer: &FooterStruct{Year: time.Now().Year()},
		Header: &HeaderStruct{Title: spit.Id},
	}
	renderTemplate(w, "view", bundle)
}

func deleteHandler(w http.ResponseWriter, r *http.Request, id string) {
	spit, err := LoadSpit(id)
	if err != nil {
		http.Error(w, "Could not find the Spit specified!", http.StatusBadRequest)
		return
	}
	if err = spit.Del(); err != nil {
		http.Error(w, "Could not delete the spit specified!", http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

func addHandler(w http.ResponseWriter, r *http.Request) {
	if strings.ToLower(r.Method) == "get" {
		renderTemplate(w, "add", "")
		return
	} else if strings.ToLower(r.Method) == "post" {
		// TODO make sure the data we want exist

		// create the new Spit
		spit, err := NewSpit()
		fmt.Println("new: ", spit.Id, spit.IdRaw)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		spit.Exp, err = strconv.Atoi(r.FormValue("exp"))
		if err != nil {
			spit.Exp = 0
		}
		spit.Content = r.FormValue("content")
		spit.Save()

		bundle := &TemplateBundle{
			Spit:   spit,
			Footer: &FooterStruct{Year: time.Now().Year()},
			Header: &HeaderStruct{Title: spit.Id},
		}
		renderTemplate(w, "add_success", bundle)

		return
	}
	http.Error(w, "Not supported method", http.StatusMethodNotAllowed)
	return
}

// show all posts
func rootHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/a", http.StatusFound)
	return
	/*
		//fmt.Fprintf(w, "Hi there, I love you %s\n", r.URL.Path)
		spits, err := LoadAllSpits()
		if err != nil {
			http.Error(w, "Could not load spits", http.StatusInternalServerError)
			return
		}
		bundle := &TemplateBundleIndex{
			Footer: &FooterStruct{Year: time.Now().Year()},
			Header: &HeaderStruct{Title: "All spits"},
			Spits:  spits,
		}
		renderTemplate(w, "index", bundle)
	*/
}

func main() {

	// add
	http.HandleFunc("/a", addHandler)

	// delete
	http.HandleFunc("/d/", makeHandler(deleteHandler))

	// view
	http.HandleFunc("/v/", makeHandler(viewHandler))
	http.HandleFunc("/v/ls", rootHandler)
	http.HandleFunc("/", rootHandler)

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.ListenAndServe(":40090", nil)
}

//////////////////////// HELPERS ////////////////////
