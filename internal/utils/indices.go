package utils

func GetPositiveIndex(size uint, index int64) (int64, error) {
	if index >= 0 {
		if index >= int64(size) {
			return int64(size) - 1, nil
		}
		return index, nil
	}
	if index < -1*int64(size) {
		return 0, nil
	}
	return int64(size) + index, nil
}

func ValidateIndices(start int64, stop int64, size uint) bool {
	start, err := GetPositiveIndex(size, start)
	if err != nil {
		return false
	}

	stop, err = GetPositiveIndex(size, stop)
	if err != nil {
		return false
	}

	if start > stop {
		return false
	}

	return true
}
