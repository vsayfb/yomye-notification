package config

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"cloud.google.com/go/parametermanager/apiv1/parametermanagerpb"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/googleapis/gax-go/v2"
)

const testFirebaseCredentials = `{
  "type":"service_account",
  "project_id":"test-project",
  "private_key":"private-key",
  "client_email":"firebase@example.iam.gserviceaccount.com"
}`

func TestGCPEnvironmentDefaultsLocationToGlobal(t *testing.T) {
	t.Setenv(EnvGoogleCloudProject, " test-project ")
	t.Setenv(EnvGCPParameterLocation, "")

	projectID, location, err := gcpEnvironment()
	if err != nil {
		t.Fatalf("gcpEnvironment() error = %v", err)
	}
	if projectID != "test-project" || location != DefaultGCPParameterLocation {
		t.Fatalf("gcpEnvironment() = %q, %q", projectID, location)
	}
}

func TestLoadGCPWithFactoriesSuccessAndClosesClients(t *testing.T) {
	parameterClient := newFakeGCPParameterClient()
	secretClient := &fakeGCPSecretClient{
		payload: []byte(`{
          "dbname":"notifications",
          "engine":"postgres",
          "host":"database.internal",
          "password":"database-password",
          "port":5432,
          "username":"notification_user"
        }`),
	}

	cfg, err := loadGCPWithFactories(
		context.Background(),
		"test-project",
		"europe-west3",
		func(context.Context) (gcpParameterClient, error) { return parameterClient, nil },
		func(context.Context) (gcpSecretClient, error) { return secretClient, nil },
	)
	if err != nil {
		t.Fatalf("loadGCPWithFactories() error = %v", err)
	}
	if cfg.APP.Env != EnvironmentProduction {
		t.Errorf("APP.Env = %q, want production", cfg.APP.Env)
	}
	if cfg.DB.Host != "parameter-db.internal" || cfg.DB.Port != "5432" || cfg.DB.Name != "notifications" {
		t.Errorf("DB host/port/name were not loaded from Parameter Manager")
	}
	if cfg.DB.User != "notification_user" || cfg.DB.Password != "database-password" {
		t.Error("DB credentials were not loaded from Secret Manager")
	}
	if cfg.DB.SSLMode != "require" {
		t.Errorf("DB SSLMode = %q, want require", cfg.DB.SSLMode)
	}
	if cfg.APP.FirebaseCredentials == nil || cfg.APP.FirebaseCredentials.ProjectID != "test-project" {
		t.Error("Firebase credentials were not loaded from Parameter Manager")
	}
	if !parameterClient.closed || !secretClient.closed {
		t.Errorf("clients closed: parameter=%v secret=%v, want both true", parameterClient.closed, secretClient.closed)
	}

	wantPrefix := "projects/test-project/locations/europe-west3/parameters/"
	for _, name := range gcpParameterNames {
		want := wantPrefix + name + "/versions/latest"
		if !parameterClient.requested[want] {
			t.Errorf("parameter resource %q was not requested", want)
		}
	}
	if secretClient.requested != "projects/test-project/secrets/postgres/versions/latest" {
		t.Errorf("secret resource = %q", secretClient.requested)
	}
}

func TestLoadGCPParametersMissingParameter(t *testing.T) {
	client := newFakeGCPParameterClient()
	missingResource := "projects/test-project/locations/global/parameters/" + ParameterDBName + "/versions/latest"
	delete(client.values, missingResource)

	_, err := loadGCPParameters(
		context.Background(),
		client,
		"test-project",
		"global",
		gcpParameterNames,
	)
	if err == nil || !strings.Contains(err.Error(), ParameterDBName) {
		t.Fatalf("loadGCPParameters() error = %v, want missing parameter name", err)
	}
}

func TestLoadGCPConfigRejectsInvalidDBPort(t *testing.T) {
	client := newFakeGCPParameterClient()
	resource := "projects/test-project/locations/global/parameters/" + ParameterDBPort + "/versions/latest"
	client.values[resource] = []byte("not-a-port")

	_, err := loadGCPConfig(
		context.Background(),
		client,
		&fakeGCPSecretClient{},
		"test-project",
		"global",
	)
	if err == nil || !strings.Contains(err.Error(), ParameterDBPort) {
		t.Fatalf("loadGCPConfig() error = %v, want db-port error", err)
	}
}

