package storage

import (
	"errors"

	"gopkg.in/mistifyio/go-zfs.v2"
)

var (
	// ErrVolumeNotExists is returned when an invalid volume name is provided
	// by a caller.
	ErrVolumeNotExists = errors.New("volume not found")
)

// Volume is a block storage volume which is allocated from a Pool.  Typically,
// this is a ZFS-based zvol.
type Volume interface {
	Name() string
	Size() uint64

	Destroy() error
}

// Zvol is a ZFS-backed implementation of Volume.  It represents block storage
// which may be allocated and released.
type Zvol struct {
	zvol *zfs.Dataset
}

// Destroy completely destroys this volume.
func (z *Zvol) Destroy() error {
	return z.zvol.Destroy(zfs.DestroyRecursive)
}

// Name returns the name of a ZFS zvol.
func (z *Zvol) Name() string {
	return z.zvol.Name
}

// Size returns the size of a ZFS zvol.
func (z *Zvol) Size() uint64 {
	return z.zvol.Volsize
}
