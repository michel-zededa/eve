package ociutil

import (
	"fmt"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	log "github.com/sirupsen/logrus"
)

func manifestsDescImg(image string, options []remote.Option) (name.Reference, *remote.Descriptor, v1.Image, []byte, []byte, int64, error) {
	var (
		manifestDirect, manifestResolved []byte
		img                              v1.Image
		desc                             *remote.Descriptor
		size                             int64
		err                              error
		ref                              name.Reference
	)
	log.Infof("manifestsDescImg(%s)", image)
	// this one should go away
	log.Infof("options %#v", options)

	ref, err = name.ParseReference(image)
	if err != nil {
		log.Errorf("error parsing image (%s): %v", image, err)
		return ref, desc, img, manifestDirect, manifestResolved, size, fmt.Errorf("parsing reference %q: %v", image, err)
	}

	// first get the root manifest. This might be an index or a manifest
	log.Infof("manifestsDescImg(%s) getting image ref %#v", image, ref)
	desc, err = remote.Get(ref, options...)
	if err != nil {
		return ref, desc, img, manifestDirect, manifestResolved, size, fmt.Errorf("error getting manifest: %v", err)
	}
	manifestDirect = desc.Manifest

	// This is where it gets the image manifest, but does not actually save anything
	// It is the manifest of the image itself, not of the index (if it is
	// an index), so it actually does resolve platform-specific
	img, err = desc.Image()
	if err != nil {
		return ref, desc, img, manifestDirect, manifestResolved, size, fmt.Errorf("error pulling image ref: %v", err)
	}

	// check out the manifest and hash
	manifestResolved, err = img.RawManifest()
	if err != nil {
		return ref, desc, img, manifestDirect, manifestResolved, size, fmt.Errorf("error getting resolved manifest bytes: %v", err)
	}

	// we now get the actual manifest, to get the layers and resolve the size
	manifest, err := img.Manifest()
	if err != nil {
		return ref, desc, img, manifestDirect, manifestResolved, size, fmt.Errorf("error getting resolved manifest object: %v", err)
	}

	// size starts at the default of 0
	// add the size of the config
	size += manifest.Config.Size
	// next the layers and their sizes
	for _, layer := range manifest.Layers {
		size += layer.Size
	}

	return ref, desc, img, manifestDirect, manifestResolved, size, nil
}
