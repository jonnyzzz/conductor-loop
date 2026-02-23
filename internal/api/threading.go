package api

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/messagebus"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

const (
	threadLinkFileName            = "TASK-THREAD-LINK.yaml"
	threadMessageTypeUserRequest  = "USER_REQUEST"
	threadParentTypeQuestion      = "QUESTION"
	threadParentTypeFact          = "FACT"
	threadMetaParentProjectIDKey  = "thread_parent_project_id"
	threadMetaParentTaskIDKey     = "thread_parent_task_id"
	threadMetaParentRunIDKey      = "thread_parent_run_id"
	threadMetaParentMessageIDKey  = "thread_parent_message_id"
	threadMetaParentTypeKey       = "thread_parent_message_type"
	threadMetaChildProjectIDKey   = "thread_child_project_id"
	threadMetaChildTaskIDKey      = "thread_child_task_id"
	threadMetaChildRunIDKey       = "thread_child_run_id"
	threadMetaChildMessageIDKey   = "thread_child_message_id"
	threadMetaSourceMessageIDKey  = "thread_source_message_id"
	threadMetaDirectionKey        = "thread_direction"
	threadDirectionChildTask      = "child_task"
	threadDirectionSourceTask     = "source_task"
	threadParentLinkKind          = "thread_answer"
	threadParentProjectMetaKey    = "project_id"
	threadParentTaskMetaKey       = "task_id"
	threadParentRunMetaKey        = "run_id"
	threadSupportedParentTypeList = "QUESTION,FACT"
)

var threadSupportedParentTypes = map[string]struct{}{
	threadParentTypeQuestion: {},
	threadParentTypeFact:     {},
}

// ThreadParentReference identifies the source message a threaded task answers.
type ThreadParentReference struct {
	ProjectID   string `json:"project_id" yaml:"parent_project_id"`
	TaskID      string `json:"task_id" yaml:"parent_task_id"`
	RunID       string `json:"run_id" yaml:"parent_run_id"`
	MessageID   string `json:"message_id" yaml:"parent_message_id"`
	MessageType string `json:"message_type,omitempty" yaml:"parent_message_type,omitempty"`
}

type taskThreadLinkFile struct {
	ParentProjectID string    `yaml:"parent_project_id"`
	ParentTaskID    string    `yaml:"parent_task_id"`
	ParentRunID     string    `yaml:"parent_run_id"`
	ParentMessageID string    `yaml:"parent_message_id"`
	ParentType      string    `yaml:"parent_message_type,omitempty"`
	LinkedAt        time.Time `yaml:"linked_at"`
}

type threadedParentContext struct {
	Parent  ThreadParentReference
	Message *messagebus.Message
}

func normalizeThreadMessageType(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return threadMessageTypeUserRequest
	}
	return value
}

func normalizeThreadParent(parent *ThreadParentReference) ThreadParentReference {
	if parent == nil {
		return ThreadParentReference{}
	}
	return ThreadParentReference{
		ProjectID: strings.TrimSpace(parent.ProjectID),
		TaskID:    strings.TrimSpace(parent.TaskID),
		RunID:     strings.TrimSpace(parent.RunID),
		MessageID: strings.TrimSpace(parent.MessageID),
	}
}

