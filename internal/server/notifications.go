package server

import (
	"context"
	"log"

	notificationsv1 "github.com/agynio/egress-rules/.gen/go/agynio/api/notifications/v1"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	notificationSource               = "egress-rules"
	egressRuleUpdatedEvent           = "egress_rule.updated"
	egressRuleAttachmentUpdatedEvent = "egress_rule_attachment.updated"
)

func (s *Server) publishRuleUpdated(ctx context.Context, organizationID uuid.UUID, ruleID uuid.UUID, operation string) {
	payload, err := structpb.NewStruct(map[string]any{
		"organization_id": organizationID.String(),
		"egress_rule_id":  ruleID.String(),
		"operation":       operation,
	})
	if err != nil {
		panic(err)
	}
	if _, err := s.notificationsClient.Publish(ctx, &notificationsv1.PublishRequest{
		Event:   egressRuleUpdatedEvent,
		Rooms:   []string{organizationRoom(organizationID)},
		Payload: payload,
		Source:  notificationSource,
	}); err != nil {
		log.Printf("publish %s: %v", egressRuleUpdatedEvent, err)
	}
}

func (s *Server) publishAttachmentUpdated(ctx context.Context, organizationID uuid.UUID, ruleID uuid.UUID, attachmentID uuid.UUID, agentID uuid.UUID, operation string) {
	payload, err := structpb.NewStruct(map[string]any{
		"organization_id":           organizationID.String(),
		"egress_rule_id":            ruleID.String(),
		"egress_rule_attachment_id": attachmentID.String(),
		"agent_id":                  agentID.String(),
		"operation":                 operation,
	})
	if err != nil {
		panic(err)
	}
	if _, err := s.notificationsClient.Publish(ctx, &notificationsv1.PublishRequest{
		Event:   egressRuleAttachmentUpdatedEvent,
		Rooms:   []string{organizationRoom(organizationID)},
		Payload: payload,
		Source:  notificationSource,
	}); err != nil {
		log.Printf("publish %s: %v", egressRuleAttachmentUpdatedEvent, err)
	}
}

func organizationRoom(organizationID uuid.UUID) string {
	return "organization:" + organizationID.String()
}
