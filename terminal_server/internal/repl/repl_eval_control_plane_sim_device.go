package repl

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

func (s *state) evalSimDeviceNew(ctx context.Context, args []string, jsonOut bool) error {
	plain := nonFlagArgsSkippingFlagValues(args, "--caps")
	if len(plain) < 1 {
		return errors.New("usage: sim device new <id> [--caps <cap[,cap...]>]")
	}
	form := url.Values{"device_id": {plain[0]}}
	if capsRaw := strings.TrimSpace(flagValue(args, "--caps")); capsRaw != "" {
		for _, capValue := range parseCSVValues(capsRaw) {
			form.Add("caps", capValue)
		}
	}
	body, err := s.postFormJSON(ctx, "/admin/api/sim/devices/new", form)
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	_, err = fmt.Fprintf(s.out, "OK  action=sim.device.new device=%s\n", plain[0])
	return err
}

func (s *state) evalSimDeviceRm(ctx context.Context, args []string, jsonOut bool) error {
	plain := nonFlagArgs(args)
	if len(plain) < 1 {
		return errors.New("usage: sim device rm <id>")
	}
	body, err := s.postFormJSON(ctx, "/admin/api/sim/devices/rm", url.Values{"device_id": {plain[0]}})
	if err != nil {
		return err
	}
	if jsonOut {
		return writeJSON(s.out, body)
	}
	_, err = fmt.Fprintf(s.out, "OK  action=sim.device.rm device=%s deleted=%s\n", plain[0], toString(body["deleted"]))
	return err
}
