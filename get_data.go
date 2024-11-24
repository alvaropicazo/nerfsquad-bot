package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

func get_api_key() (string, error) {
	keys_file := make(map[string]interface{})

	keys_byte, err := os.ReadFile("./config/keys.yaml")
	if err != nil {
		return "yamlFile.Get err #%v ", err
	}
	err = yaml.Unmarshal(keys_byte, keys_file)
	if err != nil {
		return "Unmarshal: %v", err
	}

	return keys_file["keys"].([]interface{})[0].(string), nil
}

func get_apis_obj() (map[string]interface{}, error) {
	apis_file := make(map[string]interface{})

	apis_byte, err := os.ReadFile("./config/apis.yaml")
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(apis_byte, apis_file)
	if err != nil {
		return nil, err
	}

	return apis_file, nil
}

func get_wallet_obj() (map[interface{}]interface{}, error) {
	wallets_file := make(map[interface{}]interface{})

	wallets_byte, err := os.ReadFile("./config/wallets.yaml")
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(wallets_byte, wallets_file)
	if err != nil {
		return nil, err
	}

	return wallets_file, nil

}
