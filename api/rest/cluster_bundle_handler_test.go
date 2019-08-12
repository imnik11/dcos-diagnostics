package rest

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/dcos/dcos-diagnostics/api/rest/client"
	m "github.com/dcos/dcos-diagnostics/api/rest/client/mock"

	"github.com/dcos/dcos-diagnostics/dcos"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRemoteBundleCreationConflictErrorWhenBundleExists(t *testing.T) {
	workdir, err := ioutil.TempDir("", "work-dir")
	require.NoError(t, err)
	defer os.RemoveAll(workdir)
	// err = os.RemoveAll(workdir) // just check if dcos-diagnostics will create whole path to workdir
	// require.NoError(t, err)

	id := "bundle-0"
	err = os.Mkdir(filepath.Join(workdir, id), 0x666)
	require.NoError(t, err)

	now, err := time.Parse(time.RFC3339, "2015-08-05T08:40:51.620Z")
	require.NoError(t, err)

	tools := new(MockedTools)

	c := new(m.TestifyClient)

	coord := new(mockCoordinator)
	bh := ClusterBundleHandler{
		workDir:    workdir,
		coord:      coord,
		client:     c,
		tools:      tools,
		timeout:    time.Second,
		clock:      &MockClock{now: now},
		urlBuilder: MockURLBuilder{},
	}

	router := mux.NewRouter()
	router.HandleFunc(bundleEndpoint, bh.Create).Methods(http.MethodPut)

	req, err := http.NewRequest(http.MethodPut, client.BundlesEndpoint+"/"+id, nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusConflict, rr.Code)
}

