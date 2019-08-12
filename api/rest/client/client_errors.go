package client

import (
	"fmt"
)

type DiagnosticsBundleNotFoundError struct {
	ID string
}

func (d *DiagnosticsBundleNotFoundError) Error() string {
	return fmt.Sprintf("bundle %s not found", d.ID)
}

type DiagnosticsBundleUnreadableError struct {
	ID string
}

func (d *DiagnosticsBundleUnreadableError) Error() string {
	return fmt.Sprintf("bundle %s not readable", d.ID)
}

type DiagnosticsBundleNotCompletedError struct {
	ID string
}

func (d *DiagnosticsBundleNotCompletedError) Error() string {
	return fmt.Sprintf("bundle %s canceled or already deleted", d.ID)
}

type DiagnosticsBundleAlreadyExists struct {
	ID string
}

func (d *DiagnosticsBundleAlreadyExists) Error() string {
	return fmt.Sprintf("bundle %s already exists", d.ID)
}
