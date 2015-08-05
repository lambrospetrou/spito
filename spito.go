package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/pat"
	"github.com/julienschmidt/httprouter"
	"github.com/lambrospetrou/spito/spit"
	"html/template"
	"log"
	"net/http"
	"runtime"
	"strings"
	"time"
)

var templates = template.Must(template.ParseFiles(
	"spitoweb/index.html"))

func renderTemplate(w http.ResponseWriter, tmpl string, o interface{}) {
	// now we can call the correct template by the basename filename
	err := templates.ExecuteTemplate(w, tmpl+".html", o)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

type APIResultError struct {
	Errors []string `json:"errors"`
}

type APIAddResult struct {
	Id                   string    `json:"id"`
	Exp                  int       `json:"expiration"`
	Content              string    `json:"content"`
	SpitType             string    `json:"spit_type"`
	DateCreated          time.Time `json:"date_created"`
	FormattedCreatedTime string    `json:"date_created_fmt"`
	IsURL                bool      `json:"is_url"`
	AbsoluteURL          string    `json:"absolute_url"`

	Message string `json: "message"`
}
type APIViewResult struct {
	Id                   string    `json:"id"`
	Exp                  int       `json:"expiration"`
	Content              string    `json:"content"`
	SpitType             string    `json:"spit_type"`
	DateCreated          time.Time `json:"date_created"`
	FormattedCreatedTime string    `json:"date_created_fmt"`
	IsURL                bool      `json:"is_url"`
	AbsoluteURL          string    `json:"absolute_url"`
	Clicks               uint64    `json:"clicks"`

	Message string `json: "message"`
}

type APIDeleteResult struct {
	Message string `json: "message"`
}

func requireSpitID(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// test for general format
		id := r.URL.Query().Get(":id")
		if !spit.ValidateSpitID(id) {
			http.Error(w, "Invalid Spit id.", http.StatusBadRequest)
			return
		}
		fn(w, r, id)
	}
}

func requireSpitIDHttpRouter(fn func(http.ResponseWriter, *http.Request, string)) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		// test for general format
		id := ps.ByName("id")
		if !spit.ValidateSpitID(id) {
			http.Error(w, "Invalid Spit id.", http.StatusBadRequest)
			return
		}
		fn(w, r, id)
	}
}

func httpRouterNoParams(h http.Handler) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		h.ServeHTTP(w, r)
	}
}
func httpRouterNoParamsFn(fn func(http.ResponseWriter, *http.Request)) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		fn(w, r)
	}
}

func apiDeleteHandler(w http.ResponseWriter, r *http.Request, id string) {
	s, err := spit.Load(id)
	if err != nil {
		http.Error(w, "Could not find the Spit specified!", http.StatusBadRequest)
		return
	}
	if err = s.Del(); err != nil {
		http.Error(w, "Could not delete the spit specified!", http.StatusBadRequest)
		return
	}
	result := &APIDeleteResult{Message: "Successfully deleted Spit: " + s.Id()}
	b, e := json.Marshal(result)
	if e != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	w.Write(b)
	return
}

