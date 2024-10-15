package config

import (
	"context"
	"encoding/json"
	"fmt"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
)

const ConfigSecretName = "projects/865842770127/secrets/youtube-custom-feeds-config/versions/latest"
const ClientIdSecretName = "projects/865842770127/secrets/youtube-custom-feeds-client-id-json/versions/latest"
const ApiKeySecretName = "projects/865842770127/secrets/youtube-custom-feeds-api-key/versions/latest"

// Used to retireve secrets from google secret manager
func GetSecret(secretName string) (string, error) {
	ctx := context.Background()

	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return "", fmt.Errorf("in GetSecret(): error creating secretmanager client: %v", err)
	}
	defer client.Close()

	accessRequest := &secretmanagerpb.AccessSecretVersionRequest{
		Name: secretName,
	}

	result, err := client.AccessSecretVersion(ctx, accessRequest)
	if err != nil {
		return "", fmt.Errorf("in GetSecret(): error accessing secret version: %v", err)
	}

	secretData := string(result.Payload.Data)

	return secretData, nil
}

func GetClientId() (string, error) {

	type ClientSecrets struct {
		ClientId string `json:"client_id"`
	}

	var clientStruct ClientSecrets

	clientString, err := GetSecret(ClientIdSecretName)
	if err != nil {
		return "", fmt.Errorf("in GetClientId(): error retrieving client secrets from secrets: %s", err)
	}

	err = json.Unmarshal([]byte(clientString), &clientStruct)
	if err != nil {
		return "", fmt.Errorf("in GetClientId(): error unmarshaling clientString: %s", err)
	}

	return clientStruct.ClientId, nil
}
