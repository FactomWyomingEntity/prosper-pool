package stratum

// ShareCheck determines if the share has a chance of being accepted by the
// network. In factom, between minute 0 and minute 1, there is no chance
// the share will be accepted, as the jobid will be for the previous height.
type ShareCheck interface {
	CanSubmit() bool
	CanSubmitHeight(h int32) bool
}

type AlwaysYesShareCheck struct{}

func (AlwaysYesShareCheck) CanSubmit() bool              { return true }
func (AlwaysYesShareCheck) CanSubmitHeight(h int32) bool { return true }
