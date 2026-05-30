package metadata_test

import (
	"testing"

	"github.com/microsoft/typescript-go/angular-packages/compiler_cli/ngtsc/metadata"
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/stretchr/testify/assert"
)

func TestLocalMetadataRegistry(t *testing.T) {
	registry := metadata.NewLocalMetadataRegistry()

	node1 := &ast.Node{}
	node2 := &ast.Node{}
	node3 := &ast.Node{}

	dirMeta := &metadata.DirectiveMeta{
		Kind:     metadata.MetaKindDirective,
		Ref:      metadata.Reference{Node: node1},
		Selector: "test-dir",
	}

	moduleMeta := &metadata.NgModuleMeta{
		Ref: metadata.Reference{Node: node2},
	}

	pipeMeta := &metadata.PipeMeta{
		Ref:  metadata.Reference{Node: node3},
		Name: "test-pipe",
	}

	registry.RegisterDirective(node1, dirMeta)
	registry.RegisterNgModule(node2, moduleMeta)
	registry.RegisterPipe(node3, pipeMeta)

	assert.Equal(t, dirMeta, registry.GetDirectiveMetadata(node1))
	assert.Equal(t, moduleMeta, registry.GetNgModuleMetadata(node2))
	assert.Equal(t, pipeMeta, registry.GetPipeMetadata(node3))

	// Get non-existent
	assert.Nil(t, registry.GetDirectiveMetadata(&ast.Node{}))
}
