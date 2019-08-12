package mock

import (
	"context"

	"github.com/dcos/dcos-diagnostics/api/rest/client"
)

type Client struct {
	MockCreateBundle func(ctx context.Context, node string, ID string) (*client.Bundle, error)
	MockStatus       func(ctx context.Context, node string, ID string) (*client.Bundle, error)
	MockGetFile      func(ctx context.Context, node string, ID string, path string) (err error)
	MockList         func(ctx context.Context, node string) ([]*client.Bundle, error)
	MockDelete       func(ctx context.Context, node string, id string) error
}

func (_m *Client) CreateBundle(ctx context.Context, node string, ID string) (*client.Bundle, error) {
	return _m.MockCreateBundle(ctx, node, ID)
}

func (_m *Client) Delete(ctx context.Context, node string, id string) error {
	return _m.MockDelete(ctx, node, id)
}

func (_m *Client) GetFile(ctx context.Context, node string, ID string, path string) error {
	return _m.MockGetFile(ctx, node, ID, path)
}

func (_m *Client) List(ctx context.Context, node string) ([]*client.Bundle, error) {
	return _m.MockList(ctx, node)
}

func (_m *Client) Status(ctx context.Context, node string, ID string) (*client.Bundle, error) {
	return _m.MockStatus(ctx, node, ID)
}
