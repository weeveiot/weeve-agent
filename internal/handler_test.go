package internal

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/controller"
)

/*  */
func TestPipelinePost(t *testing.T) {

	req, err := http.NewRequest("POST", "/pipelines", nil)
	if err != nil {
		t.Fatal(err)
	}
	// q := req.URL.Query()
	// q.Add("id", "43bb4d42-eed5-4a86-ba3c-946b97e3b085")
	// req.URL.RawQuery = q.Encode()
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(controller.POSTpipelines)
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

}
