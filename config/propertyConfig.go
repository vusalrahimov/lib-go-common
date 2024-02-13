package config

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/vault/api"
	"github.com/spf13/viper"
	_ "github.com/spf13/viper/remote"
	"log"
	"os"
	"strings"
)

const (
	ConsulProvider   = "consul"
	AuthPath         = "auth/approle/login"
	VaultSecretPath  = "/secret"
	VaultRoleIdKey   = "VAULT_ROLE_ID"
	VaultSecretIdKey = "VAULT_SECRET_ID"
)

func init() {
	loadConfig()
}

func loadConfig() {
	var env = os.Getenv("APP_ENV")
	if env == "local" {
		loadLocalConfig()
	} else {
		loadRemoteConfig(env)
	}
}

func loadLocalConfig() {
	viper.SetConfigName("config-local")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("resources")
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatal(err)
	}
}

func loadEnvConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("resources")
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatal(err)
	}
}

func loadRemoteConfig(env string) {
	loadEnvConfig()
	endpoint, path := buildConsulEndpointAndPath(env)
	err := viper.AddRemoteProvider(ConsulProvider, endpoint, path)
	if err != nil {
		panic(err)
	}
	consulErr := viper.ReadRemoteConfig()
	if consulErr != nil {
		log.Fatal(consulErr)
	}
	viper.SetConfigType("json")
	vaultConfigErr := viper.ReadConfig(strings.NewReader(getVaultConfigAsJson(env)))
	if vaultConfigErr != nil {
		log.Fatal(vaultConfigErr)
	}
}

func getVaultConfigAsJson(env string) string {
	var vaultClient = getVaultClient()
	var vaultPath = viper.GetString("vault.path")
	secretData, err := vaultClient.KVv2(VaultSecretPath).Get(context.Background(), fmt.Sprintf("%s%s", vaultPath, env))
	if err != nil {
		log.Fatal(err)
	}

	var jsonData, jsonError = json.MarshalIndent(secretData.Data, "", "  ")
	if jsonError != nil {
		log.Fatal(jsonError)
	}

	return string(jsonData)
}

func buildConsulEndpointAndPath(env string) (string, string) {
	var host = viper.GetString("consul.host")
	var port = viper.GetString("consul.port")
	var consulPath = viper.GetString("consul.path")

	var endpoint = fmt.Sprintf("%s:%s", host, port)
	var path = fmt.Sprintf("%s%s", consulPath, env)

	return endpoint, path
}

func getVaultClient() *api.Client {
	var roleId = os.Getenv(VaultRoleIdKey)
	var secretId = os.Getenv(VaultSecretIdKey)
	vaultConfig := api.DefaultConfig()
	vaultConfig.Address = buildVaultEndpoint()
	vaultClient, err := api.NewClient(vaultConfig)

	if err != nil {
		log.Fatal(err)
	}

	var secret = &api.Secret{
		Data: map[string]interface{}{"role_id": roleId, "secret_id": secretId},
	}

	response, err := vaultClient.Logical().Write(AuthPath, secret.Data)
	if err != nil {
		log.Fatal(err)
	}

	if response.Auth == nil || response.Auth.ClientToken == "" {
		log.Fatal("Vault Authentication failed")
	}
	vaultClient.SetToken(response.Auth.ClientToken)
	return vaultClient
}

func buildVaultEndpoint() string {
	var scheme = viper.GetString("vault.scheme")
	var host = viper.GetString("vault.host")
	var port = viper.GetInt("vault.port")

	return fmt.Sprintf("%s://%s:%d", scheme, host, port)
}
