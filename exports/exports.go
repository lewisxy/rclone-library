// Package exports exports function for c library

package main


import (
	"context"
	// "encoding/base64"
	"encoding/json"
	// "flag"
	// "fmt"
	// "log"
	// "mime"
	"net/http"
	// "net/url"
	// "path/filepath"
	// "regexp"
	// "sort"
	"strings"
	// "sync"
	// "time"
	"io"
	"bytes"
	"C"
	"runtime"

	// "github.com/rclone/rclone/fs/rc/webgui"

	"github.com/pkg/errors"
	// "github.com/prometheus/client_golang/prometheus"
	// "github.com/prometheus/client_golang/prometheus/promhttp"
	// "github.com/skratchdot/open-golang/open"

	// "github.com/rclone/rclone/cmd/serve/httplib"
	// "github.com/rclone/rclone/cmd/serve/httplib/serve"
	"github.com/rclone/rclone/fs"
	// "github.com/rclone/rclone/fs/accounting"
	// "github.com/rclone/rclone/fs/cache"
	// "github.com/rclone/rclone/fs/config"
	// "github.com/rclone/rclone/fs/list"
	"github.com/rclone/rclone/fs/rc"
	"github.com/rclone/rclone/fs/rc/jobs"
	// "github.com/rclone/rclone/fs/rc/rcflags"
	// "github.com/rclone/rclone/lib/random"

	_ "github.com/rclone/rclone/backend/all" // import all backends
	_ "github.com/rclone/rclone/cmd/all"    // import all commands
	_ "github.com/rclone/rclone/lib/plugin" // import plugins
)

func init() {
	// do nothing
}

// call to init the library
//export Cinit
func Cinit() {
	// TODO: what need to be initialized manually?
}

// call to destroy the whole thing
//export Cdestroy
func Cdestroy() {
	// TODO: how to clean up? what happens when rcserver terminates?
	// what about unfinished async jobs?
	runtime.GC()
}

// Caller is responsible for freeing the memory for output
// TODO: how to specify config file?
//export CRPC
func CRPC(method *C.char, input *C.char) (output *C.char, status C.int) {
	res, s := callFunctionJSON(C.GoString(method), C.GoString(input))
	return C.CString(res), C.int(s)
}

// copied from rcserver.go
// writeError writes a formatted error to the output
func writeError(path string, in rc.Params, w io.Writer, err error, status int) {
	fs.Errorf(nil, "rc: %q: error: %v", path, err)
	// Adjust the error return for some well known errors
	errOrig := errors.Cause(err)
	switch {
	case errOrig == fs.ErrorDirNotFound || errOrig == fs.ErrorObjectNotFound:
		status = http.StatusNotFound
	case rc.IsErrParamInvalid(err) || rc.IsErrParamNotFound(err):
		status = http.StatusBadRequest
	}
	// w.WriteHeader(status)
	err = rc.WriteJSON(w, rc.Params{
		"status": status,
		"error":  err.Error(),
		"input":  in,
		"path":   path,
	})
	if err != nil {
		// can't return the error at this point
		fs.Errorf(nil, "rc: failed to write JSON output: %v", err)
	}
}

// operations/uploadfile and core/command are not supported as they need request or response object
// modified from handlePost in rcserver.go
// call a rc function using JSON to input parameters and output the resulted JSON
func callFunctionJSON(method string, input string) (output string, status int) {
	// create a buffer to capture the output
	buf := new(bytes.Buffer)
	in := make(rc.Params)
	err := json.NewDecoder(strings.NewReader(input)).Decode(&in)
	if err != nil {
		// TODO: handle error
		writeError(method, in, buf, errors.Wrap(err, "failed to read input JSON"), http.StatusBadRequest)
		return buf.String(), http.StatusBadRequest
	}

	// Find the call
	call := rc.Calls.Get(method)
	if call == nil {
		writeError(method, in, buf, errors.Errorf("couldn't find method %q", method), http.StatusNotFound)
		return buf.String(), http.StatusNotFound
	}

	// TODO: handle these cases
	if call.NeedsRequest {
		writeError(method, in, buf, errors.Errorf("method %q needs request, not supported", method), http.StatusNotFound)
		return buf.String(), http.StatusNotFound
		// Add the request to RC
		//in["_request"] = r
	}
	if call.NeedsResponse {
		writeError(method, in, buf, errors.Errorf("method %q need response, not supported", method), http.StatusNotFound)
		return buf.String(), http.StatusNotFound
		//in["_response"] = w
	}

	// Check to see if it is async or not
	isAsync, err := in.GetBool("_async")
	if rc.NotErrParamNotFound(err) {
		writeError(method, in, buf, err, http.StatusBadRequest)
		return buf.String(), http.StatusBadRequest
	}
	delete(in, "_async") // remove the async parameter after parsing so vfs operations don't get confused

	fs.Debugf(nil, "rc: %q: with parameters %+v", method, in)
	var out rc.Params
	if isAsync {
		out, err = jobs.StartAsyncJob(call.Fn, in)
	} else {
		//var jobID int64
		//out, jobID, err = jobs.ExecuteJob(r.Context(), call.Fn, in)
		// TODO: what is r.Context()? use Background() for the moment
		out, _, err = jobs.ExecuteJob(context.Background(), call.Fn, in)
		// TODO: probably don't need this, Is jobID is already given in the output?
		//w.Header().Add("x-rclone-jobid", fmt.Sprintf("%d", jobID))
	}
	if err != nil {
		// handle error
		writeError(method, in, buf, err, http.StatusInternalServerError)
		return buf.String(), http.StatusInternalServerError
	}
	if out == nil {
		out = make(rc.Params)
	}

	fs.Debugf(nil, "rc: %q: reply %+v: %v", method, out, err)
	err = rc.WriteJSON(buf, out)
	if err != nil {
		writeError(method, in, buf, err, http.StatusInternalServerError)
		return buf.String(), http.StatusInternalServerError
		fs.Errorf(nil, "rc: failed to write JSON output: %v", err)
	}
	return buf.String(), http.StatusOK
}

// do nothing here
func main() {}