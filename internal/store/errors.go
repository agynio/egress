package store

import "errors"

var (
	ErrRuleNotFound       = errors.New("egress rule not found")
	ErrRuleAlreadyExists  = errors.New("egress rule already exists")
	ErrAttachmentNotFound = errors.New("egress rule attachment not found")
	ErrAttachmentExists   = errors.New("egress rule attachment already exists")
	ErrRuleHasAttachments = errors.New("egress rule has attachments")
)
