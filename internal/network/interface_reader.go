package network

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
)

func readInterfaceFile(name, field string) (string, error) {
	path := interfacePath(name) + "/" + field

	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(data)), nil
}

func readRequiredString(name, field string) (string, error) {
	value, err := readInterfaceFile(name, field)
	if err != nil {
		return "", fmt.Errorf("read %s/%s: %w", name, field, err)
	}

	if value == "" {
		return "", fmt.Errorf("%s/%s is empty", name, field)
	}

	return value, nil
}

func readRequiredUint(name, field string) (uint64, error) {
	value, err := readRequiredString(name, field)
	if err != nil {
		return 0, err
	}

	n, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse %s/%s value %q: %w", name, field, value, err)
	}

	return n, nil
}

func readOptionalUint(name, field string) (*uint64, error) {
	value, err := readInterfaceFile(name, field)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.EINVAL) {
			return nil, nil
		}

		return nil, fmt.Errorf("read optional %s/%s: %w", name, field, err)
	}

	n, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse %s/%s value %q: %w", name, field, value, err)
	}

	return &n, nil
}

func readOptionalString(name, field string) (*string, error) {
	value, err := readInterfaceFile(name, field)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.EINVAL) {
			return nil, nil
		}

		return nil, fmt.Errorf("read optional %s/%s: %w", name, field, err)
	}

	if value == "" {
		return nil, nil
	}

	return &value, nil
}

func ReadInterface(name string) (Interface, error) {
	hardwareAddr, err := readRequiredString(name, "address")
	if err != nil {
		return Interface{}, err
	}

	operState, err := readRequiredString(name, "operstate")
	if err != nil {
		return Interface{}, err
	}

	carrier, err := readRequiredUint(name, "carrier")
	if err != nil {
		return Interface{}, err
	}

	if carrier != 0 && carrier != 1 {
		return Interface{}, fmt.Errorf("invalid %s/carrier value %d", name, carrier)
	}

	mtu, err := readRequiredUint(name, "mtu")
	if err != nil {
		return Interface{}, err
	}

	speed, err := readOptionalUint(name, "speed")
	if err != nil {
		return Interface{}, err
	}

	duplex, err := readOptionalString(name, "duplex")
	if err != nil {
		return Interface{}, err
	}

	return Interface{
		Name:         name,
		HardwareAddr: hardwareAddr,
		OperState:    operState,
		Carrier:      carrier == 1,
		MTU:          mtu,
		SpeedMbps:    speed,
		Duplex:       duplex,
	}, nil
}
