package bootstrap

import (
	"fmt"
	"slices"

	"github.com/ilyakaznacheev/cleanenv"
)

const (
	ProductionEnvironmentName  = "production"
	DevelopmentEnvironmentName = "development"
)

type Env struct {
	CassandraHosts        []string `env:"CASSANDRA_HOSTS" env-required:"true"`
	HTTPPortNumber        int      `env:"HTTP_PORT_NUMBER" env-default:"8080"`
	EnvironmentName       string   `env:"ENVIRONMENT_NAME" env-required:"true"`
	RabbitMQURL           string   `env:"RABBITMQ_URL" env-required:"true"`
	RedisURL              string   `env:"REDIS_URL" env-required:"true"`
	UIDGeneratorStartTime string   `env:"UNIQUE_ID_GENERATOR_START_TIME" env-default:"2024-06-13"`
	MachineID             uint16   `env:"MACHINE_ID" env-required:"true"`
}

func newEnv() (*Env, error) {
	var env Env
	err := cleanenv.ReadConfig(".env", &env)
	if err != nil {
		return nil, err
	}

	if !slices.Contains(
		[]string{DevelopmentEnvironmentName, ProductionEnvironmentName},
		env.EnvironmentName,
	) {
		return nil, fmt.Errorf(
			"ENVIRONMENT_NAME must be one of %s or %s",
			ProductionEnvironmentName,
			DevelopmentEnvironmentName,
		)
	}

	return &env, nil
}
