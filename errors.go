package main

import "fmt"

type ChecksumMissingError [32]byte
type ChecksumInvalidError [32]byte
type UnsubscribeRejectedError string
type UnsubscribeUnexpectedError struct {
	Message string
	Code    int
}

func (e ChecksumMissingError) Error() string {
	return fmt.Sprintf("checksum missing: %x", []byte(e[:]))
}

func (e ChecksumInvalidError) Error() string {
	return fmt.Sprintf("checksum invalid: %x", []byte(e[:]))
}

func (e UnsubscribeRejectedError) Error() string {
	return fmt.Sprintf("rejected: %s", string(e))
}

func (e UnsubscribeUnexpectedError) Error() string {
	return fmt.Sprintf("unexpected error: %s", e.Message[:50])
}
