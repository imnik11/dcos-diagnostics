package api

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	assertPackage "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"net/http/httptest"
	"net/http"
	"strings"
)

type HelpersTestSuite struct {
	suite.Suite
	assert *assertPackage.Assertions
	testID string
}

// SetUp/Teardown
func (s *HelpersTestSuite) SetupTest() {
	// Setup assert function
	s.assert = assertPackage.New(s.T())

	// Set a unique test ID
	s.testID = fmt.Sprintf("tmp-%d", time.Now().UnixNano())
}
func (s *HelpersTestSuite) TearDownTest() {}

//Tests
func (s *HelpersTestSuite) TestReadFileNoFile() {
	r, err := readFile("/noFile")
	s.assert.Error(err)
	s.assert.Nil(r)
}

func (s *HelpersTestSuite) TestReadFile() {
	// create a test file
	tempFile, err := ioutil.TempFile("", s.testID)
	if err == nil {
		defer os.Remove(tempFile.Name())
	}
	s.assert.NoError(err)
	tempFile.WriteString(s.testID)
	tempFile.Close()

	r, err := readFile(filepath.Join(tempFile.Name()))
	if err == nil {
		defer r.Close()
	}

	s.assert.NotNil(r)
	s.assert.NoError(err)
	buf := new(bytes.Buffer)
	io.Copy(buf, r)
	s.assert.Equal(buf.String(), s.testID)
}

func (s *HelpersTestSuite) TestHttpClientCopyHeadersWhenFollowsRedirects() {
	//Previously there was code that does it directly but
	server := httptest.NewServer(http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "redirect") {
			w.Write([]byte(r.Header.Get("custom-header")))
		} else {
			http.Redirect(w, r, "/redirect", http.StatusFound)
		}
	}))

	client := NewHTTPClient(0, nil)

	req, err := http.NewRequest("GET", server.URL, nil)
	s.assert.NoError(err)
	req.Header.Add("custom-header", "test value")

	resp, err := client.Do(req)
	s.assert.NoError(err)

	body, err := ioutil.ReadAll(resp.Body)
	s.assert.NoError(err)
	s.assert.Equal("test value", string(body))

}

// Run test suit
func TestHelpersTestSuit(t *testing.T) {
	suite.Run(t, new(HelpersTestSuite))
}
