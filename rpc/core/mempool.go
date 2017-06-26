package core

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"

	abci "github.com/tendermint/abci/types"
	data "github.com/tendermint/go-wire/data"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	"github.com/tendermint/tendermint/types"
)

//-----------------------------------------------------------------------------
// NOTE: tx should be signed, but this is only checked at the app level (not by Tendermint!)

// Returns right away, with no response
func BroadcastTxAsync(tx types.Tx) (*ctypes.ResultBroadcastTx, error) {
	err := mempool.CheckTx(tx, nil)
	if err != nil {
		return nil, fmt.Errorf("Error broadcasting transaction: %v", err)
	}
	return &ctypes.ResultBroadcastTx{Hash: tx.Hash()}, nil
}

// Returns with the response from CheckTx
func BroadcastTxSync(tx types.Tx) (*ctypes.ResultBroadcastTx, error) {
	resCh := make(chan *abci.Response, 1)
	err := mempool.CheckTx(tx, func(res *abci.Response) {
		resCh <- res
	})
	if err != nil {
		return nil, fmt.Errorf("Error broadcasting transaction: %v", err)
	}
	res := <-resCh
	r := res.GetCheckTx()
	return &ctypes.ResultBroadcastTx{
		Code: r.Code,
		Data: r.Data,
		Log:  r.Log,
		Hash: tx.Hash(),
	}, nil
}

// CONTRACT: only returns error if mempool.BroadcastTx errs (ie. problem with the app)
// or if we timeout waiting for tx to commit.
// If CheckTx or DeliverTx fail, no error will be returned, but the returned result
// will contain a non-OK ABCI code.
func BroadcastTxCommit(tx types.Tx) (*ctypes.ResultBroadcastTxCommit, error) {
	// subscribe to tx being committed in block
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	deliverTxResCh := make(chan interface{})
	q := types.EventQueryTx(tx)
	err := eventBus.Subscribe(ctx, "mempool", q, deliverTxResCh)
	if err != nil {
		err = errors.Wrap(err, "failed to subscribe to tx")
		logger.Error("Error broadcasting transaction", "err", err)
		return nil, fmt.Errorf("Error broadcasting transaction: %v", err)
	}
	defer eventBus.Unsubscribe(context.Background(), "mempool", q)

	// broadcast the tx and register checktx callback
	checkTxResCh := make(chan *abci.Response, 1)
	err = mempool.CheckTx(tx, func(res *abci.Response) {
		checkTxResCh <- res
	})
	if err != nil {
		logger.Error("Error broadcasting transaction", "err", err)
		return nil, fmt.Errorf("Error broadcasting transaction: %v", err)
	}
	checkTxRes := <-checkTxResCh
	checkTxR := checkTxRes.GetCheckTx()
	if checkTxR.Code != abci.CodeType_OK {
		// CheckTx failed!
		return &ctypes.ResultBroadcastTxCommit{
			CheckTx:   checkTxR.Result(),
			DeliverTx: abci.Result{},
			Hash:      tx.Hash(),
		}, nil
	}

	// Wait for the tx to be included in a block,
	// timeout after something reasonable.
	// TODO: configurable?
	timer := time.NewTimer(60 * 2 * time.Second)
	select {
	case deliverTxResMsg := <-deliverTxResCh:
		deliverTxRes := deliverTxResMsg.(types.TMEventData).Unwrap().(types.EventDataTx)
		// The tx was included in a block.
		deliverTxR := &abci.ResponseDeliverTx{
			Code: deliverTxRes.Code,
			Data: deliverTxRes.Data,
			Log:  deliverTxRes.Log,
		}
		logger.Info("DeliverTx passed ", "tx", data.Bytes(tx), "response", deliverTxR)
		return &ctypes.ResultBroadcastTxCommit{
			CheckTx:   checkTxR.Result(),
			DeliverTx: deliverTxR.Result(),
			Hash:      tx.Hash(),
			Height:    deliverTxRes.Height,
		}, nil
	case <-timer.C:
		logger.Error("failed to include tx")
		return &ctypes.ResultBroadcastTxCommit{
			CheckTx:   checkTxR.Result(),
			DeliverTx: abci.Result{},
			Hash:      tx.Hash(),
		}, fmt.Errorf("Timed out waiting for transaction to be included in a block")
	}

	panic("Should never happen!")
}

func UnconfirmedTxs() (*ctypes.ResultUnconfirmedTxs, error) {
	txs := mempool.Reap(-1)
	return &ctypes.ResultUnconfirmedTxs{len(txs), txs}, nil
}

func NumUnconfirmedTxs() (*ctypes.ResultUnconfirmedTxs, error) {
	return &ctypes.ResultUnconfirmedTxs{N: mempool.Size()}, nil
}
