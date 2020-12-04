package termux

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
)

// Volume has info off a specific stream
type Volume struct {
	// call, system, ring, music, alarm, notification
	Volume    uint8
	MaxVolume uint8
}

type termuxVolumeRes struct {
	Stream    string
	Volume    uint8
	MaxVolume uint8
}

// VolumeInfo returns the devices volume streams
func VolumeInfo() (map[string]Volume, error) {
	streams := make(map[string]Volume)

	cmd := exec.Command("termux-volume")
	output, err := cmd.Output()
	if err != nil {
		return streams, err
	}

	res := make([]termuxVolumeRes, 0)
	if err := json.Unmarshal(output, &res); err != nil {
		return streams, err
	}

	for _, stream := range res {
		streams[stream.Stream] = Volume{Volume: stream.Volume, MaxVolume: stream.MaxVolume}
	}

	return streams, nil
}

// VolumeOf returns the volume info of a specific stream
func VolumeOf(stream string) (Volume, error) {
	info, err := VolumeInfo()
	if err != nil {
		return Volume{}, err
	}

	vol, exists := info[stream]
	if !exists {
		return Volume{}, errors.New("stream does not exist")
	}

	return vol, nil
}

// SetVolume sets the volume of the given stream
func SetVolume(stream string, vol uint8) error {
	cmd := exec.Command("termux-volume", stream, fmt.Sprintf("%d", vol))
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}
