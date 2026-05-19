package repl

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

func (s *state) evalControlPlaneMessage(ctx context.Context, sub string, args []string, jsonOut bool) error {
	switch sub {
	case "rooms":
		return s.evalMessageRooms(ctx)
	case "room":
		return s.evalMessageRoom(ctx, args, jsonOut)
	case "ls":
		return s.evalMessageLs(ctx, args)
	case "get":
		return s.evalMessageGet(ctx, args)
	case "unread":
		return s.evalMessageUnread(ctx, args)
	case "post":
		return s.evalMessagePost(ctx, args, jsonOut)
	case "dm":
		return s.evalMessageDM(ctx, args, jsonOut)
	case "thread":
		return s.evalMessageThread(ctx, args, jsonOut)
	case "ack":
		return s.evalMessageAck(ctx, args, jsonOut)
	default:
		return fmt.Errorf("unknown command: message %s", sub)
	}
}

func (s *state) evalMessageRooms(ctx context.Context) error {
	body, err := s.fetchJSON(ctx, "/admin/api/message/rooms")
	if err != nil {
		return err
	}
	return writeJSON(s.out, body)
}

func (s *state) evalMessageRoom(ctx context.Context, args []string, jsonOut bool) error {
	plain := nonFlagArgs(args[1:])
	if len(plain) < 1 {
		return errors.New("usage: message room <new|show>")
	}
	action := strings.ToLower(plain[0])
	switch action {
	case "new":
		return s.evalMessageRoomNew(ctx, plain, jsonOut)
	case "show":
		return s.evalMessageRoomShow(ctx, plain)
	default:
		return fmt.Errorf("unknown command: message room %s", action)
	}
}

func (s *state) evalMessageRoomNew(ctx context.Context, plain []string, jsonOut bool) error {
	if len(plain) < 2 {
		return errors.New("usage: message room new <name>")
	}
	body, err := s.postFormJSON(ctx, "/admin/api/message/room", url.Values{"name": {strings.Join(plain[1:], " ")}})
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	roomID := ""
	roomName := strings.Join(plain[1:], " ")
	if roomMap, ok := body["room"].(map[string]any); ok {
		roomID = toString(roomMap["id"])
		if name := toString(roomMap["name"]); name != "" {
			roomName = name
		}
	}
	_, err = fmt.Fprintf(s.out, "OK  room=%s name=%s action=create\n", roomID, roomName)
	return err
}

func (s *state) evalMessageRoomShow(ctx context.Context, plain []string) error {
	if len(plain) < 2 {
		return errors.New("usage: message room show <room>")
	}
	body, err := s.fetchJSONQuery(ctx, "/admin/api/message/room", url.Values{"room": {plain[1]}})
	if err != nil {
		return err
	}
	return writeJSON(s.out, body)
}

func (s *state) evalMessageLs(ctx context.Context, args []string) error {
	query := url.Values{}
	plain := nonFlagArgs(args[1:])
	if len(plain) > 0 {
		query.Set("room", plain[0])
	}
	body, err := s.fetchJSONQuery(ctx, "/admin/api/message", query)
	if err != nil {
		return err
	}
	return writeJSON(s.out, body)
}

func (s *state) evalMessageGet(ctx context.Context, args []string) error {
	plain := nonFlagArgs(args[1:])
	if len(plain) < 1 {
		return errors.New("usage: message get <message>")
	}
	body, err := s.fetchJSONQuery(ctx, "/admin/api/message/get", url.Values{"message_id": {plain[0]}})
	if err != nil {
		return err
	}
	return writeJSON(s.out, body)
}

func (s *state) evalMessageUnread(ctx context.Context, args []string) error {
	plain := nonFlagArgs(args[1:])
	if len(plain) < 1 {
		return errors.New("usage: message unread <identity> [room]")
	}
	query := url.Values{"identity_id": {plain[0]}}
	if len(plain) > 1 {
		query.Set("room", plain[1])
	}
	body, err := s.fetchJSONQuery(ctx, "/admin/api/message/unread", query)
	if err != nil {
		return err
	}
	return writeJSON(s.out, body)
}

func (s *state) evalMessagePost(ctx context.Context, args []string, jsonOut bool) error {
	plain := nonFlagArgs(args[1:])
	if len(plain) < 2 {
		return errors.New("usage: message post <room> <text>")
	}
	body, err := s.postFormJSON(ctx, "/admin/api/message/post", url.Values{
		"room": {plain[0]},
		"text": {strings.Join(plain[1:], " ")},
	})
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	messageID := ""
	if msgMap, ok := body["message"].(map[string]any); ok {
		messageID = toString(msgMap["id"])
	}
	_, err = fmt.Fprintf(s.out, "OK  message=%s\n", messageID)
	return err
}

func (s *state) evalMessageDM(ctx context.Context, args []string, jsonOut bool) error {
	plain := nonFlagArgs(args[1:])
	if len(plain) < 2 {
		return errors.New("usage: message dm <target> <text>")
	}
	body, err := s.postFormJSON(ctx, "/admin/api/message/dm", url.Values{
		"target_ref": {plain[0]},
		"text":       {strings.Join(plain[1:], " ")},
	})
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	messageID := ""
	if msgMap, ok := body["message"].(map[string]any); ok {
		messageID = toString(msgMap["id"])
	}
	_, err = fmt.Fprintf(s.out, "OK  message=%s target=%s action=dm\n", messageID, plain[0])
	return err
}

func (s *state) evalMessageThread(ctx context.Context, args []string, jsonOut bool) error {
	plain := nonFlagArgs(args[1:])
	if len(plain) < 2 {
		return errors.New("usage: message thread <root-message> <text>")
	}
	body, err := s.postFormJSON(ctx, "/admin/api/message/thread", url.Values{
		"root_ref": {plain[0]},
		"text":     {strings.Join(plain[1:], " ")},
	})
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	messageID := ""
	if msgMap, ok := body["message"].(map[string]any); ok {
		messageID = toString(msgMap["id"])
	}
	_, err = fmt.Fprintf(s.out, "OK  message=%s root=%s action=thread\n", messageID, plain[0])
	return err
}

func (s *state) evalMessageAck(ctx context.Context, args []string, jsonOut bool) error {
	plain := nonFlagArgs(args[1:])
	if len(plain) < 2 {
		return errors.New("usage: message ack <identity> <message>")
	}
	body, err := s.postFormJSON(ctx, "/admin/api/message/ack", url.Values{
		"identity_id": {plain[0]},
		"message_id":  {plain[1]},
	})
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	_, err = fmt.Fprintf(s.out, "OK  identity=%s message=%s action=ack\n", plain[0], plain[1])
	return err
}
