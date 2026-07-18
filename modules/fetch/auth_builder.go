package fetch

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/dop251/goja"
)

type builderStore struct {
	credentials sync.Map // map[*goja.Object]credentialSource
}

func newBuilderStore() *builderStore { return &builderStore{} }

type credentialSource interface {
	apply(ctx context.Context, req *http.Request) error
	redacted() string
}

type noneCredential struct{}

func (noneCredential) apply(context.Context, *http.Request) error { return nil }
func (noneCredential) redacted() string                           { return "none" }

type bearerCredential struct {
	policy   Policy
	token    string
	envName  string
	filePath string
	jsonPath string
}

func (c *bearerCredential) apply(_ context.Context, req *http.Request) error {
	token, err := c.resolve()
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	return nil
}

func (c *bearerCredential) redacted() string { return "bearer(<redacted>)" }

func (c *bearerCredential) resolve() (string, error) {
	switch {
	case strings.TrimSpace(c.token) != "":
		return strings.TrimSpace(c.token), nil
	case strings.TrimSpace(c.envName) != "":
		if err := c.policy.CheckCredentialEnv(c.envName); err != nil {
			return "", err
		}
		value := strings.TrimSpace(os.Getenv(c.envName))
		if value == "" {
			return "", fmt.Errorf("credential env var %q is empty", c.envName)
		}
		return value, nil
	case strings.TrimSpace(c.filePath) != "":
		if err := c.policy.CheckCredentialFile(c.filePath); err != nil {
			return "", err
		}
		data, err := os.ReadFile(c.filePath)
		if err != nil {
			return "", fmt.Errorf("read credential file %q: %w", c.filePath, err)
		}
		if strings.TrimSpace(c.jsonPath) == "" {
			value := strings.TrimSpace(string(data))
			if value == "" {
				return "", fmt.Errorf("credential file %q is empty", c.filePath)
			}
			return value, nil
		}
		value, err := extractJSONPath(data, c.jsonPath)
		if err != nil {
			return "", err
		}
		return value, nil
	default:
		return "", fmt.Errorf("bearer credential source is not configured")
	}
}

func (s *builderStore) newNoneAuth(vm *goja.Runtime) *goja.Object {
	obj := vm.NewObject()
	s.credentials.Store(obj, noneCredential{})
	return obj
}

func (s *builderStore) newBearerAuth(vm *goja.Runtime, policy Policy) *goja.Object {
	cred := &bearerCredential{policy: policy}
	obj := vm.NewObject()
	s.credentials.Store(obj, cred)
	_ = obj.Set("token", func(value string) *goja.Object {
		cred.token = strings.TrimSpace(value)
		cred.envName = ""
		cred.filePath = ""
		return obj
	})
	_ = obj.Set("fromEnv", func(name string) *goja.Object {
		cred.envName = strings.TrimSpace(name)
		cred.token = ""
		cred.filePath = ""
		return obj
	})
	_ = obj.Set("fromFile", func(path string) *goja.Object {
		cred.filePath = strings.TrimSpace(path)
		cred.token = ""
		cred.envName = ""
		return obj
	})
	_ = obj.Set("jsonPath", func(path string) *goja.Object {
		cred.jsonPath = strings.TrimSpace(path)
		return obj
	})
	_ = obj.Set("toString", func() string { return cred.redacted() })
	return obj
}

func (s *builderStore) credential(vm *goja.Runtime, value goja.Value) (credentialSource, error) {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return nil, fmt.Errorf("client.auth(...) expects value returned by fetch.auth.*")
	}
	raw, ok := s.credentials.Load(value.ToObject(vm))
	if !ok {
		return nil, fmt.Errorf("client.auth(...) expects value returned by fetch.auth.*")
	}
	credential, ok := raw.(credentialSource)
	if !ok || credential == nil {
		return nil, fmt.Errorf("internal fetch auth spec has invalid type")
	}
	return credential, nil
}

func extractJSONPath(data []byte, path string) (string, error) {
	var decoded any
	if err := json.Unmarshal(data, &decoded); err != nil {
		return "", fmt.Errorf("credential file json: %w", err)
	}
	current := decoded
	for _, part := range strings.Split(path, ".") {
		part = strings.TrimSpace(part)
		if part == "" {
			return "", fmt.Errorf("credential json path %q is invalid", path)
		}
		object, ok := current.(map[string]any)
		if !ok {
			return "", fmt.Errorf("credential json path %q does not resolve to a string", path)
		}
		current, ok = object[part]
		if !ok {
			return "", fmt.Errorf("credential json path %q not found", path)
		}
	}
	value, ok := current.(string)
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("credential json path %q does not resolve to a non-empty string", path)
	}
	return strings.TrimSpace(value), nil
}
