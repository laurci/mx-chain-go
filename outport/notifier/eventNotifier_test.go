package notifier_test

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/multiversx/mx-chain-core-go/data"
	"github.com/multiversx/mx-chain-core-go/data/block"
	"github.com/multiversx/mx-chain-core-go/data/outport"
	outportSenderData "github.com/multiversx/mx-chain-core-go/websocketOutportDriver/data"
	"github.com/multiversx/mx-chain-go/outport/mock"
	"github.com/multiversx/mx-chain-go/outport/notifier"
	"github.com/multiversx/mx-chain-go/testscommon"
	"github.com/multiversx/mx-chain-go/testscommon/hashingMocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createMockEventNotifierArgs() notifier.ArgsEventNotifier {
	return notifier.ArgsEventNotifier{
		HttpClient:      &mock.HTTPClientStub{},
		Marshaller:      &testscommon.MarshalizerMock{},
		Hasher:          &hashingMocks.HasherMock{},
		PubKeyConverter: &testscommon.PubkeyConverterMock{},
	}
}

func TestNewEventNotifier(t *testing.T) {
	t.Parallel()

	t.Run("nil http client", func(t *testing.T) {
		t.Parallel()

		args := createMockEventNotifierArgs()
		args.HttpClient = nil

		en, err := notifier.NewEventNotifier(args)
		require.Nil(t, en)
		require.Equal(t, notifier.ErrNilHTTPClientWrapper, err)
	})

	t.Run("nil marshaller", func(t *testing.T) {
		t.Parallel()

		args := createMockEventNotifierArgs()
		args.Marshaller = nil

		en, err := notifier.NewEventNotifier(args)
		require.Nil(t, en)
		require.Equal(t, notifier.ErrNilMarshaller, err)
	})

	t.Run("nil hasher", func(t *testing.T) {
		t.Parallel()

		args := createMockEventNotifierArgs()
		args.Hasher = nil

		en, err := notifier.NewEventNotifier(args)
		require.Nil(t, en)
		require.Equal(t, notifier.ErrNilHasher, err)
	})

	t.Run("nil pub key converter", func(t *testing.T) {
		t.Parallel()

		args := createMockEventNotifierArgs()
		args.PubKeyConverter = nil

		en, err := notifier.NewEventNotifier(args)
		require.Nil(t, en)
		require.Equal(t, notifier.ErrNilPubKeyConverter, err)
	})

	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		en, err := notifier.NewEventNotifier(createMockEventNotifierArgs())
		require.Nil(t, err)
		require.NotNil(t, en)
	})
}

func TestSaveBlock(t *testing.T) {
	t.Parallel()

	args := createMockEventNotifierArgs()

	txHash1 := "txHash1"
	scrHash1 := "scrHash1"

	wasCalled := false
	args.HttpClient = &mock.HTTPClientStub{
		PostCalled: func(route string, payload interface{}) error {
			saveBlockData := payload.(outportSenderData.ArgsSaveBlock)

			require.Equal(t, hex.EncodeToString([]byte(txHash1)), saveBlockData.TransactionsPool.Logs[0].TxHash)
			for txHash := range saveBlockData.TransactionsPool.Txs {
				require.Equal(t, hex.EncodeToString([]byte(txHash1)), txHash)
			}

			for scrHash := range saveBlockData.TransactionsPool.Scrs {
				require.Equal(t, hex.EncodeToString([]byte(scrHash1)), scrHash)
			}

			wasCalled = true
			return nil
		},
	}

	en, _ := notifier.NewEventNotifier(args)

	saveBlockData := &outport.ArgsSaveBlockData{
		HeaderHash: []byte{},
		TransactionsPool: &outport.Pool{
			Txs: map[string]data.TransactionHandlerWithGasUsedAndFee{
				txHash1: nil,
			},
			Scrs: map[string]data.TransactionHandlerWithGasUsedAndFee{
				scrHash1: nil,
			},
			Logs: []*data.LogData{
				{
					TxHash: txHash1,
				},
			},
		},
	}

	err := en.SaveBlock(saveBlockData)
	require.Nil(t, err)

	require.True(t, wasCalled)
}

func TestRevertIndexedBlock(t *testing.T) {
	t.Parallel()

	args := createMockEventNotifierArgs()

	wasCalled := false
	args.HttpClient = &mock.HTTPClientStub{
		PostCalled: func(route string, payload interface{}) error {
			wasCalled = true
			return nil
		},
	}

	en, _ := notifier.NewEventNotifier(args)

	header := &block.Header{
		Nonce: 1,
		Round: 2,
		Epoch: 3,
	}
	err := en.RevertIndexedBlock(header, &block.Body{})
	require.Nil(t, err)

	require.True(t, wasCalled)
}

func TestFinalizedBlock(t *testing.T) {
	t.Parallel()

	args := createMockEventNotifierArgs()

	wasCalled := false
	args.HttpClient = &mock.HTTPClientStub{
		PostCalled: func(route string, payload interface{}) error {
			wasCalled = true
			return nil
		},
	}

	en, _ := notifier.NewEventNotifier(args)

	hash := []byte("headerHash")
	err := en.FinalizedBlock(hash)
	require.Nil(t, err)

	require.True(t, wasCalled)
}

func TestMockFunctions(t *testing.T) {
	t.Parallel()

	defer func() {
		r := recover()
		if r != nil {
			assert.Fail(t, fmt.Sprintf("should have not panicked: %v", r))
		}
	}()

	en, err := notifier.NewEventNotifier(createMockEventNotifierArgs())
	require.Nil(t, err)
	require.False(t, en.IsInterfaceNil())

	err = en.SaveRoundsInfo(nil)
	require.Nil(t, err)

	err = en.SaveValidatorsRating("", nil)
	require.Nil(t, err)

	err = en.SaveValidatorsPubKeys(nil, 0)
	require.Nil(t, err)

	err = en.SaveAccounts(0, nil, 0)
	require.Nil(t, err)

	err = en.Close()
	require.Nil(t, err)
}