func (s *Server) validateThreadedParent(req *TaskCreateRequest) (*threadedParentContext, *apiError) {
	if req == nil {
		return nil, apiErrorInternal("task create request is nil", nil)
	}
	threadMessageType := strings.TrimSpace(req.ThreadMessageType)

	if req.ThreadParent == nil {
		if threadMessageType != "" {
			return nil, apiErrorBadRequest("thread_message_type requires thread_parent")
		}
		return nil, nil
	}

	req.ThreadMessageType = normalizeThreadMessageType(threadMessageType)

	if req.ThreadMessageType != threadMessageTypeUserRequest {
		return nil, apiErrorBadRequest("thread_message_type must be USER_REQUEST")
	}

	parent := normalizeThreadParent(req.ThreadParent)
	if err := validateIdentifier(parent.ProjectID, "thread_parent.project_id"); err != nil {
		return nil, err
	}
	if err := validateIdentifier(parent.TaskID, "thread_parent.task_id"); err != nil {
		return nil, err
	}
	if err := validateIdentifier(parent.RunID, "thread_parent.run_id"); err != nil {
		return nil, err
	}
	if err := validateIdentifier(parent.MessageID, "thread_parent.message_id"); err != nil {
		return nil, err
	}

	parentTaskDir, ok := findProjectTaskDir(s.rootDir, parent.ProjectID, parent.TaskID)
	if !ok {
		return nil, apiErrorNotFound("parent task not found")
	}
	parentBusPath := filepath.Join(parentTaskDir, "TASK-MESSAGE-BUS.md")
	parentBus, err := messagebus.NewMessageBus(parentBusPath)
	if err != nil {
		return nil, apiErrorInternal("open parent message bus", err)
	}
	parentMessages, err := parentBus.ReadMessages("")
	if err != nil {
		return nil, apiErrorInternal("read parent message bus", err)
	}

	var parentMessage *messagebus.Message
	for _, msg := range parentMessages {
		if msg != nil && msg.MsgID == parent.MessageID {
			parentMessage = msg
			break
		}
	}
	if parentMessage == nil {
		return nil, apiErrorNotFound("parent message not found")
	}

	if parentMessage.ProjectID != parent.ProjectID || parentMessage.TaskID != parent.TaskID {
		return nil, apiErrorBadRequest("invalid parent reference: parent message does not belong to parent project/task")
	}
	if strings.TrimSpace(parentMessage.RunID) == "" {
		return nil, apiErrorBadRequest("invalid parent reference: source message has empty run_id")
	}
	if parentMessage.RunID != parent.RunID {
		return nil, apiErrorBadRequest(fmt.Sprintf("invalid parent reference: parent run_id mismatch (expected %s)", parentMessage.RunID))
	}

	parentType := strings.TrimSpace(parentMessage.Type)
	if _, ok := threadSupportedParentTypes[parentType]; !ok {
		return nil, apiErrorConflict(fmt.Sprintf("parent message type %q is not supported", parentType), map[string]string{
			"allowed_parent_types": threadSupportedParentTypeList,
		})
	}
	parent.MessageType = parentType
	req.ThreadParent = &parent

	return &threadedParentContext{
		Parent:  parent,
		Message: parentMessage,
	}, nil
}

func writeTaskThreadLink(taskDir string, parent ThreadParentReference) error {
	if strings.TrimSpace(taskDir) == "" {
		return errors.New("task dir is empty")
	}
	link := taskThreadLinkFile{
		ParentProjectID: parent.ProjectID,
		ParentTaskID:    parent.TaskID,
		ParentRunID:     parent.RunID,
		ParentMessageID: parent.MessageID,
		ParentType:      parent.MessageType,
		LinkedAt:        time.Now().UTC(),
	}
	data, err := yaml.Marshal(link)
	if err != nil {
		return errors.Wrap(err, "encode thread link")
	}
	path := filepath.Join(taskDir, threadLinkFileName)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return errors.Wrap(err, "write thread link")
	}
	return nil
}

func readTaskThreadLink(taskDir string) (*ThreadParentReference, error) {
	if strings.TrimSpace(taskDir) == "" {
		return nil, errors.New("task dir is empty")
	}
	path := filepath.Join(taskDir, threadLinkFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "read thread link")
	}

	var link taskThreadLinkFile
	if err := yaml.Unmarshal(data, &link); err != nil {
		return nil, errors.Wrap(err, "decode thread link")
	}
	if strings.TrimSpace(link.ParentProjectID) == "" ||
		strings.TrimSpace(link.ParentTaskID) == "" ||
		strings.TrimSpace(link.ParentRunID) == "" ||
		strings.TrimSpace(link.ParentMessageID) == "" {
		return nil, errors.New("thread link is missing required parent fields")
	}
	return &ThreadParentReference{
		ProjectID:   strings.TrimSpace(link.ParentProjectID),
		TaskID:      strings.TrimSpace(link.ParentTaskID),
		RunID:       strings.TrimSpace(link.ParentRunID),
		MessageID:   strings.TrimSpace(link.ParentMessageID),
		MessageType: strings.TrimSpace(link.ParentType),
	}, nil
}

