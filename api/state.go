package api

import (
	"errors"
	"net/http"
	"path"

	"github.com/coreos/fleet/client"
	"github.com/coreos/fleet/log"
	"github.com/coreos/fleet/schema"
)

func wireUpStateResource(mux *http.ServeMux, prefix string, cAPI client.API) {
	base := path.Join(prefix, "state")
	sr := stateResource{cAPI, base}
	mux.Handle(base, &sr)
}

type stateResource struct {
	cAPI     client.API
	basePath string
}

func (sr *stateResource) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		sendError(rw, http.StatusMethodNotAllowed, errors.New("only GET supported against this resource"))
		return
	}

	sr.list(rw, req)
}

func (sr *stateResource) list(rw http.ResponseWriter, req *http.Request) {
	token, err := findNextPageToken(req.URL)
	if err != nil {
		sendError(rw, http.StatusBadRequest, err)
		return
	}

	if token == nil {
		def := DefaultPageToken()
		token = &def
	}

	var machineID, unitName string
	for _, val := range req.URL.Query()["machineID"] {
		machineID = val
		break
	}
	for _, val := range req.URL.Query()["unitName"] {
		unitName = val
		break
	}

	page, err := getUnitStatePage(sr.cAPI, machineID, unitName, *token)
	if err != nil {
		log.Errorf("Failed fetching page of UnitStates: %v", err)
		sendError(rw, http.StatusInternalServerError, nil)
		return
	}

	sendResponse(rw, http.StatusOK, &page)
}

func getUnitStatePage(cAPI client.API, machineID, unitName string, tok PageToken) (*schema.UnitStatePage, error) {
	states, err := cAPI.UnitStates()
	if err != nil {
		return nil, err
	}
	var filtered []*schema.UnitState
	for _, us := range states {
		if machineID != "" && machineID != us.MachineID {
			continue
		}
		if unitName != "" && unitName != us.Name {
			continue
		}
		filtered = append(filtered, us)
	}

	items, next := extractUnitStatePageData(filtered, tok)
	page := schema.UnitStatePage{
		States: items,
	}

	if next != nil {
		page.NextPageToken = next.Encode()
	}

	return &page, nil
}

func extractUnitStatePageData(all []*schema.UnitState, tok PageToken) (items []*schema.UnitState, next *PageToken) {
	total := len(all)

	startIndex := int((tok.Page - 1) * tok.Limit)
	stopIndex := int(tok.Page * tok.Limit)

	if startIndex < total {
		if stopIndex > total {
			stopIndex = total
		} else {
			n := tok.Next()
			next = &n
		}

		items = all[startIndex:stopIndex]
	}

	return
}
