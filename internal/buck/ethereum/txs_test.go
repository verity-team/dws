package ethereum

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestParseInputData(t *testing.T) {
	tests := []struct {
		input          string
		expectedAddr   string
		expectedAmount decimal.Decimal
		expectedError  string
	}{
		// Valid input case
		{
			input:          "0xa9059cbb000000000000000000000000865a1f30b979e4bf3ab30562daee05f917ec052700000000000000000000000000000000000000000000000000000000000be293",
			expectedAddr:   "0x865a1f30b979e4bf3ab30562daee05f917ec0527",
			expectedAmount: decimal.NewFromInt32(778899),
			expectedError:  "",
		},
		// Input with invalid length
		{
			input:          "0xa9059cbb000000000000000000000000865a1f30b979e4bf3ab30562daee05f917ec0527000000000000000000000000000000000000000000000000de0b6b3a7640",
			expectedAddr:   "",
			expectedAmount: decimal.Zero,
			expectedError:  "input has invalid length",
		},
		// Input with incorrect function signature
		{
			input:          "0x12345678000000000000000000000000865a1f30b979e4bf3ab30562daee05f917ec0527000000000000000000000000000000000000000000000000de0b6b3a76400000",
			expectedAddr:   "",
			expectedAmount: decimal.Zero,
			expectedError:  "input does not start with the expected function signature",
		},
		// Input with invalid amount
		{
			input:          "0xa9059cbb000000000000000000000000865a1f30b979e4bf3ab30562daee05f917ec0527000000000000000000000000000000000000000000000000invalidamount000",
			expectedAddr:   "",
			expectedAmount: decimal.Zero,
			expectedError:  "failed to convert amount to uint64",
		},
	}

	for _, test := range tests {
		addr, amount, err := parseInputData(test.input)

		// Check the address, amount, and error
		assert.Equal(t, test.expectedAddr, addr, "Address mismatch")
		assert.Equal(t, test.expectedAmount, amount, "Amount mismatch")

		if test.expectedError != "" {
			assert.NotNil(t, err, "Expected an error")
			assert.Equal(t, test.expectedError, err.Error(), "Error message mismatch")
		} else {
			assert.Nil(t, err, "Expected no error")
		}
	}
}
