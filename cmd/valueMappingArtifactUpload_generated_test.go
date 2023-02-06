package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValueMappingArtifactUploadCommand(t *testing.T) {
	t.Parallel()

	testCmd := ValueMappingArtifactUploadCommand()

	// only high level testing performed - details are tested in step generation procedure
	assert.Equal(t, "valueMappingArtifactUpload", testCmd.Use, "command name incorrect")

}