func TestLoadGCPPostgresSecretRejectsMalformedJSONWithoutPayloadLeak(t *testing.T) {
	const secretPayload = `{"password":"do-not-leak"`
	client := &fakeGCPSecretClient{payload: []byte(secretPayload)}
	var dst gcpPostgresSecret

	err := loadGCPPostgresSecret(
		context.Background(),
		client,
		"projects/test-project/secrets/postgres",
		&dst,
	)
	if err == nil || !strings.Contains(err.Error(), "invalid JSON") {
		t.Fatalf("loadGCPPostgresSecret() error = %v, want invalid JSON", err)
	}
	if strings.Contains(err.Error(), "do-not-leak") || strings.Contains(err.Error(), secretPayload) {
		t.Fatalf("error leaked secret payload: %v", err)
	}
}

func TestLoadGCPPostgresSecretAccessFailure(t *testing.T) {
	client := &fakeGCPSecretClient{err: errors.New("permission denied")}
	var dst gcpPostgresSecret

	err := loadGCPPostgresSecret(
		context.Background(),
		client,
		"projects/test-project/secrets/postgres",
		&dst,
	)
	if err == nil || !strings.Contains(err.Error(), ParameterRDSSecretArn) || !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("loadGCPPostgresSecret() error = %v, want contextual access error", err)
	}
}

func TestLoadGCPWithFactoriesClosesParameterClientWhenSecretClientCreationFails(t *testing.T) {
	parameterClient := newFakeGCPParameterClient()

	_, err := loadGCPWithFactories(
		context.Background(),
		"test-project",
		"global",
		func(context.Context) (gcpParameterClient, error) { return parameterClient, nil },
		func(context.Context) (gcpSecretClient, error) { return nil, errors.New("client creation failed") },
	)
	if err == nil {
		t.Fatal("loadGCPWithFactories() error = nil, want client creation error")
	}
	if !parameterClient.closed {
		t.Error("Parameter Manager client was not closed")
	}
}

type fakeGCPParameterClient struct {
	values    map[string][]byte
	errs      map[string]error
	requested map[string]bool
	closed    bool
}

func newFakeGCPParameterClient() *fakeGCPParameterClient {
	values := map[string][]byte{}
	for name, value := range map[string]string{
		ParameterDBHost:              "parameter-db.internal",
		ParameterDBPort:              "5432",
		ParameterDBName:              "notifications",
		ParameterFirebaseCredentials: testFirebaseCredentials,
		ParameterRDSSecretArn:        "projects/test-project/secrets/postgres",
	} {
		resource := fmt.Sprintf(
			"projects/test-project/locations/global/parameters/%s/versions/latest",
			name,
		)
		values[resource] = []byte(value)
		regionalResource := fmt.Sprintf(
			"projects/test-project/locations/europe-west3/parameters/%s/versions/latest",
			name,
		)
		values[regionalResource] = []byte(value)
	}
	return &fakeGCPParameterClient{
		values:    values,
		errs:      map[string]error{},
		requested: map[string]bool{},
	}
}

func (c *fakeGCPParameterClient) RenderParameterVersion(
	_ context.Context,
	request *parametermanagerpb.RenderParameterVersionRequest,
	_ ...gax.CallOption,
) (*parametermanagerpb.RenderParameterVersionResponse, error) {
	c.requested[request.GetName()] = true
	if err := c.errs[request.GetName()]; err != nil {
		return nil, err
	}
	value, ok := c.values[request.GetName()]
	if !ok {
		return &parametermanagerpb.RenderParameterVersionResponse{}, nil
	}
	return &parametermanagerpb.RenderParameterVersionResponse{RenderedPayload: value}, nil
}

func (c *fakeGCPParameterClient) Close() error {
	c.closed = true
	return nil
}

type fakeGCPSecretClient struct {
	payload   []byte
	err       error
	requested string
	closed    bool
}

func (c *fakeGCPSecretClient) AccessSecretVersion(
	_ context.Context,
	request *secretmanagerpb.AccessSecretVersionRequest,
	_ ...gax.CallOption,
) (*secretmanagerpb.AccessSecretVersionResponse, error) {
	c.requested = request.GetName()
	if c.err != nil {
		return nil, c.err
	}
	return &secretmanagerpb.AccessSecretVersionResponse{
		Payload: &secretmanagerpb.SecretPayload{Data: c.payload},
	}, nil
}

func (c *fakeGCPSecretClient) Close() error {
	c.closed = true
	return nil
}
