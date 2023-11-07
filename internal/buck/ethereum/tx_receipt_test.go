package ethereum

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	c "github.com/verity-team/dws/internal/common"
)

type TxReceiptSuite struct {
	suite.Suite
	body []byte
	path string
}

func (suite *TxReceiptSuite) SetupTest() {
	var err error
	suite.body, err = os.ReadFile(suite.path)
	if err != nil {
		suite.Failf("failed to read test input '%s', %v", suite.path, err)
	}
}

func (suite *TxReceiptSuite) TestSuccess() {
	rcpts, err := parseTxReceipt(suite.body)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), 2, len(rcpts))
	rcpt := rcpts[0]
	assert.Equal(suite.T(), "0x44381b", rcpt.BlockNumber)
	assert.Equal(suite.T(), "0x1", rcpt.Status)
	assert.Equal(suite.T(), "0x379738c60f658601be79e267e79cc38cea07c8f2", rcpt.From)
	rcpt = rcpts[1]
	assert.Equal(suite.T(), "0x456d71", rcpt.BlockNumber)
	assert.Equal(suite.T(), "0x1", rcpt.Status)
	assert.Equal(suite.T(), "0x9a6394faa769f066b17bd329a9d5f028719bb0bc", rcpt.From)
}

func TestTxReceiptSuite(t *testing.T) {
	s := new(TxReceiptSuite)
	s.path = "testdata/eth_getTransactionReceipt.json"
	suite.Run(t, s)
}

// MockFetcher is a mock for the Fetcher interface.
type MockFetcher struct {
	mock.Mock
}

func (m *MockFetcher) Fetch(ctxt c.Context, hs []c.Hashable) ([]c.TxByHash, error) {
	args := m.Called(ctxt, hs)
	err := args.Error(1)
	if err != nil {
		return nil, err
	}
	return make([]c.TxByHash, len(hs)), nil
}

// Test types
type TestHashable struct {
	HashValue string
}

func (th TestHashable) GetHash() string {
	return th.HashValue
}

func TestGetData(t *testing.T) {
	// Define your context and hashables
	ctxt := c.Context{}
	hashables := make([]c.Hashable, 130)

	// Define test cases
	tests := []struct {
		name          string
		hashes        []c.Hashable
		mockReturn    []c.TxByHash
		mockError     error
		expectedError bool
	}{
		{
			name:       "EmptyHashables",
			hashes:     []c.Hashable{},
			mockReturn: nil,
			mockError:  nil,
		},
		{
			name:       "SingleBatch",
			hashes:     hashables[:100], // assuming this is less than batchSize
			mockReturn: make([]c.TxByHash, 100),
			mockError:  nil,
		},
		{
			name:       "MultipleBatches",
			hashes:     hashables, // assuming this is more than batchSize
			mockReturn: make([]c.TxByHash, len(hashables)),
			mockError:  nil,
		},
		{
			name:          "FetchError",
			hashes:        hashables,
			mockReturn:    nil,
			mockError:     errors.New("fetch error"),
			expectedError: true,
		},
	}

	for _, tc := range tests {
		// Create a MockFetcher instance
		mockFetcher := new(MockFetcher)

		t.Run(tc.name, func(t *testing.T) {
			// Setup the expected calls to the mock
			mockFetcher.On("Fetch", ctxt, mock.Anything).Return(tc.mockReturn, tc.mockError)

			// Call the GetData function with the test case data
			results, err := GetData[c.TxByHash](ctxt, tc.hashes, mockFetcher)

			// Assert the expectations
			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if len(tc.hashes) > 0 {
					assert.Equal(t, len(tc.mockReturn), len(results), fmt.Sprintf("expected %d results but got %d", len(tc.mockReturn), len(results)))
				} else {
					assert.Nil(t, results, "expected empty result set")
				}
			}

			// Assert that the Fetch was called the correct number of times
			numCalls := len(tc.hashes) / batchSize
			remainder := len(tc.hashes) % batchSize
			if remainder > 0 {
				numCalls++
			}
			if tc.mockError != nil {
				numCalls = 1
			}
			mockFetcher.AssertNumberOfCalls(t, "Fetch", numCalls)
		})
	}
}
