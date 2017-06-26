package types

import (
	"fmt"

	abci "github.com/tendermint/abci/types"
	"github.com/tendermint/go-wire/data"
	tmpubsub "github.com/tendermint/tmlibs/pubsub"
	tmquery "github.com/tendermint/tmlibs/pubsub/query"
)

// Reserved event types
const (
	EventBond    = "Bond"
	EventUnbond  = "Unbond"
	EventRebond  = "Rebond"
	EventDupeout = "Dupeout"
	EventFork    = "Fork"

	EventNewBlock         = "NewBlock"
	EventNewBlockHeader   = "NewBlockHeader"
	EventNewRound         = "NewRound"
	EventNewRoundStep     = "NewRoundStep"
	EventTimeoutPropose   = "TimeoutPropose"
	EventCompleteProposal = "CompleteProposal"
	EventPolka            = "Polka"
	EventUnlock           = "Unlock"
	EventLock             = "Lock"
	EventRelock           = "Relock"
	EventTimeoutWait      = "TimeoutWait"
	EventVote             = "Vote"
)

func EventTx(tx Tx) string { return fmt.Sprintf("Tx:%X", tx.Hash()) }

///////////////////////////////////////////////////////////////////////////////
// ENCODING / DECODING
///////////////////////////////////////////////////////////////////////////////

var (
	EventDataNameNewBlock       = "new_block"
	EventDataNameNewBlockHeader = "new_block_header"
	EventDataNameTx             = "tx"
	EventDataNameRoundState     = "round_state"
	EventDataNameVote           = "vote"
)

// implements events.EventData
type TMEventDataInner interface {
	// empty interface
}

type TMEventData struct {
	TMEventDataInner `json:"unwrap"`
}

func (tmr TMEventData) MarshalJSON() ([]byte, error) {
	return tmEventDataMapper.ToJSON(tmr.TMEventDataInner)
}

func (tmr *TMEventData) UnmarshalJSON(data []byte) (err error) {
	parsed, err := tmEventDataMapper.FromJSON(data)
	if err == nil && parsed != nil {
		tmr.TMEventDataInner = parsed.(TMEventDataInner)
	}
	return
}

func (tmr TMEventData) Unwrap() TMEventDataInner {
	tmrI := tmr.TMEventDataInner
	for wrap, ok := tmrI.(TMEventData); ok; wrap, ok = tmrI.(TMEventData) {
		tmrI = wrap.TMEventDataInner
	}
	return tmrI
}

func (tmr TMEventData) Empty() bool {
	return tmr.TMEventDataInner == nil
}

const (
	EventDataTypeNewBlock       = byte(0x01)
	EventDataTypeFork           = byte(0x02)
	EventDataTypeTx             = byte(0x03)
	EventDataTypeNewBlockHeader = byte(0x04)

	EventDataTypeRoundState = byte(0x11)
	EventDataTypeVote       = byte(0x12)
)

var tmEventDataMapper = data.NewMapper(TMEventData{}).
	RegisterImplementation(EventDataNewBlock{}, EventDataNameNewBlock, EventDataTypeNewBlock).
	RegisterImplementation(EventDataNewBlockHeader{}, EventDataNameNewBlockHeader, EventDataTypeNewBlockHeader).
	RegisterImplementation(EventDataTx{}, EventDataNameTx, EventDataTypeTx).
	RegisterImplementation(EventDataRoundState{}, EventDataNameRoundState, EventDataTypeRoundState).
	RegisterImplementation(EventDataVote{}, EventDataNameVote, EventDataTypeVote)

// Most event messages are basic types (a block, a transaction)
// but some (an input to a call tx or a receive) are more exotic

type EventDataNewBlock struct {
	Block *Block `json:"block"`
}

// light weight event for benchmarking
type EventDataNewBlockHeader struct {
	Header *Header `json:"header"`
}

// All txs fire EventDataTx
type EventDataTx struct {
	Height int           `json:"height"`
	Tx     Tx            `json:"tx"`
	Data   data.Bytes    `json:"data"`
	Log    string        `json:"log"`
	Code   abci.CodeType `json:"code"`
	Error  string        `json:"error"` // this is redundant information for now
}

// NOTE: This goes into the replay WAL
type EventDataRoundState struct {
	Height int    `json:"height"`
	Round  int    `json:"round"`
	Step   string `json:"step"`

	// private, not exposed to websockets
	RoundState interface{} `json:"-"`
}

type EventDataVote struct {
	Vote *Vote
}

func (_ EventDataNewBlock) AssertIsTMEventData()       {}
func (_ EventDataNewBlockHeader) AssertIsTMEventData() {}
func (_ EventDataTx) AssertIsTMEventData()             {}
func (_ EventDataRoundState) AssertIsTMEventData()     {}
func (_ EventDataVote) AssertIsTMEventData()           {}

///////////////////////////////////////////////////////////////////////////////
// PUBSUB
///////////////////////////////////////////////////////////////////////////////

const (
	// EventTypeKey is a reserved key, used to specify event type in tags.
	EventTypeKey = "tm.events.type"
)

var (
	EventQueryBond             = tmquery.MustParse(EventTypeKey + "=" + EventBond)
	EventQueryUnbond           = tmquery.MustParse(EventTypeKey + "=" + EventUnbond)
	EventQueryRebond           = tmquery.MustParse(EventTypeKey + "=" + EventRebond)
	EventQueryDupeout          = tmquery.MustParse(EventTypeKey + "=" + EventDupeout)
	EventQueryFork             = tmquery.MustParse(EventTypeKey + "=" + EventFork)
	EventQueryNewBlock         = tmquery.MustParse(EventTypeKey + "=" + EventNewBlock)
	EventQueryNewBlockHeader   = tmquery.MustParse(EventTypeKey + "=" + EventNewBlockHeader)
	EventQueryNewRound         = tmquery.MustParse(EventTypeKey + "=" + EventNewRound)
	EventQueryNewRoundStep     = tmquery.MustParse(EventTypeKey + "=" + EventNewRoundStep)
	EventQueryTimeoutPropose   = tmquery.MustParse(EventTypeKey + "=" + EventTimeoutPropose)
	EventQueryCompleteProposal = tmquery.MustParse(EventTypeKey + "=" + EventCompleteProposal)
	EventQueryPolka            = tmquery.MustParse(EventTypeKey + "=" + EventPolka)
	EventQueryUnlock           = tmquery.MustParse(EventTypeKey + "=" + EventUnlock)
	EventQueryLock             = tmquery.MustParse(EventTypeKey + "=" + EventLock)
	EventQueryRelock           = tmquery.MustParse(EventTypeKey + "=" + EventRelock)
	EventQueryTimeoutWait      = tmquery.MustParse(EventTypeKey + "=" + EventTimeoutWait)
	EventQueryVote             = tmquery.MustParse(EventTypeKey + "=" + EventVote)
)

func EventQueryTx(tx Tx) tmpubsub.Query {
	return tmquery.MustParse(EventTypeKey + "=" + EventTx(tx))
}

type TxEventPublisher interface {
	PublishEventTx(EventDataTx) error
}
