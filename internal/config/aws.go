package config

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/vsayfb/gig-platform-notification-lambda/pkg/fb"
)

const ParameterPath = "/yevmiye/app/"

const (
	ParameterRDSSecretArn        = "rds_secret_arn"
	ParameterFirebaseCredentials = "firebase_credentials"
	ParameterDBHost              = "db_host"
	ParameterDBPort              = "db_port"
	ParameterDBName              = "db_name"
)

type rdsSecret struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func loadAWS(ctx context.Context) (*Config, error) {
	awsCfg, err := awscfg.LoadDefaultConfig(ctx)

	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	ssmClient := ssm.NewFromConfig(awsCfg)
	secretClient := secretsmanager.NewFromConfig(awsCfg)

	params, err := loadParameters(ctx, ssmClient)

	if err != nil {
		return nil, err
	}

	var dbSecret rdsSecret

	if err := loadSecret(ctx, secretClient, params[ParameterRDSSecretArn], &dbSecret); err != nil {
		return nil, err
	}

	var creds fb.FirebaseServiceAccount

	err = json.Unmarshal([]byte(params[ParameterFirebaseCredentials]), &creds)

	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal Firebase credentials: %w", err)
	}

	return &Config{
		DB: DBConfig{
			Host:     params[ParameterDBHost],
			Port:     params[ParameterDBPort],
			User:     dbSecret.Username,
			Password: dbSecret.Password,
			Name:     params[ParameterDBName],
			SSLMode:  "require",
		},
		APP: AppConfig{
			Env:                 EnvironmentProduction,
			FirebaseCredentials: &creds,
		},
	}, nil
}

func loadParameters(ctx context.Context, client *ssm.Client) (map[string]string, error) {
	names := []string{
		parameter(ParameterDBHost),
		parameter(ParameterDBPort),
		parameter(ParameterDBName),
		parameter(ParameterFirebaseCredentials),
		parameter(ParameterRDSSecretArn),
	}

	out, err := client.GetParameters(ctx, &ssm.GetParametersInput{
		Names:          names,
		WithDecryption: aws.Bool(true),
	})

	if err != nil {
		return nil, fmt.Errorf("read parameter store: %w", err)
	}

	params := make(map[string]string)

	for _, p := range out.Parameters {
		key := strings.TrimPrefix(aws.ToString(p.Name), ParameterPath)
		params[key] = aws.ToString(p.Value)
	}

	return params, nil
}

func loadSecret(
	ctx context.Context,
	client *secretsmanager.Client,
	name string,
	dst any,
) error {

	out, err := client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(name),
	})

	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(aws.ToString(out.SecretString)), dst)
}

func parameter(name string) string {
	return ParameterPath + name
}
