package types

const (
	txEventBufferCapacity = 1000
)

// Interface assertions
var _ TxEventPublisher = (*TxEventBuffer)(nil)

// TxEventBuffer is a buffer of events.
type TxEventBuffer struct {
	next   TxEventPublisher
	events []EventDataTx
}

// NewTxEventBuffer returns a new buffer
func NewTxEventBuffer(next *EventBus) *TxEventBuffer {
	return &TxEventBuffer{
		next:   next,
		events: make([]EventDataTx, 0, txEventBufferCapacity),
	}
}

// PublishWithTags buffers an event to be fired upon finality.
func (b *TxEventBuffer) PublishEventTx(e EventDataTx) error {
	b.events = append(b.events, e)
	return nil
}

// Flush fires events by running next.PublishWithTags on all cached events.
// Blocks. Clears cached events.
func (b *TxEventBuffer) Flush() error {
	for _, e := range b.events {
		err := b.next.PublishEventTx(e)
		if err != nil {
			return err
		}
	}
	b.events = make([]EventDataTx, 0, txEventBufferCapacity)
	return nil
}
