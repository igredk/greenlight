package data

import (
	"fmt"
	"strconv"
)

type Runtime int32

// Implement a MarshalJSON() method on the Runtime type so that it satisfies the json.Marshaler interface.
// This should return the JSON-encoded value for the movie runtime in the format "<runtime> mins".
func (r Runtime) MarshalJSON() ([]byte, error) {
	jsonValue := fmt.Sprintf("%d mins", r)
	// Wrap the string in double quotes in order to be a valid *JSON string*.
	quotedJSONValue := strconv.Quote(jsonValue)

	return []byte(quotedJSONValue), nil
}
