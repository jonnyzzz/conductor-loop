package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/messagebus"
	"github.com/spf13/cobra"
)

func newBusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bus",
		Short: "Read and post messages to the message bus",
	}
	cmd.AddCommand(newBusPostCmd())
	cmd.AddCommand(newBusReadCmd())
	return cmd
}

func newBusPostCmd() *cobra.Command {
	var (
		busPath   string
		msgType   string
		projectID string
		taskID    string
		runID     string
		body      string
	)

	cmd := &cobra.Command{
		Use:   "post",
		Short: "Post a message to the message bus",
		RunE: func(cmd *cobra.Command, args []string) error {
			if busPath == "" {
				return fmt.Errorf("--bus is required")
			}
			if body == "" {
				info, err := os.Stdin.Stat()
				if err == nil && (info.Mode()&os.ModeCharDevice) == 0 {
					data, err := io.ReadAll(os.Stdin)
					if err != nil {
						return fmt.Errorf("read stdin: %w", err)
					}
					body = string(data)
				}
			}
			bus, err := messagebus.NewMessageBus(busPath)
			if err != nil {
				return err
			}
			msg := &messagebus.Message{
				Type:      msgType,
				ProjectID: projectID,
				TaskID:    taskID,
				RunID:     runID,
				Body:      body,
			}
			msgID, err := bus.AppendMessage(msg)
			if err != nil {
				return err
			}
			fmt.Printf("msg_id: %s\n", msgID)
			return nil
		},
	}

	cmd.Flags().StringVar(&busPath, "bus", "", "path to message bus file (required)")
	cmd.Flags().StringVar(&msgType, "type", "INFO", "message type")
	cmd.Flags().StringVar(&projectID, "project", "", "project ID")
	cmd.Flags().StringVar(&taskID, "task", "", "task ID")
	cmd.Flags().StringVar(&runID, "run", "", "run ID")
	cmd.Flags().StringVar(&body, "body", "", "message body (reads from stdin if not provided and stdin is a pipe)")

	return cmd
}

func newBusReadCmd() *cobra.Command {
	var (
		busPath string
		tail    int
		follow  bool
	)

	cmd := &cobra.Command{
		Use:   "read",
		Short: "Read messages from the message bus",
		RunE: func(cmd *cobra.Command, args []string) error {
			if busPath == "" {
				return fmt.Errorf("--bus is required")
			}
			bus, err := messagebus.NewMessageBus(busPath, messagebus.WithPollInterval(500*time.Millisecond))
			if err != nil {
				return err
			}
			messages, err := bus.ReadMessages("")
			if err != nil {
				return err
			}
			if tail > 0 && len(messages) > tail {
				messages = messages[len(messages)-tail:]
			}
			for _, msg := range messages {
				printBusMessage(msg)
			}
			if !follow {
				return nil
			}
			var lastID string
			if len(messages) > 0 {
				lastID = messages[len(messages)-1].MsgID
			}
			for {
				time.Sleep(500 * time.Millisecond)
				newMsgs, err := bus.ReadMessages(lastID)
				if err != nil {
					if errors.Is(err, messagebus.ErrSinceIDNotFound) {
						lastID = ""
						continue
					}
					return err
				}
				for _, msg := range newMsgs {
					printBusMessage(msg)
					lastID = msg.MsgID
				}
			}
		},
	}

	cmd.Flags().StringVar(&busPath, "bus", "", "path to message bus file (required)")
	cmd.Flags().IntVar(&tail, "tail", 20, "print last N messages")
	cmd.Flags().BoolVar(&follow, "follow", false, "watch for new messages (Ctrl-C to exit)")

	return cmd
}

func printBusMessage(msg *messagebus.Message) {
	ts := msg.Timestamp.Format("2006-01-02 15:04:05")
	fmt.Printf("[%s] (%s) %s\n", ts, msg.Type, msg.Body)
}
