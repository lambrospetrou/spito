package main

import (
	"html/template"
	"net/http"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

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
		renderTemplate(w, "add", nil)
		return
	} else if strings.ToLower(r.Method) == "post" {
		result := struct {
			InputExp     string
			InputContent string
			Errors       map[string]string
		}{}

		// do the validation of the parameters
		result.Errors = ValidateSpitParameters(r)

		// if we have errors display the add page again
		if len(result.Errors) > 0 {
			result.InputContent = r.PostFormValue("content")
			result.InputExp = r.PostFormValue("exp")
			renderTemplate(w, "add", &result)
			return
		}

		// create the new Spit since everything is fine
		spit, err := NewSpit()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		spit.Exp, _ = strconv.Atoi(r.PostFormValue("exp"))
		spit.Content = strings.TrimSpace(r.PostFormValue("content"))

		// Save the spit and return the view page
		if err = spit.Save(); err != nil {
			http.Error(w, "Could not create your spit, go back and try again",
				http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/v/"+spit.Id, http.StatusFound)
		return
	} // end of POST
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
	bundle := &struct {
		Spit   *Spit
		Footer *struct{ Year int }
		Header *struct{ Title string }
	}{
		Spit:   spit,
		Footer: &struct{ Year int }{Year: time.Now().Year()},
		Header: &struct{ Title string }{Title: spit.Id},
	}
	renderTemplate(w, "view", bundle)
}

// show all posts
func rootHandler(w http.ResponseWriter, r *http.Request) {
	//fmt.Println("path: ", r.URL.Path)
	var id string = r.URL.Path[1:]
	if len(id) == 0 {
		// load the index page
		renderTemplate(w, "add", nil)
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

	runtime.GOMAXPROCS(runtime.NumCPU())

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
