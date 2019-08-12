package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

const BundlesEndpoint = "/system/health/v1/node/diagnostics"

type Bundle struct {
	ID      string    `json:"id,omitempty"`
	Size    int64     `json:"size,omitempty"` // length in bytes for regular files; 0 when Canceled or Deleted
	Status  Status    `json:"status"`
	Started time.Time `json:"started_at,omitempty"`
	Stopped time.Time `json:"stopped_at,omitempty"`
	Errors  []string  `json:"errors,omitempty"`
}

// Client is an interface that can talk with dcos-diagnostics REST API and manipulate remote bundles
type Client interface {
	// CreateBundle requests the given node to start a bundle creation process with that is identified by the given ID
	CreateBundle(ctx context.Context, node string, ID string) (*Bundle, error)
	// Status returns the status of the bundle with the given ID on the given node
	Status(ctx context.Context, node string, ID string) (*Bundle, error)
	// GetFile downloads the bundle file of the bundle with the given ID from the node
	// url and save it to local filesystem under given path.
	// Returns an error if there were a problem.
	GetFile(ctx context.Context, node string, ID string, path string) (err error)
	// List will get the list of available bundles on the given node
	List(ctx context.Context, node string) ([]*Bundle, error)
	// Delete will delete the bundle with the given ID from the given node
	Delete(ctx context.Context, node string, id string) error
}

type DiagnosticsClient struct {
	client *http.Client
}

// NewDiagnosticsClient constructs a diagnostics client
func NewDiagnosticsClient(client *http.Client) DiagnosticsClient {
	return DiagnosticsClient{
		client: client,
	}
}

func (d DiagnosticsClient) CreateBundle(ctx context.Context, node string, ID string) (*Bundle, error) {
	url := remoteURL(node, ID)

	logrus.WithField("ID", ID).WithField("url", url).Info("sending bundle creation request")

	type payload struct {
		Type Type `json:"type"`
	}

	body := jsonMarshal(payload{
		Type: Local,
	})

	request, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	request.WithContext(ctx)

	resp, err := d.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	err = handleErrorCode(resp, url, ID)
	if err != nil {
		return nil, err
	}

	bundle := &Bundle{}
	err = json.NewDecoder(resp.Body).Decode(bundle)
	if err != nil {
		return nil, err
	}

	return bundle, nil
}

func (d DiagnosticsClient) Status(ctx context.Context, node string, ID string) (*Bundle, error) {
	url := remoteURL(node, ID)

	logrus.WithField("ID", ID).WithField("url", url).Info("checking status of bundle")

	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	request.WithContext(ctx)

	resp, err := d.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bundle := &Bundle{}

	err = handleErrorCode(resp, url, ID)
	if err != nil {
		return nil, err
	}

	err = json.NewDecoder(resp.Body).Decode(bundle)
	if err != nil {
		return nil, err
	}

	return bundle, nil
}

func (d DiagnosticsClient) GetFile(ctx context.Context, node string, ID string, path string) error {
	url := fmt.Sprintf("%s/file", remoteURL(node, ID))

	logrus.WithField("ID", ID).WithField("url", url).Info("downloading local bundle from node")

	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := d.client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	err = handleErrorCode(resp, url, ID)
	if err != nil {
		return err
	}

	destinationFile, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("could not create a file: %s", err)
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func (d DiagnosticsClient) List(ctx context.Context, node string) ([]*Bundle, error) {
	url := fmt.Sprintf("%s%s", node, BundlesEndpoint)

	logrus.WithField("node", node).Info("getting list of bundles from node")

	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := d.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// there are no expected error messages that could come from this so having
	// a null ID should be fine as the only case this should hit is the default
	// unexpected status code case.
	err = handleErrorCode(resp, url, "")
	if err != nil {
		return nil, err
	}

	bundles := []*Bundle{}
	err = json.NewDecoder(resp.Body).Decode(&bundles)
	if err != nil {
		return nil, err
	}

	return bundles, nil
}

func (d DiagnosticsClient) Delete(ctx context.Context, node string, id string) error {
	url := remoteURL(node, id)

	logrus.WithField("node", node).WithField("ID", id).Info("deleting bundle from node")

	request, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	resp, err := d.client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return handleErrorCode(resp, url, id)
}

func handleErrorCode(resp *http.Response, url string, bundleID string) error {
	switch {
	case resp.StatusCode == http.StatusNotModified:
		return &DiagnosticsBundleNotCompletedError{ID: bundleID}
	case resp.StatusCode == http.StatusNotFound:
		return &DiagnosticsBundleNotFoundError{ID: bundleID}
	case resp.StatusCode == http.StatusInternalServerError:
		return &DiagnosticsBundleUnreadableError{ID: bundleID}
	case resp.StatusCode != http.StatusOK:
		body := make([]byte, 100)
		resp.Body.Read(body)
		return fmt.Errorf("received unexpected status code [%d] from %s: %s", resp.StatusCode, url, string(body))
	}
	return nil
}

func remoteURL(node string, ID string) string {
	url := fmt.Sprintf("%s%s/%s", node, BundlesEndpoint, ID)
	return url
}

// jsonMarshal is a replacement for json.Marshal when we are 100% sure
// there won't now be any error on marshaling.
func jsonMarshal(v interface{}) []byte {
	rawJSON, err := json.Marshal(v)

	if err != nil {
		logrus.WithError(err).Fatalf("Could not marshal %v: %s", v, err)
	}
	return rawJSON
}
