package hostauth_test

import (
	"context"
	"strings"
	"testing"

	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/app"
	"github.com/go-go-golems/go-go-goja/pkg/xgoja/hostauth"
)

type fakeFactory struct{}

func (*fakeFactory) BuildHostAuthServices(context.Context, *values.Values) (*hostauth.Services, error) {
	return &hostauth.Services{}, nil
}

func TestLookupServiceFactory(t *testing.T) {
	host := app.HostServices{}
	factory := &fakeFactory{}
	if err := host.SetHostService(hostauth.ServiceFactoryKey, factory); err != nil {
		t.Fatalf("SetHostService: %v", err)
	}
	got, ok, err := hostauth.LookupServiceFactory(host)
	if err != nil {
		t.Fatalf("LookupServiceFactory: %v", err)
	}
	if !ok || got != factory {
		t.Fatalf("factory = %#v ok=%v", got, ok)
	}
}

func TestLookupServiceFactoryMissing(t *testing.T) {
	got, ok, err := hostauth.LookupServiceFactory(app.HostServices{})
	if err != nil || ok || got != nil {
		t.Fatalf("factory = %#v ok=%v err=%v", got, ok, err)
	}
}

func TestLookupServiceFactoryRejectsWrongType(t *testing.T) {
	host := app.HostServices{}
	if err := host.SetHostService(hostauth.ServiceFactoryKey, "bad"); err != nil {
		t.Fatalf("SetHostService: %v", err)
	}
	_, _, err := hostauth.LookupServiceFactory(host)
	if err == nil || !strings.Contains(err.Error(), "must implement hostauth.ServiceFactory") {
		t.Fatalf("error = %v", err)
	}
}

func TestLookupServiceFactoryRejectsNilTypedPointer(t *testing.T) {
	host := app.HostServices{}
	var factory *fakeFactory
	if err := host.SetHostService(hostauth.ServiceFactoryKey, factory); err != nil {
		t.Fatalf("SetHostService: %v", err)
	}
	_, _, err := hostauth.LookupServiceFactory(host)
	if err == nil || !strings.Contains(err.Error(), "is nil") {
		t.Fatalf("error = %v", err)
	}
}

func TestLookupServiceFactoryRejectsMultiValueService(t *testing.T) {
	host := app.HostServices{}
	if err := host.AddHostService(hostauth.ServiceFactoryKey, &fakeFactory{}); err != nil {
		t.Fatalf("AddHostService first: %v", err)
	}
	if err := host.AddHostService(hostauth.ServiceFactoryKey, &fakeFactory{}); err != nil {
		t.Fatalf("AddHostService second: %v", err)
	}
	_, _, err := hostauth.LookupServiceFactory(host)
	if err == nil || !strings.Contains(err.Error(), "must implement hostauth.ServiceFactory") || !strings.Contains(err.Error(), "[]interface") {
		t.Fatalf("error = %v", err)
	}
}

func TestLookupServices(t *testing.T) {
	host := app.HostServices{}
	services := &hostauth.Services{}
	if err := host.SetHostService(hostauth.ServicesKey, services); err != nil {
		t.Fatalf("SetHostService: %v", err)
	}
	got, ok, err := hostauth.LookupServices(host)
	if err != nil {
		t.Fatalf("LookupServices: %v", err)
	}
	if !ok || got != services {
		t.Fatalf("services = %#v ok=%v", got, ok)
	}
}

func TestLookupServicesMissing(t *testing.T) {
	got, ok, err := hostauth.LookupServices(app.HostServices{})
	if err != nil || ok || got != nil {
		t.Fatalf("services = %#v ok=%v err=%v", got, ok, err)
	}
}

func TestLookupServicesRejectsWrongType(t *testing.T) {
	host := app.HostServices{}
	if err := host.SetHostService(hostauth.ServicesKey, "bad"); err != nil {
		t.Fatalf("SetHostService: %v", err)
	}
	_, _, err := hostauth.LookupServices(host)
	if err == nil || !strings.Contains(err.Error(), "must be *hostauth.Services") {
		t.Fatalf("error = %v", err)
	}
}
