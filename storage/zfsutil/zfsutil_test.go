package zfsutil

import (
	"errors"
	"fmt"
	"testing"

	"gopkg.in/mistifyio/go-zfs.v2"
)

// errorTest is a struct used for testing common error-checking functions.
type errorTest struct {
	text string
	err  error
	ok   bool
}

// TestIsZFSPermissionDenied verifies that ZFS permission denied errors are
// properly detected.
func TestIsZFSPermissionDenied(t *testing.T) {
	// Try all common failure tests, add one successful test
	tests := append(errTests(), &errorTest{
		text: "ZFS error, permission denied",
		err: &zfs.Error{
			Stderr: fmt.Sprintf("Unable to open %s: Permission denied.\n", devZFS),
		},
		ok: true,
	})

	// Run all tests to check output
	for _, test := range tests {
		if ok := IsZFSPermissionDenied(test.err); ok != test.ok {
			t.Fatalf("unexpected result: %v != %v [text: %s]", ok, test.ok, test.text)
		}
	}
}

// TestIsZpoolNotExists verifies that ZFS zstore zpool not found errors are
// properly detected.
func TestIsZpoolNotExists(t *testing.T) {
	// Try all common failure tests, add one successful test
	tests := append(errTests(), &errorTest{
		text: "ZFS error, zstore zpool not found",
		err: &zfs.Error{
			Stderr: fmt.Sprintf("cannot open '%s': no such pool\n", ZpoolName),
		},
		ok: true,
	})

	// Run all tests to check output
	for _, test := range tests {
		if ok := IsZpoolNotExists(test.err); ok != test.ok {
			t.Fatalf("unexpected result: %v != %v [text: %s]", ok, test.ok, test.text)
		}
	}
}

// TestIsDatasetNotExists verifies that ZFS zstore dataset not found errors are
// properly detected.
func TestIsDatasetNotExists(t *testing.T) {
	// Try all common failure tests, add one successful test
	tests := append(errTests(), &errorTest{
		text: "ZFS error, dataset does not exist",
		err: &zfs.Error{
			Stderr: "dataset does not exist\n",
		},
		ok: true,
	})

	// Run all tests to check output
	for _, test := range tests {
		if ok := IsDatasetNotExists(test.err); ok != test.ok {
			t.Fatalf("unexpected result: %v != %v [text: %s]", ok, test.ok, test.text)
		}
	}
}

// TestIsOutOfSpace verifies that ZFS zstore zpool out of space errors are
// properly detected.
func TestIsOutOfSpace(t *testing.T) {
	// Try all common failure tests, add one successful test
	tests := append(errTests(), &errorTest{
		text: "ZFS error, zpool out of space",
		err: &zfs.Error{
			Stderr: "out of space\n",
		},
		ok: true,
	})

	// Run all tests to check output
	for _, test := range tests {
		if ok := IsOutOfSpace(test.err); ok != test.ok {
			t.Fatalf("unexpected result: %v != %v [text: %s]", ok, test.ok, test.text)
		}
	}
}

// errTests returns some common errorTest values which should not register
// as a specific type of ZFS error.
func errTests() []*errorTest {
	return []*errorTest{
		{
			text: "no error",
			err:  nil,
			ok:   false,
		},
		{
			text: "string error",
			err:  errors.New("foo"),
			ok:   false,
		},
		{
			text: "some ZFS error",
			err: &zfs.Error{
				Stderr: "some error",
			},
			ok: false,
		},
	}
}
