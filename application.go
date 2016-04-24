package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/gorilla/pat"
	"github.com/lambrospetrou/spito/spit"
)

type APIResultError struct {
	Errors []string `json:"errors"`
}

type APIAddResult struct {
	Id             string `json:"id"`
	Content        string `json:"content"`
	SpitType       string `json:"spit_type"`
	DateCreated    string `json:"date_created"`
	DateExpiration string `json:"date_expiration"`
	IsURL          bool   `json:"is_url"`
	AbsoluteURL    string `json:"absolute_url"`

	Message string `json:"message"`
}
type APIViewResult struct {
	Id             string `json:"id"`
	Content        string `json:"content"`
	SpitType       string `json:"spit_type"`
	DateCreated    string `json:"date_created"`
	DateExpiration string `json:"date_expiration"`
	IsURL          bool   `json:"is_url"`
	AbsoluteURL    string `json:"absolute_url"`
	Clicks         uint64 `json:"clicks"`

	Message string `json:"message"`
}

func requireSpitID(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// test for general format
		id := r.URL.Query().Get(":id")
		if !spit.ValidateSpitId(id) {
			http.Error(w, "Invalid Spit id.", http.StatusBadRequest)
			return
		}
		fn(w, r, id)
	}
}

func apiAddHandler(w http.ResponseWriter, r *http.Request) {
	if strings.ToLower(r.Method) != "post" {
		http.Error(w, "Not supported method", http.StatusMethodNotAllowed)
		return
	}

	s, err := CoreAddMultiSpit(r)

	if err != nil {
		if validationRes, ok := err.(*ErrCoreAdd); ok {
			// it was an error during request validation
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
		} else if errDB, ok := err.(*ErrCoreAddDB); ok {
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
		Id: s.Id, Content: s.Content, SpitType: s.SpitType,
		DateCreated: s.DateCreated, DateExpiration: s.DateExpiration, IsURL: spit.IsUrl(s),
		AbsoluteURL: spit.AbsoluteUrl(s), Message: "Successfully added new Spit!",
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

func apiViewHandler(w http.ResponseWriter, r *http.Request, id string) {
	// fetch the Spit with the requested id
	s, err := spit.Load(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// we are good to go - spit fetched successfully
	result := &APIViewResult{
		Id: s.IdHashOnly(), Content: s.Content, SpitType: s.SpitType,
		DateCreated: s.DateCreated, DateExpiration: s.DateExpiration, IsURL: spit.IsUrl(s),
		AbsoluteURL: spit.AbsoluteUrl(s), Clicks: s.MetricClicks,
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

// webRedirectHandler() tries to find the Spit with the passed ID and either redirects to it
// if it is a URL or it goes to the Spit viewer
// NOT IN USE NOW
func webRedirectHandler(w http.ResponseWriter, r *http.Request, id string) {
	// make sure there is a valid Spit ID
	if !spit.ValidateSpitId(id) {
		http.Error(w, "Invalid Spit id.", http.StatusBadRequest)
		return
	}

	WEB_APP_URL := "/_/"

	// No id provided redirect to the app - old version with Material
	if len(id) == 0 {
		http.Redirect(w, r, WEB_APP_URL, http.StatusMovedPermanently)
		return
	}

	// fetch the Spit with the requested id
	s, err := spit.Load(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// check if this Spit is a URL that we should redirect to
	if spit.IsUrl(s) {
		// HTTP 1.1.
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		// HTTP 1.0.
		w.Header().Set("Pragma", "no-cache")
		// Proxies
		w.Header().Set("Expires", "0")
		http.Redirect(w, r, s.Content, http.StatusMovedPermanently)
		return
	}
	// this is a text Spit so display it inside the app
	http.Redirect(w, r, WEB_APP_URL+"#/view/"+id, http.StatusFound)
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
	fmt.Fprintln(w, "Spito Api V2.0 path: ", r.URL.Path)
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
	port := os.Getenv("PORT")
	if port == "" {
		port = "40090"
	}
	log.Println("Starting Spito at: ", port)

	// Use output file
	/*
		f, _ := os.Create("/var/log/spitoapi/spitoapi-web-server.log")
		defer f.Close()
		log.SetOutput(f)
	*/

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

	/////////////////
	// VIEW ROUTERS
	/////////////////
	//router.Get("/", rootHandler)
	router.Get("/{id}", CORSEnable(requireSpitID(webRedirectHandler)))
	router.Get("/", CORSEnable(requireSpitID(webRedirectHandler)))
	http.Handle("/", router)

	/**
	 *	SINGLE-DOUBLE LETTER DOMAINS ARE RESERVED FOR INTERNAL USAGE
	 */
	// downloads handler - /d/
	fs_d := http.FileServer(http.Dir("downloads"))
	http.Handle("/d/", http.StripPrefix("/d/", fs_d))
	// static files handler - /s/
	fs_s := http.FileServer(http.Dir("spitoweb/s"))
	http.Handle("/s/", http.StripPrefix("/s/", fs_s))
	fs_app := http.FileServer(http.Dir("spitoweb"))
	http.Handle("/_/", http.StripPrefix("/_/", fs_app))
	//router.ServeFiles("/static/*filepath", http.Dir("static"))

	http.ListenAndServe(":"+port, nil)
	//http.ListenAndServe(":40090", router)
}

//////////////////////// HELPERS ////////////////////
