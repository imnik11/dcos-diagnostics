package fetcher

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/dcos/dcos-diagnostics/dcos"
	"github.com/sirupsen/logrus"
)

// EndpointFetchRequest is a struct passed to Fetcher with information about URL to be fetched
type EndpointFetchRequest struct {
	URL      string
	Node     dcos.Node
	FileName string
}

// FetchStatusUpdate is an update message published by Fetcher when EndpointFetchRequest is done. If error occurred	 during
// fetch then Error field is not nil.
type FetchStatusUpdate struct {
	URL   string
	Error error
}

// FetchBulkResponse is a message published when Fetcher finish it job due to cancelled context or closed input chanel
type FetchBulkResponse struct {
	ZipFilePath string
}

// Fetcher is a struct responsible for fetching nodes endpoints
type Fetcher struct {
	file         *os.File
	client       *http.Client
	input        <-chan EndpointFetchRequest
	statusUpdate chan<- FetchStatusUpdate
	output       chan<- FetchBulkResponse
}

// New creates new Fetcher. Fetcher needs to be started with Run()
func New(
	tempdir string,
	client *http.Client,
	input <-chan EndpointFetchRequest,
	statusUpdate chan<- FetchStatusUpdate,
	output chan<- FetchBulkResponse,
) (*Fetcher, error) {
	f, err := ioutil.TempFile(tempdir, "")
	if err != nil {
		return nil, fmt.Errorf("could not create temp zip file in %s: %s", tempdir, err)
	}

	fetcher := &Fetcher{f, client, input, statusUpdate, output}

	return fetcher, nil
}

// Run starts fetcher. This method should be run as a gorutine
func (f *Fetcher) Run(ctx context.Context) {
	zipWriter := zip.NewWriter(f.file)

	f.workOffRequests(ctx, zipWriter)

	filename := f.file.Name()
	if err := zipWriter.Close(); err != nil {
		logrus.WithError(err).Errorf("Could not close zip writer %s", f.file.Name())
	}
	if err := f.file.Close(); err != nil {
		logrus.WithError(err).Errorf("Could not close zip file %s", f.file.Name())
	}
	f.output <- FetchBulkResponse{
		ZipFilePath: filename,
	}
}

func (f *Fetcher) workOffRequests(ctx context.Context, zipWriter *zip.Writer) {
	for {
		select {
		case <-ctx.Done():
			return
		case in, ok := <-f.input:
			if !ok {
				return
			}
			err := getDataToZip(ctx, f.client, in, zipWriter)
			select {
			case <-ctx.Done():
				return
			default:
				f.statusUpdate <- FetchStatusUpdate{
					URL:   in.URL,
					Error: err,
				}
			}
		}
	}
}

func getDataToZip(ctx context.Context, client *http.Client, r EndpointFetchRequest, zipWriter *zip.Writer) error {
	resp, err := get(ctx, client, r.URL)
	if err != nil {
		return fmt.Errorf("could not get from url %s: %s", r.URL, err)
	}
	defer resp.Body.Close()
	if resp.Header.Get("Content-Encoding") == "gzip" {
		r.FileName += ".gz"
	}

	filename := filepath.Join(r.Node.IP+"_"+r.Node.Role, r.FileName)
	zipFile, err := zipWriter.Create(filename)
	if err != nil {
		return fmt.Errorf("could not create a %s in the zip: %s", filename, err)
	}
	if _, err := io.Copy(zipFile, resp.Body); err != nil {
		return fmt.Errorf("could not copy data to zip: %s", err)
	}

	return nil
}

func get(ctx context.Context, client *http.Client, url string) (*http.Response, error) {
	logrus.Debugf("Using URL %s to collect a log", url)
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create a new HTTP request: %s", err)
	}
	request = request.WithContext(ctx)
	request.Header.Add("Accept-Encoding", "gzip")

	resp, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("could not fetch url %s: %s", url, err)
	}

	if resp.StatusCode != http.StatusOK {
		body, e := ioutil.ReadAll(resp.Body)
		defer resp.Body.Close()
		errMsg := fmt.Sprintf("unable to fetch %s. Return code %d.", url, resp.StatusCode)
		if e != nil {
			return nil, fmt.Errorf("%s Could not read body: %s", errMsg, e)
		}
		return nil, fmt.Errorf("%s Body: %s", errMsg, string(body))
	}

	return resp, err
}