func TestDeleteBundle(t *testing.T) {
	workdir, err := ioutil.TempDir("", "work-dir")
	require.NoError(t, err)
	err = os.RemoveAll(workdir) // just check if dcos-diagnostics will create whole path to workdir
	require.NoError(t, err)

	now, err := time.Parse(time.RFC3339, "2015-08-05T08:40:51.620Z")
	require.NoError(t, err)

	tools := new(MockedTools)
	tools.On("GetMasterNodes").Return([]dcos.Node{
		{
			Role: "master",
			IP:   "192.0.2.2",
		},
		{
			Role: "master",
			IP:   "192.0.2.4",
		},
		{
			Role: "master",
			IP:   "192.0.2.5",
		},
	}, nil)

	ctx := context.TODO()

	id := "bundle-0"
	c := new(m.TestifyClient)
	c.On("Delete", ctx, "http://192.0.2.2", id).Return(&client.DiagnosticsBundleNotFoundError{ID: id})
	c.On("Delete", ctx, "http://192.0.2.4", id).Return(nil)
	c.On("Delete", ctx, "http://192.0.2.5", id).Return(&client.DiagnosticsBundleNotFoundError{ID: id})

	coord := new(mockCoordinator)
	bh := ClusterBundleHandler{
		workDir:    workdir,
		coord:      coord,
		client:     c,
		tools:      tools,
		timeout:    time.Second,
		clock:      &MockClock{now: now},
		urlBuilder: MockURLBuilder{},
	}

	router := mux.NewRouter()
	router.HandleFunc(bundleEndpoint, bh.Delete).Methods(http.MethodDelete)

	req, err := http.NewRequest(http.MethodDelete, client.BundlesEndpoint+"/"+id, nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestDeleteAlreadyDeletedBundle(t *testing.T) {
	workdir, err := ioutil.TempDir("", "work-dir")
	require.NoError(t, err)
	err = os.RemoveAll(workdir) // just check if dcos-diagnostics will create whole path to workdir
	require.NoError(t, err)

	now, err := time.Parse(time.RFC3339, "2015-08-05T08:40:51.620Z")
	require.NoError(t, err)

	tools := new(MockedTools)
	tools.On("GetMasterNodes").Return([]dcos.Node{
		{
			Role: "master",
			IP:   "192.0.2.2",
		},
		{
			Role: "master",
			IP:   "192.0.2.4",
		},
		{
			Role: "master",
			IP:   "192.0.2.5",
		},
	}, nil)

	ctx := context.TODO()

	id := "bundle-0"
	c := new(m.TestifyClient)
	c.On("Delete", ctx, "http://192.0.2.2", id).Return(&client.DiagnosticsBundleNotFoundError{ID: id})
	c.On("Delete", ctx, "http://192.0.2.4", id).Return(&client.DiagnosticsBundleNotCompletedError{ID: id})
	c.On("Delete", ctx, "http://192.0.2.5", id).Return(&client.DiagnosticsBundleNotFoundError{ID: id})

	coord := new(mockCoordinator)
	bh := ClusterBundleHandler{
		workDir:    workdir,
		coord:      coord,
		client:     c,
		tools:      tools,
		timeout:    time.Second,
		clock:      &MockClock{now: now},
		urlBuilder: MockURLBuilder{},
	}

	router := mux.NewRouter()
	router.HandleFunc(bundleEndpoint, bh.Delete).Methods(http.MethodDelete)

	req, err := http.NewRequest(http.MethodDelete, client.BundlesEndpoint+"/"+id, nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotModified, rr.Code)
}

func TestDeleteBundleThatIsntFound(t *testing.T) {
	workdir, err := ioutil.TempDir("", "work-dir")
	require.NoError(t, err)
	err = os.RemoveAll(workdir) // just check if dcos-diagnostics will create whole path to workdir
	require.NoError(t, err)

	now, err := time.Parse(time.RFC3339, "2015-08-05T08:40:51.620Z")
	require.NoError(t, err)

	tools := new(MockedTools)
	tools.On("GetMasterNodes").Return([]dcos.Node{
		{
			Role: "master",
			IP:   "192.0.2.2",
		},
		{
			Role: "master",
			IP:   "192.0.2.4",
		},
		{
			Role: "master",
			IP:   "192.0.2.5",
		},
	}, nil)

	ctx := context.TODO()

	id := "bundle-0"
	c := new(m.TestifyClient)
	c.On("Delete", ctx, "http://192.0.2.2", id).Return(&client.DiagnosticsBundleNotFoundError{ID: id})
	c.On("Delete", ctx, "http://192.0.2.4", id).Return(&client.DiagnosticsBundleNotFoundError{ID: id})
	c.On("Delete", ctx, "http://192.0.2.5", id).Return(&client.DiagnosticsBundleNotFoundError{ID: id})

	coord := new(mockCoordinator)
	bh := ClusterBundleHandler{
		workDir:    workdir,
		coord:      coord,
		client:     c,
		tools:      tools,
		timeout:    time.Second,
		clock:      &MockClock{now: now},
		urlBuilder: MockURLBuilder{},
	}

	router := mux.NewRouter()
	router.HandleFunc(bundleEndpoint, bh.Delete).Methods(http.MethodDelete)

	req, err := http.NewRequest(http.MethodDelete, client.BundlesEndpoint+"/"+id, nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestDeleteUnreadableBundle(t *testing.T) {
	workdir, err := ioutil.TempDir("", "work-dir")
	require.NoError(t, err)
	err = os.RemoveAll(workdir) // just check if dcos-diagnostics will create whole path to workdir
	require.NoError(t, err)

	now, err := time.Parse(time.RFC3339, "2015-08-05T08:40:51.620Z")
	require.NoError(t, err)

	tools := new(MockedTools)
	tools.On("GetMasterNodes").Return([]dcos.Node{
		{
			Role: "master",
			IP:   "192.0.2.2",
		},
		{
			Role: "master",
			IP:   "192.0.2.4",
		},
		{
			Role: "master",
			IP:   "192.0.2.5",
		},
	}, nil)

	ctx := context.TODO()

	id := "bundle-0"
	c := new(m.TestifyClient)
	c.On("Delete", ctx, "http://192.0.2.2", id).Return(&client.DiagnosticsBundleNotFoundError{ID: id})
	c.On("Delete", ctx, "http://192.0.2.4", id).Return(&client.DiagnosticsBundleUnreadableError{ID: id})
	c.On("Delete", ctx, "http://192.0.2.5", id).Return(&client.DiagnosticsBundleNotFoundError{ID: id})

	coord := new(mockCoordinator)
	bh := ClusterBundleHandler{
		workDir:    workdir,
		coord:      coord,
		client:     c,
		tools:      tools,
		timeout:    time.Second,
		clock:      &MockClock{now: now},
		urlBuilder: MockURLBuilder{},
	}

	router := mux.NewRouter()
	router.HandleFunc(bundleEndpoint, bh.Delete).Methods(http.MethodDelete)

	req, err := http.NewRequest(http.MethodDelete, client.BundlesEndpoint+"/"+id, nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestStatusForBundle(t *testing.T) {
	workdir, err := ioutil.TempDir("", "work-dir")
	require.NoError(t, err)
	err = os.RemoveAll(workdir) // just check if dcos-diagnostics will create whole path to workdir
	require.NoError(t, err)

	now, err := time.Parse(time.RFC3339, "2015-08-05T08:40:51.620Z")
	require.NoError(t, err)

	tools := new(MockedTools)
	tools.On("GetMasterNodes").Return([]dcos.Node{
		{
			Role: "master",
			IP:   "192.0.2.2",
		},
		{
			Role: "master",
			IP:   "192.0.2.4",
		},
		{
			Role: "master",
			IP:   "192.0.2.5",
		},
	}, nil)

	ctx := context.TODO()

	c := new(m.TestifyClient)
	c.On("Status", ctx, "http://192.0.2.2", "bundle-0").Return(nil, fmt.Errorf("asdf"))
	c.On("Status", ctx, "http://192.0.2.4", "bundle-0").Return(&client.Bundle{
		ID:      "bundle-0",
		Status:  client.Done,
		Started: now,
		Stopped: now.Add(1 * time.Hour),
	}, nil)
	c.On("Status", ctx, "http://192.0.2.5", "bundle-0").Return(nil, fmt.Errorf("asdf"))

	coord := new(mockCoordinator)
	bh := ClusterBundleHandler{
		workDir:    workdir,
		coord:      coord,
		client:     c,
		tools:      tools,
		timeout:    time.Second,
		clock:      &MockClock{now: now},
		urlBuilder: MockURLBuilder{},
	}

	router := mux.NewRouter()
	router.HandleFunc(bundleEndpoint, bh.Status).Methods(http.MethodGet)

	req, err := http.NewRequest(http.MethodGet, client.BundlesEndpoint+"/bundle-0", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.JSONEq(t, string(jsonMarshal(client.Bundle{
		ID:      "bundle-0",
		Status:  client.Done,
		Started: now,
		Stopped: now.Add(time.Hour),
	})), rr.Body.String())
}

func TestStatusOnMissingBundle(t *testing.T) {
	workdir, err := ioutil.TempDir("", "work-dir")
	require.NoError(t, err)
	err = os.RemoveAll(workdir) // just check if dcos-diagnostics will create whole path to workdir
	require.NoError(t, err)

	now, err := time.Parse(time.RFC3339, "2015-08-05T08:40:51.620Z")
	require.NoError(t, err)

	tools := new(MockedTools)
	tools.On("GetMasterNodes").Return([]dcos.Node{
		{
			Role: "master",
			IP:   "192.0.2.2",
		},
		{
			Role: "master",
			IP:   "192.0.2.4",
		},
		{
			Role: "master",
			IP:   "192.0.2.5",
		},
	}, nil)

	ctx := context.TODO()

	id := "bundle-0"
	c := new(m.TestifyClient)
	c.On("Status", ctx, "http://192.0.2.2", id).Return(nil, &client.DiagnosticsBundleNotFoundError{ID: id})
	c.On("Status", ctx, "http://192.0.2.4", id).Return(nil, &client.DiagnosticsBundleNotFoundError{ID: id})
	c.On("Status", ctx, "http://192.0.2.5", id).Return(nil, &client.DiagnosticsBundleNotFoundError{ID: id})

	coord := new(mockCoordinator)
	bh := ClusterBundleHandler{
		workDir:    workdir,
		coord:      coord,
		client:     c,
		tools:      tools,
		timeout:    time.Second,
		clock:      &MockClock{now: now},
		urlBuilder: MockURLBuilder{},
	}

	router := mux.NewRouter()
	router.HandleFunc(bundleEndpoint, bh.Status).Methods(http.MethodGet)

	req, err := http.NewRequest(http.MethodGet, client.BundlesEndpoint+"/bundle-0", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestStatusForUnreadableBundle(t *testing.T) {
	workdir, err := ioutil.TempDir("", "work-dir")
	require.NoError(t, err)
	err = os.RemoveAll(workdir) // just check if dcos-diagnostics will create whole path to workdir
	require.NoError(t, err)

	now, err := time.Parse(time.RFC3339, "2015-08-05T08:40:51.620Z")
	require.NoError(t, err)

	tools := new(MockedTools)
	tools.On("GetMasterNodes").Return([]dcos.Node{
		{
			Role: "master",
			IP:   "192.0.2.2",
		},
	}, nil)

	ctx := context.TODO()

	id := "bundle-0"
	c := new(m.TestifyClient)
	c.On("Status", ctx, "http://192.0.2.2", id).Return(nil, &client.DiagnosticsBundleUnreadableError{ID: id})

	coord := new(mockCoordinator)
	bh := ClusterBundleHandler{
		workDir:    workdir,
		coord:      coord,
		client:     c,
		tools:      tools,
		timeout:    time.Second,
		clock:      &MockClock{now: now},
		urlBuilder: MockURLBuilder{},
	}

	router := mux.NewRouter()
	router.HandleFunc(bundleEndpoint, bh.Status).Methods(http.MethodGet)

	req, err := http.NewRequest(http.MethodGet, client.BundlesEndpoint+"/bundle-0", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestDownloadBundle(t *testing.T) {
	workdir, err := ioutil.TempDir("", "work-dir")
	require.NoError(t, err)
	err = os.RemoveAll(workdir) // just check if dcos-diagnostics will create whole path to workdir
	require.NoError(t, err)

	now, err := time.Parse(time.RFC3339, "2015-08-05T08:40:51.620Z")
	require.NoError(t, err)

	tools := new(MockedTools)
	tools.On("GetMasterNodes").Return([]dcos.Node{
		{
			Role: "master",
			IP:   "192.0.2.2",
		},
		{
			Role: "master",
			IP:   "192.0.2.4",
		},
		{
			Role: "master",
			IP:   "192.0.2.5",
		},
	}, nil)

	bundleZip, err := filepath.Abs(filepath.Join("testdata", "combined.zip"))
	require.NoError(t, err)

	expectedBytes, err := ioutil.ReadFile(bundleZip)
	require.NoError(t, err)

	ctx := context.TODO()

	id := "bundle-0"
	c := new(m.TestifyClient)
	c.On("Status", ctx, "http://192.0.2.2", id).Return(nil, &client.DiagnosticsBundleNotFoundError{ID: id})
	c.On("Status", ctx, "http://192.0.2.4", id).Return(nil, &client.DiagnosticsBundleNotFoundError{ID: id})
	c.On("Status", ctx, "http://192.0.2.5", id).Return(&client.Bundle{
		ID: "bundle-0",
	}, nil)
	c.On("GetFile", ctx, "http://192.0.2.5", id, mock.AnythingOfType("string")).Return(func(ctx context.Context, url string, id string, tempZipFile string) error {
		// copy the testdata zip to the location Client is expected to put the downloaded zip
		f, err := os.Create(tempZipFile)
		require.NoError(t, err)
		io.Copy(f, bytes.NewBuffer(expectedBytes))
		return nil
	})

	coord := new(mockCoordinator)
	bh := ClusterBundleHandler{
		workDir:    workdir,
		coord:      coord,
		client:     c,
		tools:      tools,
		timeout:    time.Second,
		clock:      &MockClock{now: now},
		urlBuilder: MockURLBuilder{},
	}

	router := mux.NewRouter()
	router.HandleFunc(bundleFileEndpoint, bh.Download).Methods(http.MethodGet)

	req, err := http.NewRequest(http.MethodGet, client.BundlesEndpoint+"/"+id+"/file", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	assert.Equal(t, bytes.NewBuffer(expectedBytes), rr.Body)
}

func TestDownloadMissingBundle(t *testing.T) {
	workdir, err := ioutil.TempDir("", "work-dir")
	require.NoError(t, err)
	err = os.RemoveAll(workdir) // just check if dcos-diagnostics will create whole path to workdir
	require.NoError(t, err)

	now, err := time.Parse(time.RFC3339, "2015-08-05T08:40:51.620Z")
	require.NoError(t, err)

	tools := new(MockedTools)
	tools.On("GetMasterNodes").Return([]dcos.Node{
		{
			Role: "master",
			IP:   "192.0.2.2",
		},
		{
			Role: "master",
			IP:   "192.0.2.4",
		},
		{
			Role: "master",
			IP:   "192.0.2.5",
		},
	}, nil)

	ctx := context.TODO()

	id := "bundle-0"
	c := new(m.TestifyClient)
	c.On("Status", ctx, "http://192.0.2.2", id).Return(nil, &client.DiagnosticsBundleNotFoundError{ID: id})
	c.On("Status", ctx, "http://192.0.2.4", id).Return(nil, &client.DiagnosticsBundleNotFoundError{ID: id})
	c.On("Status", ctx, "http://192.0.2.5", id).Return(nil, &client.DiagnosticsBundleNotFoundError{ID: id})

	coord := new(mockCoordinator)
	bh := ClusterBundleHandler{
		workDir:    workdir,
		coord:      coord,
		client:     c,
		tools:      tools,
		timeout:    time.Second,
		clock:      &MockClock{now: now},
		urlBuilder: MockURLBuilder{},
	}

	router := mux.NewRouter()
	router.HandleFunc(bundleFileEndpoint, bh.Download).Methods(http.MethodGet)

	req, err := http.NewRequest(http.MethodGet, client.BundlesEndpoint+"/"+id+"/file", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestDownloadUnreadableBundle(t *testing.T) {
	workdir, err := ioutil.TempDir("", "work-dir")
	require.NoError(t, err)
	err = os.RemoveAll(workdir) // just check if dcos-diagnostics will create whole path to workdir
	require.NoError(t, err)

	now, err := time.Parse(time.RFC3339, "2015-08-05T08:40:51.620Z")
	require.NoError(t, err)

	tools := new(MockedTools)
	tools.On("GetMasterNodes").Return([]dcos.Node{
		{
			Role: "master",
			IP:   "192.0.2.2",
		},
		{
			Role: "master",
			IP:   "192.0.2.4",
		},
		{
			Role: "master",
			IP:   "192.0.2.5",
		},
	}, nil)

	ctx := context.TODO()

	id := "bundle-0"
	c := new(m.TestifyClient)
	c.On("Status", ctx, "http://192.0.2.2", id).Return(nil, &client.DiagnosticsBundleNotFoundError{ID: id})
	c.On("Status", ctx, "http://192.0.2.4", id).Return(nil, &client.DiagnosticsBundleNotFoundError{ID: id})
	c.On("Status", ctx, "http://192.0.2.5", id).Return(nil, &client.DiagnosticsBundleUnreadableError{ID: id})

	coord := new(mockCoordinator)
	bh := ClusterBundleHandler{
		workDir:    workdir,
		coord:      coord,
		client:     c,
		tools:      tools,
		timeout:    time.Second,
		clock:      &MockClock{now: now},
		urlBuilder: MockURLBuilder{},
	}

	router := mux.NewRouter()
	router.HandleFunc(bundleFileEndpoint, bh.Download).Methods(http.MethodGet)

	req, err := http.NewRequest(http.MethodGet, client.BundlesEndpoint+"/"+id+"/file", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestListWithBundlesOnMultipleMasters(t *testing.T) {
	workdir, err := ioutil.TempDir("", "work-dir")
	require.NoError(t, err)
	err = os.RemoveAll(workdir) // just check if dcos-diagnostics will create whole path to workdir
	require.NoError(t, err)

	now, err := time.Parse(time.RFC3339, "2015-08-05T08:40:51.620Z")
	require.NoError(t, err)

	tools := new(MockedTools)
	tools.On("GetMasterNodes").Return([]dcos.Node{
		{
			Role: "master",
			IP:   "192.0.2.2",
		},
		{
			Role: "master",
			IP:   "192.0.2.4",
		},
		{
			Role: "master",
			IP:   "192.0.2.5",
		},
	}, nil)

	ctx := context.TODO()

	expectedBundles := []*client.Bundle{
		{
			ID: "bundle-0",
		},
		{
			ID: "bundle-1",
		},
		{
			ID: "bundle-2",
		},
	}

	c := new(m.TestifyClient)
	c.On("List", ctx, "http://192.0.2.2").Return([]*client.Bundle{expectedBundles[0]}, nil)
	c.On("List", ctx, "http://192.0.2.4").Return([]*client.Bundle{expectedBundles[1]}, nil)
	c.On("List", ctx, "http://192.0.2.5").Return([]*client.Bundle{expectedBundles[2]}, nil)

	coord := new(mockCoordinator)
	bh := ClusterBundleHandler{
		workDir:    workdir,
		coord:      coord,
		client:     c,
		tools:      tools,
		timeout:    time.Second,
		clock:      &MockClock{now: now},
		urlBuilder: MockURLBuilder{},
	}

	router := mux.NewRouter()
	router.HandleFunc(client.BundlesEndpoint, bh.List).Methods(http.MethodGet)

	req, err := http.NewRequest(http.MethodGet, client.BundlesEndpoint, nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.JSONEq(t, string(jsonMarshal(expectedBundles)), rr.Body.String())
}

func TestRemoteBundleCreation(t *testing.T) {
	workdir, err := ioutil.TempDir("", "work-dir")
	require.NoError(t, err)
	err = os.RemoveAll(workdir) // just check if dcos-diagnostics will create whole path to workdir
	require.NoError(t, err)

	now, err := time.Parse(time.RFC3339, "2015-08-05T08:40:51.620Z")
	require.NoError(t, err)

	tools := new(MockedTools)
	tools.On("GetMasterNodes").Return([]dcos.Node{
		{
			Leader: true,
			Role:   "master",
			IP:     "192.0.2.2",
		},
	}, nil)
	tools.On("GetAgentNodes").Return([]dcos.Node{
		{
			Role: "agent",
			IP:   "192.0.2.1",
		},
		{
			Role: "agent",
			IP:   "192.0.2.3",
		},
	}, nil)

	bundleZip, err := filepath.Abs(filepath.Join("testdata", "combined.zip"))
	require.NoError(t, err)

	expectedBytes, err := ioutil.ReadFile(bundleZip)
	require.NoError(t, err)

	ctx := context.TODO()

	c := new(m.TestifyClient)
	c.On("Status", ctx, "http://192.0.2.2", "bundle-0").Return(&client.Bundle{
		ID:      "bundle-0",
		Started: now.Add(time.Hour),
		Stopped: now.Add(2 * time.Hour),
		Status:  client.Done,
	}, nil)
	c.On("GetFile", ctx, "http://192.0.2.2", "bundle-0", mock.AnythingOfType("string")).Return(func(_ context.Context, _ string, _ string, tempZipFile string) error {
		// copy the testdata zip to the location Client is expected to put the downloaded zip
		f, err := os.Create(tempZipFile)
		require.NoError(t, err)
		io.Copy(f, bytes.NewBuffer(expectedBytes))
		return nil
	})
	c.On("Delete", ctx, "http://192.0.2.2", "bundle-0").Return(nil)

	coord := new(mockCoordinator)
	bh := ClusterBundleHandler{
		workDir:    workdir,
		coord:      coord,
		client:     c,
		tools:      tools,
		timeout:    time.Second,
		clock:      &MockClock{now: now},
		urlBuilder: MockURLBuilder{},
	}

	router := mux.NewRouter()
	router.HandleFunc(bundleEndpoint, bh.Create).Methods(http.MethodPut)
	router.HandleFunc(bundleEndpoint, bh.Status).Methods(http.MethodGet)
	router.HandleFunc(bundleEndpoint, bh.Delete).Methods(http.MethodDelete)
	router.HandleFunc(bundleFileEndpoint, bh.Download).Methods(http.MethodGet)

	t.Run("send creation request", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPut, client.BundlesEndpoint+"/bundle-0", nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.JSONEq(t, string(jsonMarshal(client.Bundle{
			ID:      "bundle-0",
			Status:  client.Started,
			Started: now.Add(time.Hour),
		})), rr.Body.String())

	})

	t.Run("get status", func(t *testing.T) {
		rr := httptest.NewRecorder()

		tries := 0
		retryLimit := 100
		for { // busy wait for bundle
			time.Sleep(time.Millisecond)
			req, err := http.NewRequest(http.MethodGet, client.BundlesEndpoint+"/bundle-0", nil)
			require.NoError(t, err)

			rr = httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if strings.Contains(rr.Body.String(), client.Done.String()) {
				break
			}
			tries++
			// keeps the test suite from hanging if something is wrong
			require.True(t, tries < retryLimit, "status wait loop exceeded retry limit")
		}

		req, err := http.NewRequest(http.MethodGet, client.BundlesEndpoint+"/bundle-0", nil)
		require.NoError(t, err)

		rr = httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.JSONEq(t, string(jsonMarshal(client.Bundle{
			ID:      "bundle-0",
			Status:  client.Done,
			Started: now.Add(time.Hour),
			Stopped: now.Add(2 * time.Hour),
			Errors:  []string{},
		})), rr.Body.String())
	})

	t.Run("get bundle-0 file and validate it", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, client.BundlesEndpoint+"/bundle-0/file", nil)
		require.NoError(t, err)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		reader, err := zip.NewReader(bytes.NewReader(rr.Body.Bytes()), int64(len(rr.Body.Bytes())))
		require.NoError(t, err)

		expectedContents := []string{
			"192.0.2.1/",
			"192.0.2.1/test.txt",
			"192.0.2.2/",
			"192.0.2.2/test.txt",
			"192.0.2.3/",
			"192.0.2.3/test.txt",
		}

		filenames := []string{}
		for _, f := range reader.File {
			filenames = append(filenames, f.Name)
		}
		sort.Strings(filenames)

		assert.Equal(t, expectedContents, filenames)
	})

	t.Run("delete bundle", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodDelete, client.BundlesEndpoint+"/bundle-0", nil)
		require.NoError(t, err)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})
}

type mockCoordinator struct{}

func (c mockCoordinator) CreateBundle(ctx context.Context, id string, nodes []node) <-chan BundleStatus {
	statuses := make(chan BundleStatus, len(nodes))

	for _, n := range nodes {
		node := n
		statuses <- BundleStatus{
			id:   id,
			node: node,
			done: true,
		}
	}

	return statuses
}

func (c mockCoordinator) CollectBundle(ctx context.Context, id string, numBundles int, statuses <-chan BundleStatus) (string, error) {
	return filepath.Abs(filepath.Join("testdata", "combined.zip"))
}

type MockURLBuilder struct{}

func (m MockURLBuilder) BaseURL(ip net.IP, _ string) (string, error) {
	return fmt.Sprintf("http://%s", ip), nil
}
