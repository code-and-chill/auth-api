package timegenerator

import "time"

// TimeGenerator provides an interface for id generation.
type TimeGenerator interface {
	Now() time.Time
}

type timegenerator struct {
}

// NewTimeGenerator instantiates a new time generator.
func NewTimeGenerator() TimeGenerator {
	return &timegenerator{}
}

func (gen *timegenerator) Now() time.Time {
	return time.Now().Truncate(time.Second)
}
