package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	parametermanager "cloud.google.com/go/parametermanager/apiv1"
	"cloud.google.com/go/parametermanager/apiv1/parametermanagerpb"
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/googleapis/gax-go/v2"
	"github.com/vsayfb/gig-platform-notification-lambda/pkg/fb"
)

const DefaultGCPParameterLocation = "global"

// GCP configuration deliberately contains no message-transport client.
// This repository's current entrypoint is an AWS Lambda/SQS consumer; a
// production Pub/Sub subscriber belongs in a separate bootstrap/transport
// adapter and must not reinterpret a Pub/Sub topic as an SQS queue URL.

var gcpParameterNames = []string{
	ParameterDBHost,
	ParameterDBPort,
	ParameterDBName,
	ParameterFirebaseCredentials,
	ParameterRDSSecretArn,
}

type gcpParameterClient interface {
	RenderParameterVersion(
		context.Context,
		*parametermanagerpb.RenderParameterVersionRequest,
		...gax.CallOption,
	) (*parametermanagerpb.RenderParameterVersionResponse, error)
	Close() error
}

type gcpSecretClient interface {
	AccessSecretVersion(
		context.Context,
		*secretmanagerpb.AccessSecretVersionRequest,
		...gax.CallOption,
	) (*secretmanagerpb.AccessSecretVersionResponse, error)
	Close() error
}

type gcpParameterClientFactory func(context.Context) (gcpParameterClient, error)
type gcpSecretClientFactory func(context.Context) (gcpSecretClient, error)

type gcpPostgresSecret struct {
	DBName   string `json:"dbname"`
	Engine   string `json:"engine"`
	Host     string `json:"host"`
	Password string `json:"password"`
	Port     int    `json:"port"`
	Username string `json:"username"`
}

func loadGCP(ctx context.Context) (*Config, error) {
	projectID, location, err := gcpEnvironment()
	if err != nil {
		return nil, err
	}

	return loadGCPWithFactories(
		ctx,
		projectID,
		location,
		func(ctx context.Context) (gcpParameterClient, error) {
			return parametermanager.NewClient(ctx)
		},
		func(ctx context.Context) (gcpSecretClient, error) {
			return secretmanager.NewClient(ctx)
		},
	)
}

func gcpEnvironment() (projectID string, location string, err error) {
	projectID = strings.TrimSpace(os.Getenv(EnvGoogleCloudProject))
	if projectID == "" {
		return "", "", fmt.Errorf("%s is required for production configuration", EnvGoogleCloudProject)
	}

	location = strings.TrimSpace(os.Getenv(EnvGCPParameterLocation))
	if location == "" {
		location = DefaultGCPParameterLocation
	}
	return projectID, location, nil
}

func loadGCPWithFactories(
	ctx context.Context,
	projectID string,
	location string,
	newParameterClient gcpParameterClientFactory,
	newSecretClient gcpSecretClientFactory,
) (cfg *Config, err error) {
	parameterClient, err := newParameterClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("create Parameter Manager client: %w", err)
	}
	defer func() {
		if closeErr := parameterClient.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close Parameter Manager client: %w", closeErr))
		}
	}()

	secretClient, err := newSecretClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("create Secret Manager client: %w", err)
	}
	defer func() {
		if closeErr := secretClient.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close Secret Manager client: %w", closeErr))
		}
	}()

	return loadGCPConfig(ctx, parameterClient, secretClient, projectID, location)
}

