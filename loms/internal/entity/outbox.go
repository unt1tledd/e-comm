package entity

type Kind int

const (
	KindUndefined Kind = iota
	KindNotification
)

func (o Kind) String() string {
	switch o {
	case KindUndefined:
		return "undefined"
	case KindNotification:
		return "notification"
	default:
		return ""
	}
}

type OutboxMessage struct {
	IdempotencyKey string
	Kind           Kind
	Data           []byte
}
