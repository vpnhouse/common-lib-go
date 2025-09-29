package shaper

type Shaper interface {
	Shape(length int) bool
}

type noShapeType struct{}

func (s *noShapeType) Shape(int) bool {
	return true
}

var NoShape = noShapeType{}
