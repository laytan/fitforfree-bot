package termux

import (
	"encoding/json"
	"os/exec"
)

// BatteryInfo Note: Comments have the values I have seen so far
type BatteryInfo struct {
	// GOOD
	Health string
	// 0 to 100
	Percentage uint8
	// UNPLUGGED, PLUGGED_AC
	Plugged string
	// DISCHARGING, CHARGING
	Status string
	// Battery temperature
	Temperature float64
	// Current coming in or out of the device can be -
	Current int
}

// Battery returns information about the current battery status
func Battery() (BatteryInfo, error) {
	info := BatteryInfo{}

	cmd := exec.Command("termux-battery-status")
	bytes, err := cmd.Output()
	if err != nil {
		return info, err
	}

	err = json.Unmarshal(bytes, &info)
	return info, err
}
