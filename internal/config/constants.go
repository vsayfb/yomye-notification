package config

const AppEnv = "APP_ENV"

const (
	EnvironmentDevelopment = "development"
	EnvironmentProduction  = "production"
)

const (
	EnvDBHost     = "DB_HOST"
	EnvDBPort     = "DB_PORT"
	EnvDBUser     = "DB_USER"
	EnvDBPassword = "DB_PASSWORD"
	EnvDBName     = "DB_NAME"
	EnvDBSSLMode  = "DB_SSLMODE"

	EnvFirebaseCredentialsPath = "FIREBASE_CREDENTIALS_PATH"
	EnvOtelCollectorAddr       = "OTEL_COLLECTOR_ADDR"
)

const (
	DefaultSSLMode           = "disable"
	DefaultOtelCollectorAddr = "localhost:4317"
)
