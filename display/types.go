package display

// Point is point struct.
type Point struct {
	X int
	Y int
}

// Size is size structure.
type Size struct {
	W int
	H int
}

// Rect is rect structure.
type Rect struct {
	Point
	Size
}
