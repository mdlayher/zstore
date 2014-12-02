package storage

import (
	"errors"

	"github.com/mdlayher/zstore/storage/zfsutil"

	"gopkg.in/mistifyio/go-zfs.v2"
)

var (
	// ErrPoolOutOfSpace is returned to callers when the underlying Pool no
	// longer has the capacity to create a new volume.
	ErrPoolOutOfSpace = errors.New("pool out of space")

	// ErrVolumeNotExists is returned when an invalid volume name is provided
	// by a caller.
	ErrVolumeNotExists = errors.New("volume not found")
)

// Pool is a storage pool from which Volumes can be created.  Typically, this
// is a ZFS-based storage pool.  The implementation is swappable to enable
// proper testing.
type Pool interface {
	Name() string

	CreateVolume(string, uint64) (Volume, error)
	Volume(string) (Volume, error)
}

// Volume is a block storage volume which is allocated from a Pool.  Typically,
// this is a ZFS-based zvol.
type Volume interface {
	Name() string
	Size() uint64

	Destroy() error
}

// Zpool is a ZFS-backed implementation of Pool.  It enables creation of Zvols,
// which implement Volume.
type Zpool struct {
	zpool *zfs.Zpool
}

// Name returns the name of a ZFS zpool.
func (z *Zpool) Name() string {
	return z.zpool.Name
}

// CreateVolume creates a new Zvol from a Zpool with the specified name and
// size in bytes.
func (z *Zpool) CreateVolume(name string, size uint64) (Volume, error) {
	// Attempt to create volume by name with specified size
	zvol, err := zfs.CreateVolume(name, size, nil)
	if err != nil {
		// If pool is out of space, return out of space
		if zfsutil.IsOutOfSpace(err) {
			return nil, ErrPoolOutOfSpace
		}

		return nil, err
	}

	return &Zvol{
		zvol: zvol,
	}, nil
}

// Volume attempts to retrieve a Zvol from a Zpool by its name.
func (z *Zpool) Volume(name string) (Volume, error) {
	// Attempt to fetch volume by name
	zvol, err := zfs.GetDataset(name)
	if err != nil {
		// If dataset does not exist, return not exists
		if zfsutil.IsDatasetNotExists(err) {
			return nil, ErrVolumeNotExists
		}

		// All other errors
		return nil, err
	}

	// Ensure dataset is a volume; if not, tell client the volume does not exist
	if zvol.Type != zfs.DatasetVolume {
		return nil, ErrVolumeNotExists
	}

	// Return wrapped Volume type
	return &Zvol{
		zvol: zvol,
	}, nil
}

// NewZpool wraps a go-zfs Zpool with a ZFS-based Pool interface implementation.
func NewZpool(zpool *zfs.Zpool) *Zpool {
	return &Zpool{
		zpool: zpool,
	}
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
