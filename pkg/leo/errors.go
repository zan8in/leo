package leo

import "errors"

var (
	ErrNoTarget       = errors.New("no input target provided")
	ErrNoTargetOrHost = errors.New("no input target or host (file) provided")
	ErrTargetFormat   = errors.New("target format error")
	ErrNoHost         = errors.New("no input host provided")
	ErrNoService      = errors.New("no input service provided")
	ErrNoUsers        = errors.New("no input usernames provided")
	ErrNoPasses       = errors.New("no input passwords provided")

	ErrNoOther = errors.New("other error")
)
