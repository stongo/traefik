package provider

import (
	log "github.com/Sirupsen/logrus"
	"github.com/containous/traefik/provider/k8s"
	"github.com/containous/traefik/safe"
	"github.com/containous/traefik/types"
	"net/http"
	"time"
)

// Kubernetes holds configurations of the Kubernetes provider.
type Kubernetes struct {
	BaseProvider `mapstructure:",squash"`
	Endpoint     string
	Domain       string
}

// Provide allows the provider to provide configurations to traefik
// using the given configuration channel.
func (provider *Kubernetes) Provide(configurationChan chan<- types.ConfigMessage, pool *safe.Pool) error {
	k8sClient, err := k8s.NewClient(provider.Endpoint, &http.Client{})
	if err != nil {
		return err
	}

	go func() {
		for {
			ingresses, err := k8sClient.GetIngresses(func(ingress k8s.Ingress) bool {
				return true
			})
			if err != nil {
				log.Printf("Error retrieving ingresses: %v", err)
				continue
			}
			templateObjects := types.Configuration{
				map[string]*types.Backend{},
				map[string]*types.Frontend{},
			}
			for _, i := range ingresses {
				for _, r := range i.Spec.Rules {
					for _, pa := range r.HTTP.Paths {
						if _, exists := templateObjects.Backends[r.Host+pa.Path]; !exists {
							templateObjects.Backends[r.Host+pa.Path] = &types.Backend{
								Servers: make(map[string]types.Server),
							}
						}
						if _, exists := templateObjects.Frontends[r.Host+pa.Path]; !exists {
							templateObjects.Frontends[r.Host+pa.Path] = &types.Frontend{
								Backend: r.Host + pa.Path,
								Routes:  make(map[string]types.Route),
							}
						}
						if _, exists := templateObjects.Frontends[r.Host+pa.Path].Routes[r.Host]; !exists {
							templateObjects.Frontends[r.Host+pa.Path].Routes[r.Host] = types.Route{
								Rule: "Host:" + r.Host,
							}
						}
						if len(pa.Path) > 0 {
							templateObjects.Frontends[r.Host+pa.Path].Routes[pa.Path] = types.Route{
								Rule: "Path:" + pa.Path,
							}
						}
						services, err := k8sClient.GetServices(func(service k8s.Service) bool {
							return service.Name == pa.Backend.ServiceName
						})
						if err != nil {
							log.Printf("Error retrieving services: %v", err)
							continue
						}
						for _, service := range services {
							templateObjects.Backends[r.Host+pa.Path].Servers[string(service.UID)] = types.Server{
								URL:    "http://" + service.Spec.ClusterIP + ":" + pa.Backend.ServicePort.String(),
								Weight: 1,
							}
						}
					}
				}
			}
			configurationChan <- types.ConfigMessage{
				ProviderName:  "kubernetes",
				Configuration: &templateObjects,
			}
			time.Sleep(30 * time.Second)
		}
	}()
	return nil
}
