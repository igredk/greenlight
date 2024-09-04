package data

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var ErrInvalidRuntimeFormat = errors.New("invalid runtime format")

type Runtime int32

// Implement a MarshalJSON() method on the Runtime type so that it satisfies the json.Marshaler interface.
// This should return the JSON-encoded value for the movie runtime in the format "<runtime> mins".
func (r Runtime) MarshalJSON() ([]byte, error) {
	jsonValue := fmt.Sprintf("%d mins", r)
	// Wrap the string in double quotes in order to be a valid *JSON string*.
	quotedJSONValue := strconv.Quote(jsonValue)

	return []byte(quotedJSONValue), nil
}

// Implement a UnmarshalJSON() method on the Runtime type so that it satisfies the json.Unmarshaler interface.
// IMPORTANT: Because UnmarshalJSON() needs to modify the receiver (our Runtime type),
// we must use a pointer receiver for this to work correctly.
// Otherwise, we will only be modifying a copy (which is then discarded when this method returns).
func (r *Runtime) UnmarshalJSON(jsonValue []byte) error {
	// We expect that the incoming JSON value will be a string in the format "<runtime> mins",
	// and the first thing we need to do is remove the surrounding double-quotes from this string.
	unquotedJSONValue, err := strconv.Unquote(string(jsonValue))
	if err != nil {
		return ErrInvalidRuntimeFormat
	}

	parts := strings.Split(unquotedJSONValue, " ")
	if len(parts) != 2 || parts[1] != "mins" {
		return ErrInvalidRuntimeFormat
	}
	// Otherwise, parse the string containing the number into an int32.
	i, err := strconv.ParseInt(parts[0], 10, 32)
	if err != nil {
		return ErrInvalidRuntimeFormat
	}
	// Convert the int32 to a Runtime type and assign this to the receiver. Note that we
	// use the * operator to deference the receiver (which is a pointer to a Runtime
	// type) in order to set the underlying value of the pointer.
	*r = Runtime(i)

	return nil
}
