package http

import (
	"context"
	"testing"

	"github.com/go-go-golems/go-go-goja/pkg/xgoja/providerapi"
)

func TestRegister(t *testing.T) {
	registry := providerapi.NewRegistry()
	if err := Register(registry); err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, ok := registry.ResolveModule(PackageID, "express"); !ok {
		t.Fatal("expected express module")
	}
	caps, ok := registry.ResolveCapabilities(PackageID)
	if !ok || len(caps) != 1 {
		t.Fatalf("capabilities = %#v ok=%v", caps, ok)
	}
}

func TestCapabilityProvidesHTTPSection(t *testing.T) {
	capability := newHTTPCapability()
	sections, err := capability.ConfigSections(providerapi.SectionContext{})
	if err != nil {
		t.Fatalf("sections: %v", err)
	}
	if len(sections) != 1 || sections[0].GetSlug() != "http" {
		t.Fatalf("sections = %#v", sections)
	}
	if sections[0].GetPrefix() != "http-" {
		t.Fatalf("prefix = %q", sections[0].GetPrefix())
	}
}

func TestCapabilityRejectsNilRuntimeHandle(t *testing.T) {
	capability := newHTTPCapability()
	if err := capability.InitRuntimeFromSections(context.Background(), nil, nil); err == nil {
		t.Fatal("expected nil runtime handle error")
	}
}
