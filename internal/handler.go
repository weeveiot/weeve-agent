package internal

import (
	"context"
	"encoding/json"
	"fmt"

	// "log"
	"net/http"

	log "github.com/sirupsen/logrus"

	jwt "github.com/dgrijalva/jwt-go"

	"strings"

	_ "gitlab.com/weeve/edge-server/edge-pipeline-service/docs"

	httpSwagger "github.com/swaggo/http-swagger"

	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/controller"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/model"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/util"

	"github.com/gorilla/mux"
)

func HandleRequests(portNum int) {
	router := mux.NewRouter().StrictSlash(true)
	router.Use(CommonMiddleware)
	// jwtMiddleware := jwtmiddleware.New(jwtmiddleware.Options{

	router.HandleFunc("/", controller.Status)
	router.HandleFunc("/login", controller.Login).Methods("POST")
	router.PathPrefix("/docs/").Handler(httpSwagger.WrapHandler)

	subRouter := router.PathPrefix("/").Subrouter()
	// TODO: This is disabled for now!!
	// subRouter.Use(JwtVerify)

	// Images
	subRouter.HandleFunc("/images/{id}", controller.GETimagesID).Methods("GET")
	subRouter.HandleFunc("/images", controller.GETimages).Methods("GET")
	subRouter.HandleFunc("/images", controller.POSTimage).Methods("POST")
	subRouter.HandleFunc("/images/{id}", controller.PUTimagesID).Methods("PUT")
	subRouter.HandleFunc("/images/{id}", controller.DELETEimagesID).Methods("DELETE")

	// Containers
	subRouter.HandleFunc("/containers/start", controller.POSTcontainersStart).Methods("POST")
	subRouter.HandleFunc("/containers/start/{id}", controller.POSTcontainersStartID).Methods("POST")
	subRouter.HandleFunc("/containers/stop", controller.POSTcontainersStop).Methods("POST")
	subRouter.HandleFunc("/containers/stop/{id}", controller.POSTcontainersStopID).Methods("POST")
	subRouter.HandleFunc("/containers/deploy", controller.POSTcontainersDeploy).Methods("POST")
	// subRouter.HandleFunc("/containers/create/{containerName}/{imageName}", controller.CreateStartContainer).Methods("POST")
	subRouter.HandleFunc("/containers/{id}", controller.DELETEcontainersID).Methods("DELETE")
	subRouter.HandleFunc("/containers", controller.GETcontainers).Methods("GET")
	subRouter.HandleFunc("/containers/{id}", controller.GETcontainersID).Methods("GET")
	subRouter.HandleFunc("/containers/{id}/logs", controller.GETcontainersIDlogs).Methods("GET")

	// Pipelines
	subRouter.HandleFunc("/pipelines", controller.POSTpipelines).Methods("POST")

	util.PrintEndpoints(router)

	// This is the main server loop!
	log.Debug("Running ListenAndServe")
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", portNum), router))
}

func CommonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// w.Header().Add("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Access-Control-Request-Headers, Access-Control-Request-Method, Connection, Host, Origin, User-Agent, Referer, Cache-Control, X-header")
		next.ServeHTTP(w, r)
	})
}

func JwtVerify(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		fmt.Printf("%+v\n", r.Header)
		var header = r.Header.Get("Authorization") //Grab the token from the header

		header = strings.TrimSpace(header)

		if header == "" {
			//Token is missing, returns with error code 403 Unauthorized
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(model.Exception{Message: "Missing auth token"})
			return
		}
		tk := &model.Token{}

		_, err := jwt.ParseWithClaims(header, tk, func(token *jwt.Token) (interface{}, error) {
			return []byte("secret"), nil
		})

		if err != nil {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(model.Exception{Message: err.Error()})
			return
		}

		ctx := context.WithValue(r.Context(), "user", tk)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
