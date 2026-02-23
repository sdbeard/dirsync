// *********************************************************************************
// Copyright © 2026 Sean Beard - All Rights Reserved
// Unauthorized copying of this file, via any medium is strictly prohibited
// Proprietary and Confidential
// *********************************************************************************
package api

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/mux"
	"github.com/justinas/alice"
	"github.com/sdbeard/dirsync/internal/types"
	apicfg "github.com/sdbeard/go-supportlib/api/config"
	"github.com/sdbeard/go-supportlib/api/handlers"
	apisvc "github.com/sdbeard/go-supportlib/api/service"
	"github.com/unrolled/render"
)

// NewSynchronizerStatusAPI creates and configures a new SynchronizerStatusAPI
// object and returns a pointer to the created object
func NewSynchronizerStatusAPI(config apicfg.ListenerConfig) *SynchronizerStatusAPI {
	newAPI := &SynchronizerStatusAPI{
		configuration: config,
		render:        render.New(),
	}

	newAPI.restService = apisvc.NewRestService(
		config,
		newAPI.initializeRouter,
	)

	return newAPI
}

/***** SynchronizerStatusAPI ******************************************************/

// SynchronizerStatusAPI defines the API for retrieving the current status for the
// execution of the directory synchronization service
type SynchronizerStatusAPI struct {
	restService   *apisvc.RestService
	configuration apicfg.ListenerConfig
	render        *render.Render
}

/***** exported functions *********************************************************/

// Start starts the running version of the API and is ready to receive requests
func (api *SynchronizerStatusAPI) Start() error {
	return api.restService.StartSimple()
}

// Stop initiaties the graceful shutdown of the API's underlying rest service
func (api *SynchronizerStatusAPI) Stop() {
	api.restService.Stop()
}

/**********************************************************************************/

// initializeRouter statisfies the requirment for defing the routing and API
// endpoints that the 'restService' uses
func (api *SynchronizerStatusAPI) initializeRouter(router chi.Router) {
	stdChain := alice.New(handlers.LoggingHandler, handlers.JSONContentTypeHandler)

	router.Route("/", func(apiRouter chi.Router) {
		apiRouter.Use(stdChain.Then)
		apiRouter.Get("/status", api.getStatus)
		apiRouter.Get("/status/{profile}", api.getProfileStatus)
	})
}

// getStatus manages the retrieval of the status object and sending in the response
// as a JSON document
func (api *SynchronizerStatusAPI) getStatus(res http.ResponseWriter, req *http.Request) {
	if strings.Contains(req.RemoteAddr, "localhost") && strings.Contains("", "localhost") {
		//Allow CORS here By * or specific origin
		res.Header().Set("Access-Control-Allow-Origin", "*")
		res.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	}

	api.render.JSON(res, 200, types.GetServiceStatus())
}

func (api *SynchronizerStatusAPI) getProfileStatus(res http.ResponseWriter, req *http.Request) {
	if strings.Contains(req.RemoteAddr, "localhost") && strings.Contains("", "localhost") {
		//Allow CORS here By * or specific origin
		res.Header().Set("Access-Control-Allow-Origin", "*")
		res.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	}

	profile := mux.Vars(req)["profile"]
	profileStatus := types.GetServiceStatus().GetSynchronizerStatus(profile)

	api.render.JSON(res, 200, profileStatus)
}

/**********************************************************************************/
