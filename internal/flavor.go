package internal

import (
	"encoding/binary"
	"errors"
	"fmt"
)

var (
	ErrCorruptedFlavorData = errors.New("failed to decode flavor selector")
)

func DecodeFlavorPick(data []byte, opt DecodeOptions) (int, error) {
	selector, sz := binary.Uvarint(data)
	if sz <= 0 {
		return 0, ErrCorruptedFlavorData
	}
	data = data[sz:]
	target, ok := opt.FlavorSelectors[uint(selector)]
	if !ok {
		return 0, fmt.Errorf("data tried to look at flavor selector %d", selector)
	}
	numCases, sz := binary.Uvarint(data)
	if sz <= 0 {
		return 0, ErrCorruptedFlavorData
	}
	data = data[sz:]
	hasElse := numCases&1 == 1
	numCases >>= 1
	for numCases > 0 {
		n, sz := binary.Uvarint(data)
		if sz <= 0 {
			return 0, ErrCorruptedFlavorData
		}
		data = data[sz:]
		if uint(n) == target {
			return decodeFlavorJump(data)
		}
		// Skip over jump target efficiently
		for {
			if len(data) == 0 {
				return 0, ErrCorruptedFlavorData
			}
			if data[0] < 128 {
				data = data[1:]
				break
			}
			data = data[1:]
		}
		numCases--
	}
	if !hasElse {
		return 0, fmt.Errorf("data didn't have a match for flavor %d value %d", selector, target)
	}
	return decodeFlavorJump(data)
}

func decodeFlavorJump(data []byte) (int, error) {
	n, sz := binary.Uvarint(data)
	if sz <= 0 {
		return 0, ErrCorruptedFlavorData
	}
	return int(n), nil
}
