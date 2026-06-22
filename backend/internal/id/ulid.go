// SPDX-License-Identifier: MIT

package id

import (
	"errors"

	"github.com/oklog/ulid/v2"
)

func NewULID() string {
	return ulid.Make().String()
}

func ValidateULID(value string) error {
	if _, err := ulid.ParseStrict(value); err != nil {
		return errors.New("invalid ULID")
	}
	return nil
}

func IsULID(value string) bool {
	return ValidateULID(value) == nil
}
