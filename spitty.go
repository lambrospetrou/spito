package main

import (
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
var validPath = regexp.MustCompile("^/(v|spitty[/]del)/(.+)$")

// BLOG HANDLERS
func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// test for general format
		m := validPath.FindStringSubmatch(r.URL.Path)
		if m == nil {
			http.NotFound(w, r)
			return
		}
		if !ValidateSpitID(m[2]) {
			http.Error(w, "Invalid Spit id.", http.StatusBadRequest)
			return
		}
		fn(w, r, m[2])
	}
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

		http.Redirect(w, r, "/v/"+spit.Id, http.StatusFound)
		return
	}
	http.Error(w, "Not supported method", http.StatusMethodNotAllowed)
	return
}

func viewHandler(w http.ResponseWriter, r *http.Request, id string) {
	// fetch the Spit with the requested id
	spit, err := LoadSpit(id)
	if err != nil {
		http.NotFound(w, r)
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

// show all posts
func rootHandler(w http.ResponseWriter, r *http.Request) {
	//fmt.Println("path: ", r.URL.Path)
	var id string = r.URL.Path[1:]
	if len(id) == 0 {
		// load the index page
		renderTemplate(w, "add", "")
		return
	}

	// make sure there is a valid Spit ID
	if !ValidateSpitID(id) {
		http.Error(w, "Invalid Spit id.", http.StatusBadRequest)
		return
	}

	// fetch the Spit with the requested id
	spit, err := LoadSpit(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// update clicks
	err = spit.ClickInc()
	if err != nil {
		http.Error(w, "Could not update analytics for Spit", http.StatusInternalServerError)
		return
	}

	// check if this Spit is a URL that we should redirect to
	if spit.IsURL {
		// HTTP 1.1.
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		// HTTP 1.0.
		w.Header().Set("Pragma", "no-cache")
		// Proxies
		w.Header().Set("Expires", "0")
		http.Redirect(w, r, spit.Content, http.StatusMovedPermanently)
		return
	}
	// this is a text Spit so display it
	http.Redirect(w, r, "/v/"+id, http.StatusFound)
	return
}

func main() {

	// add
	http.HandleFunc("/spitty/add", addHandler)

	// delete
	http.HandleFunc("/spitty/del/", makeHandler(deleteHandler))

	// view
	http.HandleFunc("/v/", makeHandler(viewHandler))

	// if there is a parameter Spit ID call action or just go to homepage
	http.HandleFunc("/", rootHandler)

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.ListenAndServe(":40090", nil)
}

//////////////////////// HELPERS ////////////////////
