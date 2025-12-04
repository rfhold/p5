package ui

import "github.com/rfhold/p5/internal/pulumi"

// Type aliases for pulumi types - allows UI to switch implementations in the future
// by changing these aliases rather than updating all UI code.

// ResourceOp represents a resource operation type (create, update, delete, etc.)
type ResourceOp = pulumi.ResourceOp

// OperationType represents an operation type (up, refresh, destroy)
type OperationType = pulumi.OperationType

// ResourceOp constants - aliased from pulumi package
const (
	OpCreate        = pulumi.OpCreate
	OpUpdate        = pulumi.OpUpdate
	OpDelete        = pulumi.OpDelete
	OpSame          = pulumi.OpSame
	OpReplace       = pulumi.OpReplace
	OpCreateReplace = pulumi.OpCreateReplace
	OpDeleteReplace = pulumi.OpDeleteReplace
	OpRead          = pulumi.OpRead
	OpRefresh       = pulumi.OpRefresh
)

// OperationType constants - aliased from pulumi package
const (
	OperationUp      = pulumi.OperationUp
	OperationRefresh = pulumi.OperationRefresh
	OperationDestroy = pulumi.OperationDestroy
)
