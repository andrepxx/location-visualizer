package math

import (
	"fmt"
	"strconv"
	"strings"
)

/*
 * Parse 32-bit fixed-point number.
 */
func ParseFixed32(number string, decimalPlaces uint8) (int32, error) {
	numberTrimmed := strings.TrimSpace(number)
	integerPartString, fractionalPartString, hasFractionalPart := strings.Cut(numberTrimmed, ".")
	negativeNumber := strings.HasPrefix(integerPartString, "-")
	value, err := strconv.ParseInt(integerPartString, 10, 32)

	/*
	 * Check if we could parse the integer part of the number.
	 */
	if err != nil {
		return 0, fmt.Errorf("%s", "Parse error")
	} else {

		/*
		 * Shift value by the required number of decimal places.
		 */
		for i := uint8(0); i < decimalPlaces; i++ {
			value *= 10
		}

		/*
		 * Handle fractional part, if present.
		 */
		if hasFractionalPart {
			lenFractionalPart := len(fractionalPartString)
			decimalPlacesInt := int(decimalPlaces)

			/*
			 * If fractional part is longer than number of decimal places, trim it.
			 */
			if lenFractionalPart > decimalPlacesInt {
				fractionalPartString = fractionalPartString[:decimalPlacesInt]
				lenFractionalPart = decimalPlacesInt
			}

			fractionalPart, err := strconv.ParseUint(fractionalPartString, 10, 32)

			/*
			 * Check if we could parse the fractional part of the number.
			 */
			if err != nil {
				return 0, fmt.Errorf("%s", "Parse error")
			} else {

				/*
				 * Shift the fractional part in case it's too short.
				 */
				for i := lenFractionalPart; i < decimalPlacesInt; i++ {
					fractionalPart *= 10
				}

				fractionalPartSigned := int64(fractionalPart)

				/*
				 * Subtract or add fractional part from or to value.
				 */
				if negativeNumber {
					value -= fractionalPartSigned
				} else {
					value += fractionalPartSigned
				}

			}

		}

		result := int32(value)
		return result, nil
	}

}
