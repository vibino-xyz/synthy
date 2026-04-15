package pinecone

import (
	"log/slog"
	"os"

	"github.com/pinecone-io/go-pinecone/v4/pinecone"
)

func NewConnection() (*pinecone.IndexConnection, error) {

	client, err := pinecone.NewClient(pinecone.NewClientParams{
		ApiKey: os.Getenv("PINECONE_API_KEY"),
	})
	if err != nil {
		slog.Error("Failed to create Client", "error", err)
		return nil, err
	}

	indexConnection, err := client.Index(pinecone.NewIndexConnParams{
		Host: os.Getenv("PINECONE_HOST"),
	})
	if err != nil {
		slog.Error("Failed to connect to index", "error", err)
		return nil, err
	}

	slog.Info("Connected to Pinecone successfully")
	return indexConnection, nil
}