func threadBaseMeta(parent ThreadParentReference, childProjectID, childTaskID, childRunID string) map[string]string {
	return map[string]string{
		threadMetaParentProjectIDKey: parent.ProjectID,
		threadMetaParentTaskIDKey:    parent.TaskID,
		threadMetaParentRunIDKey:     parent.RunID,
		threadMetaParentMessageIDKey: parent.MessageID,
		threadMetaParentTypeKey:      parent.MessageType,
		threadMetaChildProjectIDKey:  childProjectID,
		threadMetaChildTaskIDKey:     childTaskID,
		threadMetaChildRunIDKey:      childRunID,
	}
}

func copyStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func appendThreadedMessage(busPath string, msg *messagebus.Message) (string, error) {
	bus, err := messagebus.NewMessageBus(busPath)
	if err != nil {
		return "", errors.Wrap(err, "open message bus")
	}
	msgID, err := bus.AppendMessage(msg)
	if err != nil {
		return "", errors.Wrap(err, "append message")
	}
	return msgID, nil
}

func (s *Server) persistThreadedTaskLinkage(taskDir string, req TaskCreateRequest, runID string, parent *threadedParentContext) error {
	if parent == nil {
		return nil
	}
	if err := writeTaskThreadLink(taskDir, parent.Parent); err != nil {
		return errors.Wrap(err, "persist task thread link")
	}

	baseMeta := threadBaseMeta(parent.Parent, req.ProjectID, req.TaskID, runID)
	childMeta := copyStringMap(baseMeta)
	childMeta[threadMetaDirectionKey] = threadDirectionChildTask

	childParents := []messagebus.Parent{{
		MsgID: parent.Parent.MessageID,
		Kind:  threadParentLinkKind,
		Meta: map[string]string{
			threadParentProjectMetaKey: parent.Parent.ProjectID,
			threadParentTaskMetaKey:    parent.Parent.TaskID,
			threadParentRunMetaKey:     parent.Parent.RunID,
		},
	}}

	childBusPath := filepath.Join(taskDir, "TASK-MESSAGE-BUS.md")
	childMsgID, err := appendThreadedMessage(childBusPath, &messagebus.Message{
		Type:      req.ThreadMessageType,
		ProjectID: req.ProjectID,
		TaskID:    req.TaskID,
		RunID:     runID,
		Parents:   childParents,
		Meta:      childMeta,
		Body:      req.Prompt,
	})
	if err != nil {
		return errors.Wrap(err, "append threaded message to child task bus")
	}

	parentTaskDir, ok := findProjectTaskDir(s.rootDir, parent.Parent.ProjectID, parent.Parent.TaskID)
	if !ok {
		return errors.New("parent task disappeared while creating threaded linkage")
	}
	sourceMeta := copyStringMap(baseMeta)
	sourceMeta[threadMetaChildMessageIDKey] = childMsgID
	sourceMeta[threadMetaSourceMessageIDKey] = parent.Parent.MessageID
	sourceMeta[threadMetaDirectionKey] = threadDirectionSourceTask

	sourceBody := fmt.Sprintf("threaded user request opened child task %s/%s", req.ProjectID, req.TaskID)
	sourceBusPath := filepath.Join(parentTaskDir, "TASK-MESSAGE-BUS.md")
	if _, err := appendThreadedMessage(sourceBusPath, &messagebus.Message{
		Type:      req.ThreadMessageType,
		ProjectID: parent.Parent.ProjectID,
		TaskID:    parent.Parent.TaskID,
		RunID:     parent.Parent.RunID,
		Parents:   childParents,
		Meta:      sourceMeta,
		Body:      sourceBody,
	}); err != nil {
		return errors.Wrap(err, "append threaded message to source task bus")
	}

	return nil
}