func apiAddHandler(w http.ResponseWriter, r *http.Request) {
	if strings.ToLower(r.Method) == "post" {
		s, err := CoreAddMultiSpit(r)

		if err != nil {
			if _, ok := err.(*ErrCoreAdd); ok {
				// it was an error during request validation
				validationRes := err.(*ErrCoreAdd)
				errorList := make([]string, 0)
				for _, v := range validationRes.Errors {
					if len(strings.TrimSpace(v)) > 0 {
						errorList = append(errorList, v)
					}
				}
				result := &APIResultError{Errors: errorList}
				b, e := json.Marshal(result)
				if e != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json; charset=UTF-8")
				w.WriteHeader(http.StatusBadRequest)
				w.Write(b)
				return
			} else if _, ok := err.(*ErrCoreAddDB); ok {
				errDB := err.(*ErrCoreAddDB)
				log.Printf("apiAddHandler::Internal error: %v", errDB)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			} else {
				// other internal error
				http.Error(w, err.Error(), http.StatusInternalServerError)
				log.Fatalf("apiAddHandler::Unknown Error: %v", err)
				return

			}
		}

		// we are good to go - spit added successfully
		result := &APIAddResult{
			Id: s.Id(), Exp: s.Exp(), Content: s.Content(), SpitType: s.SpitType(),
			DateCreated: s.DateCreated(), FormattedCreatedTime: s.FormattedCreatedTime(),
			IsURL: s.IsURL(), AbsoluteURL: s.AbsoluteURL(), Message: "Successfully added new Spit!",
		}
		b, err := json.Marshal(result)
		if err != nil {
			log.Printf("apiAddHandler::Internal error while marshalling spit: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		w.Write(b)
		return
	}
	http.Error(w, "Not supported method", http.StatusMethodNotAllowed)
	return
}

func apiViewHandler(w http.ResponseWriter, r *http.Request, id string) {
	// fetch the Spit with the requested id
	s, err := spit.Load(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// we are good to go - spit fetched successfully
	result := &APIViewResult{
		Id: s.Id(), Exp: s.Exp(), Content: s.Content(), SpitType: s.SpitType(),
		DateCreated: s.DateCreated(), FormattedCreatedTime: s.FormattedCreatedTime(),
		IsURL: s.IsURL(), AbsoluteURL: s.AbsoluteURL(), Clicks: s.Clicks(),
		Message: "Successfully fetched Spit!",
	}
	b, err := json.Marshal(result)
	if err != nil {
		log.Printf("apiViewHandler::Internal error while marshalling spit: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	w.Write(b)
	return
}

// tries to find the Spit with the passed ID and either redirects to it if it is a URL
// or it goes to the Spit viewer
func webRedirectHandler(w http.ResponseWriter, r *http.Request, id string) {
	// make sure there is a valid Spit ID
	if !spit.ValidateSpitID(id) {
		http.Error(w, "Invalid Spit id.", http.StatusBadRequest)
		return
	}

	// fetch the Spit with the requested id
	s, err := spit.Load(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// update clicks
	err = s.ClickInc()
	if err != nil {
		http.Error(w, "Could not update analytics for Spit", http.StatusInternalServerError)
		return
	}

	// check if this Spit is a URL that we should redirect to
	if s.IsURL() {
		// HTTP 1.1.
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		// HTTP 1.0.
		w.Header().Set("Pragma", "no-cache")
		// Proxies
		w.Header().Set("Expires", "0")
		http.Redirect(w, r, s.Content(), http.StatusMovedPermanently)
		return
	}
	// this is a text Spit so display it
	http.Redirect(w, r, "#/view/"+id, http.StatusFound)
	//http.Redirect(w, r, "http://cyari.es/spito/#/view/"+id, http.StatusFound)
	return
}

func limitSizeHandler(fn func(http.ResponseWriter, *http.Request),
	size int64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, size)
		fn(w, r)
	}
}

// either parse the Spit ID or if the path is /#/something go to the website
func rootHandler(w http.ResponseWriter, r *http.Request) {
	//fmt.Println("path: ", r.URL.Path)

	// check if the path contains SpitID or the /#/xxx which means go to the website
	var id string = r.URL.Path[1:]
	if len(id) == 0 {
		// load the index page
		//renderTemplate(w, "add", nil)
		// check if this Spit is a URL that we should redirect to

		// HTTP 1.1.
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		// HTTP 1.0.
		w.Header().Set("Pragma", "no-cache")
		// Proxies
		w.Header().Set("Expires", "0")
		//http.Redirect(w, r, "http://cyari.es/spito/", http.StatusTemporaryRedirect)
		renderTemplate(w, "index", nil)
		return
	}
	// make sure there is a valid Spit ID
	webRedirectHandler(w, r, id)
}

/*
	Access-Control-Allow-Origin: http://foo.example
	Access-Control-Allow-Methods: POST, GET, OPTIONS
	Access-Control-Allow-Headers: X-PINGOTHER
	Access-Control-Max-Age: 1728000
*/
func CORSEnable(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if CORSAllowedOrigins[r.Header.Get("Origin")] {
			w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "X-Spito, Content-type")
			w.Header().Set("Access-Control-Max-Age", "1728000")
		}
		fn(w, r)
	}
}

func OKHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "")
}

func main() {

	fmt.Println("Starting Spito at: 40090")

	// use all the available cores
	runtime.GOMAXPROCS(runtime.NumCPU())

	//router := httprouter.New()
	router := pat.New()

	/////////////////
	// API ROUTERS
	/////////////////

	router.Add("OPTIONS", "/", CORSEnable(OKHandler))

	router.Get("/api/v1/spits/{id}", CORSEnable(requireSpitID(apiViewHandler)))
	router.Post("/api/v1/spits", CORSEnable(limitSizeHandler(apiAddHandler, MAX_FORM_SIZE)))

	// TODO - make these internal
	router.Delete("/api/v1/spits/{id}", CORSEnable(requireSpitID(apiDeleteHandler)))
	//router.Post("/api/v1-web/spits/{id}/delete", CORSEnable(limitSizeHandler(requireSpitID(webDeleteHandler), MAX_FORM_SIZE)))

	/////////////////
	// VIEW ROUTERS
	/////////////////
	router.Get("/", rootHandler)
	http.Handle("/", router)

	/**
	 *	SINGLE LETTER (maybe 2-letter too) DOMAINS ARE RESERVED FOR INTERNAL USAGE
	 */
	// downloads handler - /d/
	fs_d := http.FileServer(http.Dir("downloads"))
	http.Handle("/d/", http.StripPrefix("/d/", fs_d))
	// static files handler - /s/
	fs_s := http.FileServer(http.Dir("spitoweb/s"))
	http.Handle("/s/", http.StripPrefix("/s/", fs_s))
	//router.ServeFiles("/static/*filepath", http.Dir("static"))

	http.ListenAndServe(":40090", nil)
	//http.ListenAndServe(":40090", router)
}

//////////////////////// HELPERS ////////////////////
