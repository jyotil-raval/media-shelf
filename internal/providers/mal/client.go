package mal

import (
	"context"
	"fmt"

	pb "github.com/jyotil-raval/mal-updater/proto/animepb"
	"github.com/jyotil-raval/media-shelf/internal/models"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn  *grpc.ClientConn
	anime pb.AnimeServiceClient
}

func NewClient(target string) (*Client, error) {
	conn, err := grpc.NewClient(target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("connecting to mal-updater gRPC: %w", err)
	}

	return &Client{
		conn:  conn,
		anime: pb.NewAnimeServiceClient(conn),
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) GetAnime(ctx context.Context, id string) (*models.MediaItem, error) {
	resp, err := c.anime.GetAnime(ctx, &pb.GetAnimeRequest{Id: id})
	if err != nil {
		return nil, fmt.Errorf("GetAnime(%s): %w", id, err)
	}

	return &models.MediaItem{
		MediaType: "anime",
		Source:    "mal",
		SourceID:  id,
		Title:     resp.Title,
		SubType:   resp.MediaType,
		Total:     int(resp.NumEpisodes),
	}, nil
}
