package main

import (
	"encoding/json"
	"fmt"
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
	"templates/partials/header.html",
	"templates/partials/footer.html",
	"templates/view.html",
	"templates/add.html"))

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

type APIDeleteResult struct {
	Message string `json: "message"`
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
	//func httpRouterNoParams(func(http.ResponseWriter, *http.Request) fn) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		h.ServeHTTP(w, r)
	}
}
func httpRouterNoParamsFn(fn func(http.ResponseWriter, *http.Request)) httprouter.Handle {
	//func httpRouterNoParams(func(http.ResponseWriter, *http.Request) fn) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		fn(w, r)
	}
}

func webDeleteHandler(w http.ResponseWriter, r *http.Request, id string) {
	s, err := spit.Load(id)
	if err != nil {
		http.Error(w, "Could not find the Spit specified!", http.StatusBadRequest)
		return
	}
	if err = s.Del(); err != nil {
		http.Error(w, "Could not delete the spit specified!", http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
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

func webAddHandler(w http.ResponseWriter, r *http.Request) {
	if strings.ToLower(r.Method) == "post" {
		s, err := CoreAddMultiSpit(r)

		if err != nil {
			if _, ok := err.(*ErrCoreAdd); ok {
				// it was an error during request validation
				validationRes := err.(*ErrCoreAdd)
				renderTemplate(w, "add", validationRes)
				return
			} else if _, ok := err.(*ErrCoreAddDB); ok {
				errDB := err.(*ErrCoreAddDB)
				log.Printf("viewAddHandler::Internal error: %v", errDB)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			} else {
				// other internal error
				http.Error(w, err.Error(), http.StatusInternalServerError)
				log.Fatalf("viewAddHandler::Unknown Error: %v", err)
				return

			}
		}

		// spit created successfully
		http.Redirect(w, r, "/v/"+s.Id(), http.StatusFound)
		return
	} // end of POST
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
	result := &APIAddResult{
		Id: s.Id(), Exp: s.Exp(), Content: s.Content(), SpitType: s.SpitType(),
		DateCreated: s.DateCreated(), FormattedCreatedTime: s.FormattedCreatedTime(),
		IsURL: s.IsURL(), AbsoluteURL: s.AbsoluteURL(), Message: "Successfully fetched Spit!",
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

func webViewHandler(w http.ResponseWriter, r *http.Request, id string) {
	// fetch the Spit with the requested id
	s, err := spit.Load(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// display the Spit
	bundle := &struct {
		Spit   *spit.ISpit
		Footer *struct{ Year int }
		Header *struct{ Title string }
	}{
		Spit:   &s,
		Footer: &struct{ Year int }{Year: time.Now().Year()},
		Header: &struct{ Title string }{Title: s.Id()},
	}
	renderTemplate(w, "view", bundle)
}

func limitSizeHandler(fn func(http.ResponseWriter, *http.Request),
	size int64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, size)
		fn(w, r)
	}
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
	webRedirectHandler(w, r, id)
}

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
	http.Redirect(w, r, "/v/"+id, http.StatusFound)
	return
}

func main() {

	fmt.Println("Starting Spito at: 40090")

	// use all the available cores
	runtime.GOMAXPROCS(runtime.NumCPU())

	router := httprouter.New()

	router.GET("/", httpRouterNoParams(limitSizeHandler(rootHandler, MAX_FORM_SIZE)))
	//router.GET("/:id", requireSpitIDHttpRouter(webRedirectHandler))

	/////////////////
	// API ROUTERS
	/////////////////

	router.POST("/api/v1/spits", httpRouterNoParams(limitSizeHandler(apiAddHandler, MAX_FORM_SIZE)))
	router.GET("/api/v1/spits/:id", requireSpitIDHttpRouter(apiViewHandler))

	router.DELETE("/api/v1/spits/:id", requireSpitIDHttpRouter(apiDeleteHandler))
	router.POST("/api/v1-web/spits/:id/delete", requireSpitIDHttpRouter(webDeleteHandler))

	/////////////////
	// VIEW ROUTERS
	/////////////////

	router.GET("/v/:id", requireSpitIDHttpRouter(webViewHandler))

	router.POST("/", httpRouterNoParams(limitSizeHandler(webAddHandler, MAX_FORM_SIZE)))

	router.ServeFiles("/static/*filepath", http.Dir("static"))

	//http.ListenAndServe(":40090", nil)
	http.ListenAndServe(":40090", router)
}

//////////////////////// HELPERS ////////////////////
