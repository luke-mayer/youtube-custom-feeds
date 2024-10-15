package config

import (
	"context"
	"fmt"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
)

const ConfigSecretName = "projects/865842770127/secrets/youtube-custom-feeds-config"
const ClientIdSecretName = "projects/865842770127/secrets/youtube-custom-feeds-client-id-json"
const ApiKeySecretName = "projects/865842770127/secrets/youtube-custom-feeds-api-key"

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
