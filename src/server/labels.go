package server

import (
	"context"
	"fmt"
	"github.com/devlikeapro/gows/storage"
	"go.mau.fi/whatsmeow/appstate"
	"go.mau.fi/whatsmeow/proto/waSyncAction"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"

	__ "github.com/devlikeapro/gows/proto"
)

// GetLabels implements the gRPC method to retrieve all labels for a session
func (s *Server) GetLabels(ctx context.Context, req *__.GetLabelsRequest) (*__.JsonList, error) {
	cli, err := s.Sm.Get(req.Session.Id)
	if err != nil {
		return nil, err
	}

	labels, err := cli.Storage.Labels.GetAllLabels()
	if err != nil {
		return nil, fmt.Errorf("error getting labels: %w", err)
	}

	response, err := toJsonList(labels)
	if err != nil {
		return nil, fmt.Errorf("error marshaling labels: %w", err)
	}

	return response, nil
}

// buildListEdit is appstate.BuildLabelEdit plus the fields WhatsApp now needs
// for labels to show up as "Lists": type and isActive (WA Web's labelAddAction
// sends type=CUSTOM, isActive=true). Without them the phone shows a legacy label.
// Edits keep the stored type/isActive/predefinedId so built-in lists stay intact.
func buildListEdit(id string, name string, color int32, existing *storage.Label) appstate.PatchInfo {
	listType := waSyncAction.LabelEditAction_CUSTOM
	isActive := true
	var predefinedID *int32
	if existing != nil {
		if existing.Type != 0 {
			listType = waSyncAction.LabelEditAction_ListType(existing.Type)
		}
		if existing.IsActive != nil {
			isActive = *existing.IsActive
		}
		if existing.PredefinedID != 0 {
			predefinedID = proto.Int32(existing.PredefinedID)
		}
	}
	return appstate.PatchInfo{
		Type: appstate.WAPatchRegular,
		Mutations: []appstate.MutationInfo{{
			Index:   []string{appstate.IndexLabelEdit, id},
			Version: 3,
			Value: &waSyncAction.SyncActionValue{
				LabelEditAction: &waSyncAction.LabelEditAction{
					Name:         proto.String(name),
					Color:        proto.Int32(color),
					Deleted:      proto.Bool(false),
					IsActive:     proto.Bool(isActive),
					Type:         listType.Enum(),
					PredefinedID: predefinedID,
				},
			},
		}},
	}
}

// UpsertLabel implements the gRPC method to create or update a label
func (s *Server) UpsertLabel(ctx context.Context, req *__.UpsertLabelRequest) (*__.Empty, error) {
	cli, err := s.Sm.Get(req.Session.Id)
	if err != nil {
		return nil, err
	}

	// Send app state update for label edit
	label := req.Label
	existing, _ := cli.Storage.Labels.GetLabelById(label.Id)
	patch := buildListEdit(label.Id, label.Name, label.Color, existing)
	err = cli.SendAppState(ctx, patch)
	if err != nil {
		return nil, fmt.Errorf("error upserting label: %w", err)
	}
	return &__.Empty{}, nil
}

// DeleteLabel implements the gRPC method to delete a label
func (s *Server) DeleteLabel(ctx context.Context, req *__.DeleteLabelRequest) (*__.Empty, error) {
	cli, err := s.Sm.Get(req.Session.Id)
	if err != nil {
		return nil, err
	}

	// Send app state update for label delete
	label := req.Label
	patch := appstate.BuildLabelEdit(label.Id, label.Name, label.Color, true)
	err = cli.SendAppState(ctx, patch)
	if err != nil {
		return nil, fmt.Errorf("error deleting label: %w", err)
	}
	return &__.Empty{}, nil
}

func (s *Server) GetLabelsByJid(ctx context.Context, req *__.EntityByIdRequest) (*__.JsonList, error) {
	cli, err := s.Sm.Get(req.Session.Id)
	if err != nil {
		return nil, err
	}
	jid, err := types.ParseJID(req.Id)
	if err != nil {
		return nil, fmt.Errorf("error parsing jid: %w", err)
	}
	labelIds, err := cli.Storage.LabelAssociations.GetLabelIDsByJID(jid)
	if err != nil {
		return nil, fmt.Errorf("error getting labels by jid: %w", err)
	}
	labels, err := cli.Storage.Labels.GetAllLabels()
	labelById := make(map[string]*storage.Label)
	for _, label := range labels {
		labelById[label.ID] = label
	}
	labelsForJid := make([]*storage.Label, 0)
	for _, labelId := range labelIds {
		labelsForJid = append(labelsForJid, labelById[labelId])
	}
	response, err := toJsonList(labelsForJid)
	if err != nil {
		return nil, fmt.Errorf("error marshaling labels: %w", err)
	}
	return response, nil
}
func (s *Server) GetChatsByLabelId(ctx context.Context, req *__.EntityByIdRequest) (*__.JsonList, error) {
	cli, err := s.Sm.Get(req.Session.Id)
	if err != nil {
		return nil, err
	}
	labelId := req.Id
	chats, err := cli.Storage.LabelAssociations.GetJIDsByLabelID(labelId)
	if err != nil {
		return nil, fmt.Errorf("error getting chats by label id: %w", err)
	}
	response, err := toJsonList(chats)
	if err != nil {
		return nil, fmt.Errorf("error marshaling chats: %w", err)
	}
	return response, nil

}

func (s *Server) AddChatLabel(ctx context.Context, req *__.ChatLabelRequest) (*__.Empty, error) {
	cli, err := s.Sm.Get(req.Session.Id)
	if err != nil {
		return nil, err
	}
	jid, err := types.ParseJID(req.ChatId)
	patch := appstate.BuildLabelChat(jid, req.LabelId, true)
	err = cli.SendAppState(ctx, patch)
	if err != nil {
		return nil, fmt.Errorf("error adding label to chat: %w", err)
	}
	return &__.Empty{}, nil
}

func (s *Server) RemoveChatLabel(ctx context.Context, req *__.ChatLabelRequest) (*__.Empty, error) {
	cli, err := s.Sm.Get(req.Session.Id)
	if err != nil {
		return nil, err
	}
	jid, err := types.ParseJID(req.ChatId)
	patch := appstate.BuildLabelChat(jid, req.LabelId, false)
	err = cli.SendAppState(ctx, patch)
	if err != nil {
		return nil, fmt.Errorf("error removing label from chat: %w", err)
	}
	return &__.Empty{}, nil
}