func loadGCPConfig(
	ctx context.Context,
	parameterClient gcpParameterClient,
	secretClient gcpSecretClient,
	projectID string,
	location string,
) (*Config, error) {
	params, err := loadGCPParameters(ctx, parameterClient, projectID, location, gcpParameterNames)
	if err != nil {
		return nil, err
	}

	dbPort := strings.TrimSpace(params[ParameterDBPort])
	port, err := strconv.Atoi(dbPort)
	if err != nil || port < 1 || port > 65535 {
		return nil, fmt.Errorf("invalid Parameter Manager parameter %q: must be a TCP port", ParameterDBPort)
	}

	var dbSecret gcpPostgresSecret
	if err := loadGCPPostgresSecret(
		ctx,
		secretClient,
		params[ParameterRDSSecretArn],
		&dbSecret,
	); err != nil {
		return nil, err
	}

	var creds fb.FirebaseServiceAccount
	if err := json.Unmarshal([]byte(params[ParameterFirebaseCredentials]), &creds); err != nil {
		return nil, fmt.Errorf(
			"decode Parameter Manager parameter %q: invalid Firebase credentials JSON",
			ParameterFirebaseCredentials,
		)
	}
	if err := creds.Validate(); err != nil {
		return nil, fmt.Errorf(
			"validate Parameter Manager parameter %q: %w",
			ParameterFirebaseCredentials,
			err,
		)
	}

	return &Config{
		DB: DBConfig{
			Host:     strings.TrimSpace(params[ParameterDBHost]),
			Port:     dbPort,
			User:     dbSecret.Username,
			Password: dbSecret.Password,
			Name:     strings.TrimSpace(params[ParameterDBName]),
			SSLMode:  "require",
		},
		APP: AppConfig{
			Env:                 EnvironmentProduction,
			FirebaseCredentials: &creds,
		},
	}, nil
}

func loadGCPParameters(
	ctx context.Context,
	client gcpParameterClient,
	projectID string,
	location string,
	names []string,
) (map[string]string, error) {
	params := make(map[string]string, len(names))
	for _, name := range names {
		resourceName := fmt.Sprintf(
			"projects/%s/locations/%s/parameters/%s/versions/latest",
			projectID,
			location,
			name,
		)
		response, err := client.RenderParameterVersion(
			ctx,
			&parametermanagerpb.RenderParameterVersionRequest{Name: resourceName},
		)
		if err != nil {
			return nil, fmt.Errorf("render Parameter Manager parameter %q: %w", name, err)
		}
		if response == nil || len(response.GetRenderedPayload()) == 0 {
			return nil, fmt.Errorf("Parameter Manager parameter %q is missing or empty", name)
		}
		params[name] = string(response.GetRenderedPayload())
	}
	return params, nil
}

func loadGCPPostgresSecret(
	ctx context.Context,
	client gcpSecretClient,
	resourceName string,
	dst *gcpPostgresSecret,
) error {
	resourceName = strings.TrimSpace(resourceName)
	if !validGCPSecretResourceName(resourceName) {
		return fmt.Errorf(
			"Parameter Manager parameter %q must name a GCP secret resource",
			ParameterRDSSecretArn,
		)
	}

	versionName := resourceName + "/versions/latest"
	response, err := client.AccessSecretVersion(
		ctx,
		&secretmanagerpb.AccessSecretVersionRequest{Name: versionName},
	)
	if err != nil {
		return fmt.Errorf("access PostgreSQL secret referenced by parameter %q: %w", ParameterRDSSecretArn, err)
	}
	if response == nil || response.GetPayload() == nil || len(response.GetPayload().GetData()) == 0 {
		return fmt.Errorf("PostgreSQL secret referenced by parameter %q is empty", ParameterRDSSecretArn)
	}
	if err := json.Unmarshal(response.GetPayload().GetData(), dst); err != nil {
		return fmt.Errorf("decode PostgreSQL secret referenced by parameter %q: invalid JSON", ParameterRDSSecretArn)
	}
	if strings.TrimSpace(dst.Username) == "" || dst.Password == "" {
		return fmt.Errorf("PostgreSQL secret referenced by parameter %q is missing credentials", ParameterRDSSecretArn)
	}
	if strings.TrimSpace(dst.DBName) == "" || strings.TrimSpace(dst.Host) == "" || dst.Port < 1 || dst.Port > 65535 {
		return fmt.Errorf("PostgreSQL secret referenced by parameter %q is missing connection metadata", ParameterRDSSecretArn)
	}
	if dst.Engine != "" && dst.Engine != "postgres" {
		return fmt.Errorf("PostgreSQL secret referenced by parameter %q has an unsupported engine", ParameterRDSSecretArn)
	}
	return nil
}

func validGCPSecretResourceName(name string) bool {
	parts := strings.Split(name, "/")
	return len(parts) == 4 &&
		parts[0] == "projects" && parts[1] != "" &&
		parts[2] == "secrets" && parts[3] != ""
}
