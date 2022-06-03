package datamarshaler

import "fmt"

func IsBankBalanceKey(key []byte) bool {
	return isBankBalanceKey(key)
}

func IsCW20BalanceKey(key []byte) bool {
	return isCW20BalanceKey(key)
}

func IsCW20TokenInfoKey(key []byte) bool {
	return isCW20TokenInfoKey(key)
}

// fromLengthPrefix returns the amount of data signaled by the single-byte length prefix in rawData.
func fromLengthPrefix(rawData []byte) ([]byte, error) {
	if len(rawData) == 0 {
		return nil, fmt.Errorf("data is nil")
	}

	length := int(rawData[0])
	rawData = rawData[1:]
	if len(rawData) != length {
		return nil, fmt.Errorf("length prefix signals %d bytes, but total data is %d bytes long", length, len(rawData))
	}

	return rawData, nil
}
