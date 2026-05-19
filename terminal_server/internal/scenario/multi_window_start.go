package scenario

import (
	"errors"
	"strings"

	iorouter "github.com/curtcox/terminals/terminal_server/internal/io"
)

func connectMultiWindowVideoRoutes(env *Environment, source string, peers []string) error {
	for _, peer := range peers {
		if err := env.IO.Connect(peer, source, "video"); err != nil && !errors.Is(err, iorouter.ErrRouteExists) {
			return err
		}
	}
	return nil
}

func connectMultiWindowAudioRoutes(env *Environment, source string, peers []string, focusDeviceID string) error {
	if err := clearMultiWindowAudioRoutes(env, source); err != nil {
		return err
	}
	if focused, err := connectMultiWindowFocusedAudio(env, source, peers, focusDeviceID); err != nil || focused {
		return err
	}
	return connectMultiWindowMixedAudio(env, source, peers)
}

func connectMultiWindowFocusedAudio(env *Environment, source string, peers []string, focusDeviceID string) (bool, error) {
	focusedPeer := strings.TrimSpace(focusDeviceID)
	if focusedPeer == "" {
		return false, nil
	}
	for _, peer := range peers {
		if peer != focusedPeer {
			continue
		}
		if err := env.IO.Connect(peer, source, "audio"); err != nil && !errors.Is(err, iorouter.ErrRouteExists) {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func connectMultiWindowMixedAudio(env *Environment, source string, peers []string) error {
	for _, peer := range peers {
		if err := env.IO.Connect(peer, source, "audio_mix"); err != nil && !errors.Is(err, iorouter.ErrRouteExists) {
			return err
		}
	}
	return nil
}
