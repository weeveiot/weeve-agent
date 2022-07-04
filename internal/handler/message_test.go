package handler_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/weeveiot/weeve-agent/internal/handler"
)

func TestReadDeployManifestLocalPass(t *testing.T) {
	msg, err := handler.GetStatusMessage()
	if err != nil {
		t.Error("Expected status message, but got error! CAUSE --> ", err)
	}

	assert.Nil(t, msg)
	assert.NotEqual(t, nil, msg)
}
