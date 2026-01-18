package screenshot

import (
	"errors"
	"image"
)

const errUnsupportedMessage = "screenshot does not support your platform"

// errUnsupported returns an error when the platform does not support screenshot capture.
func errUnsupported() error {
	return errors.New(errUnsupportedMessage)
}

func createImage(rect image.Rectangle) (img *image.RGBA, e error) {
	img = nil
	e = errors.New("Cannot create image.RGBA")

	defer func() {
		err := recover()
		if err == nil {
			e = nil
		}
	}()
	// image.NewRGBA may panic if rect is too large.
	img = image.NewRGBA(rect)

	return img, e
}
