package build

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	client "github.com/rancher/go-rancher/v2"
)

func DeployToRancher(image string) error {

	var (
		ok      bool
		service string
	)
	if service, ok = os.LookupEnv("RANCHER_SERVICE"); !ok {
		return errors.New("RANCHER_SERVICE must be set")
	}

	return DeployToRancherService(image, service)
}

func DeployToRancherService(image string, service string) error {
	var (
		ok      bool
		timeout string
		err     error
	)

	plugin := plugin{
		DockerImage: image,
		Confirm:     true,
		Service:     service,
		Timeout:     5,
	}

	if plugin.URL, ok = os.LookupEnv("RANCHER_URL"); !ok {
		return errors.New("RANCHER_URL must be set")
	}
	if plugin.Key, ok = os.LookupEnv("RANCHER_KEY"); !ok {
		return errors.New("RANCHER_KEY must be set")
	}
	if plugin.Secret, ok = os.LookupEnv("RANCHER_SECRET"); !ok {
		return errors.New("RANCHER_SECRET must be set")
	}

	if timeout, ok = os.LookupEnv("RANCHER_TIMEOUT"); ok {
		if plugin.Timeout, err = strconv.Atoi(timeout); err != nil {
			return fmt.Errorf("invalid RANCHER_TIMEOUT '%s': %s", timeout, err)
		}
	}

	err = plugin.exec()

	return err
}

type plugin struct {
	URL            string
	Key            string
	Secret         string
	Service        string
	DockerImage    string
	StartFirst     bool
	Confirm        bool
	Timeout        int
	IntervalMillis int64
	BatchSize      int64
	YamlVerified   bool
}

func (p *plugin) exec() error {
	var wantedService, wantedStack string
	if strings.Contains(p.Service, "/") {
		parts := strings.SplitN(p.Service, "/", 2)
		wantedStack = parts[0]
		wantedService = parts[1]
	} else {
		wantedService = p.Service
	}

	if !strings.HasPrefix(p.DockerImage, "docker:") {
		p.DockerImage = fmt.Sprintf("docker:%s", p.DockerImage)
	}

	rancher, err := client.NewRancherClient(&client.ClientOpts{
		Url:       p.URL,
		AccessKey: p.Key,
		SecretKey: p.Secret,
	})
	if err != nil {
		return fmt.Errorf("Failed to create rancher client: %s", err)
	}

	// Prepare service filters for service listing
	serviceFilters := map[string]interface{}{"name": wantedService}

	// Query stacks with filter name=wantedStack
	if wantedStack != "" {
		stacks, err := rancher.Stack.List(&client.ListOpts{Filters: map[string]interface{}{"name": wantedStack}})
		if err != nil {
			return fmt.Errorf("Failed to list rancher environments: %s", err)
		}
		if len(stacks.Data) <= 0 {
			return fmt.Errorf("Unable to find stack %s", wantedStack)
		}
		// If found add stackID to serviceFilters
		serviceFilters["stackId"] = stacks.Data[0].Id
	}

	// Query services with prepared filters
	services, err := rancher.Service.List(&client.ListOpts{Filters: serviceFilters})
	if err != nil {
		return fmt.Errorf("Failed to list rancher services: %s", err)
	}
	if len(services.Data) <= 0 {
		return fmt.Errorf("Unable to find service %s", p.Service)
	}
	service := services.Data[0]

	CIProgress("Upgrading Rancher: " + p.Service)

	// Service is found, proceed with upgrade
	service.LaunchConfig.ImageUuid = p.DockerImage
	upgrade := &client.ServiceUpgrade{}
	upgrade.InServiceStrategy = &client.InServiceUpgradeStrategy{
		LaunchConfig:           service.LaunchConfig,
		SecondaryLaunchConfigs: service.SecondaryLaunchConfigs,
		StartFirst:             p.StartFirst,
		IntervalMillis:         p.IntervalMillis,
		BatchSize:              p.BatchSize,
	}
	upgrade.ToServiceStrategy = &client.ToServiceUpgradeStrategy{}
	_, err = rancher.Service.ActionUpgrade(&service, upgrade)
	if err != nil {
		return fmt.Errorf("Unable to upgrade service %s: %s", p.Service, err)
	}

	CIProgress(fmt.Sprintf("Upgraded %s to %s", p.Service, p.DockerImage))
	if p.Confirm {
		srv, err := retry(func() (interface{}, error) {
			s, e := rancher.Service.ById(service.Id)
			if e != nil {
				return nil, e
			}
			if s.State != "upgraded" {
				return nil, fmt.Errorf("Service not upgraded: %s", s.State)
			}
			return s, nil
		}, time.Duration(p.Timeout)*time.Second, 3*time.Second)

		if err != nil {
			return fmt.Errorf("Error waiting for service upgrade to complete: %s", err)
		}

		_, err = rancher.Service.ActionFinishupgrade(srv.(*client.Service))
		if err != nil {
			return fmt.Errorf("Unable to finish upgrade %s: %s", p.Service, err)
		}
		CIProgress(fmt.Sprintf("Finished upgrade %s", p.Service))
	}
	return nil
}

type retryFunc func() (interface{}, error)

func retry(f retryFunc, timeout time.Duration, interval time.Duration) (interface{}, error) {
	finish := time.After(timeout)
	for {
		result, err := f()
		if err == nil {
			return result, nil
		}
		select {
		case <-finish:
			return nil, err
		case <-time.After(interval):
		}
	}
}
