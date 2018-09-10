package detective

import (
	"encoding/json"
	"net/http"
	"sync"
)

// A Detective instance manages registered dependencies and endpoints.
// Dependencies can be registered with an instance.
// Each instance has a state which represents the health of its components.
type Detective struct {
	name         string
	client       Doer
	dependencies []*Dependency
	endpoints    []*Endpoint
}

// Create a new Detective instance. To avoid confusion, the name provided should preferably be unique among dependent detective instances.
func New(name string) *Detective {
	return &Detective{
		name:   name,
		client: &http.Client{},
	}
}

// Sets the HTTP Client to be used while hitting the endpoint of another detective HTTP ping handler.
func (d *Detective) WithHTTPClient(c Doer) *Detective {
	d.client = c
	return d
}

// Adds a new dependency to the Detective instance. The name provided should preferably be unique among dependencies registered within the same detective instance.
func (d *Detective) Dependency(name string) *Dependency {
	dependency := NewDependency(name)
	d.dependencies = append(d.dependencies, dependency)
	return dependency
}

func (d *Detective) Endpoint(url string) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	d.EndpointReq(req)
	return nil
}

func (d *Detective) EndpointReq(req *http.Request) {
	e := &Endpoint{
		client: d.client,
		req:    *req,
	}
	d.endpoints = append(d.endpoints, e)
}

func (d *Detective) getState() State {
	totalDependencyLength := len(d.dependencies) + len(d.endpoints)
	subStates := make([]State, 0, totalDependencyLength)
	var wg sync.WaitGroup
	wg.Add(totalDependencyLength)
	for _, dep := range d.dependencies {
		go func() {
			s := dep.getState()
			subStates = append(subStates, s)
			wg.Done()
		}()
	}
	for _, e := range d.endpoints {
		go func() {
			s := e.getState()
			subStates = append(subStates, s)
			wg.Done()
		}()
	}
	wg.Wait()
	s := State{Name: d.name}
	return s.WithDependencies(subStates)
}

func (d *Detective) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s := d.getState()
		sBody, err := json.Marshal(s)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write(sBody)
		return
	}
}
