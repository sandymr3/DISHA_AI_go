package data

import (
	"context"
	"fmt"
	"log/slog"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/option"
)

// FirestoreClient wraps the firestore client for DISHA DB operations.
type FirestoreClient struct {
	Client *firestore.Client
}

// Ensure you export the credentials as a file, and point the GOOGLE_APPLICATION_CREDENTIALS
// environment variable to it, or pass the json credential path to this function.
// For MVP, if credentials file is empty, we will try default auth.
func NewFirestoreClient(ctx context.Context, projectID string, credsJSON string) (*FirestoreClient, error) {
	var opts []option.ClientOption
	if credsJSON != "" {
		opts = append(opts, option.WithCredentialsJSON([]byte(credsJSON)))
	}

	config := &firebase.Config{ProjectID: projectID}
	app, err := firebase.NewApp(ctx, config, opts...)
	if err != nil {
		return nil, fmt.Errorf("error initializing firebase app: %v", err)
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting firestore client: %v", err)
	}

	slog.Info("Firestore client initialized", "projectID", projectID)

	return &FirestoreClient{
		Client: client,
	}, nil
}

// Close closes the underlying firestore client.
func (f *FirestoreClient) Close() error {
	if f.Client != nil {
		return f.Client.Close()
	}
	return nil
}
