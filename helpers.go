package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"

	"github.com/ghodss/yaml"
	"github.com/hashicorp/vault/api"
)

func YamlToMap(y []byte) (map[string]interface{}, error) {
	m := make(map[string]interface{})
	err := yaml.Unmarshal(y, &m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func JsonBytesToYamlBytes(b []byte) ([]byte, error) {
	return yaml.JSONToYAML(b)
}

func YamlBytesToJSONBytes(b []byte) ([]byte, error) {
	return yaml.YAMLToJSON(b)
}

func MarshalViaJSONToYAML(v interface{}) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return JsonBytesToYamlBytes(b)
}

func GetYamlMap(c interface{}) (map[string]interface{}, error) {
	yamlBytes, err := MarshalViaJSONToYAML(c)
	if err != nil {
		return nil, err
	}
	yamlMap, err := YamlToMap(yamlBytes)
	if err != nil {
		return nil, err
	}
	return yamlMap, nil
}

func Run(name string, args ...string) error {
	stdout, stderr, err := RunResult(name, args...)
	if err != nil {
		fmt.Println(stdout)
		fmt.Println(stderr)
		return err
	}
	return nil
}

func RunResult(name string, args ...string) (string, string, error) {
	c := exec.Command(name, args...)

	var stdOut bytes.Buffer
	var stdErr bytes.Buffer

	c.Stdout = &stdOut
	c.Stderr = &stdErr

	err := c.Run()
	if err != nil {
		return "", "", fmt.Errorf("%v: %s", err, stdErr.String())
	}

	return stdOut.String(), stdErr.String(), nil
}

func GetVaultClient(insecure bool, ldapVaultUser string, ldapVaultPassword string) (*api.Client, error) {
	var vclient *api.Client
	conf := api.DefaultConfig()
	var err error
	if insecure {
		conf.ConfigureTLS(&api.TLSConfig{Insecure: insecure})
	}
	vclient, err = api.NewClient(conf)
	vclient.SetAddress("https://csxvault.sickcn.net/") // TODO: remove tight coupling and make configurable
	if err != nil {
		return nil, err
	}

	if ldapVaultPassword == "" || ldapVaultUser == "" {
		return nil, errors.New("LDAP User and Password for Vault required")
	}

	// to pass the password
	options := map[string]interface{}{
		"password": ldapVaultPassword,
	}

	// the login path, this is configurable, change userpass to ldap etc
	path := fmt.Sprintf("/auth/ldap/login/%s", ldapVaultUser)
	// PUT call to get a token
	secret, err := vclient.Logical().Write(path, options)
	if err != nil {
		return nil, err
	}
	vclient.SetToken(secret.Auth.ClientToken)
	return vclient, nil
}
