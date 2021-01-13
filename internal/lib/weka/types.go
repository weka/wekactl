package weka

import (
	"bytes"
	"fmt"
	"github.com/rs/zerolog/log"
	"strconv"
)

const hostIdPrefix = "HostId<"
const driveIdPrefix = "DiskId<"

type HostId struct {
	hostId     int
	wekaHostId string
}

func (h HostId) String() string {
	return h.wekaHostId
}

func (h *HostId) MarshalText() ([]byte, error) {
	return []byte(h.wekaHostId), nil
}

func (h *HostId) UnmarshalText(bytes []byte) error {
	hid, err := unmarshalPrefixedID([]byte(hostIdPrefix), bytes)
	h.wekaHostId = string(bytes)
	if err != nil {
		log.Error().Err(err)
		return err
	}
	h.hostId = hid
	return nil
}

type DriveId struct {
	driveId     int
	wekaDriveId string
}

func (h DriveId) String() string {
	return h.wekaDriveId
}

func (h *DriveId) MarshalText() ([]byte, error) {
	return []byte(h.wekaDriveId), nil
}

func (h *DriveId) UnmarshalText(bytes []byte) error {
	hid, err := unmarshalPrefixedID([]byte(driveIdPrefix), bytes)
	h.wekaDriveId = string(bytes)
	if err != nil {
		log.Error().Err(err)
		return err
	}
	h.driveId = hid
	return nil
}

func unmarshalPrefixedID(prefix, text []byte) (int, error) {
	unmarshalErr := fmt.Errorf("can't extract %q value from %q", string(prefix[:len(prefix)-1]), string(text))
	if !bytes.HasPrefix(text, prefix) {
		return -1, unmarshalErr
	}
	end := bytes.LastIndexByte(text, '>')
	if end < len(prefix) {
		return -1, unmarshalErr
	}
	val := string(text[len(prefix):end])
	if val == "INVALID" {
		return -1, nil
	}
	return strconv.Atoi(val)
}