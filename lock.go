package couchbasearray

import (
	"errors"
	"log"

	"github.com/coreos/go-etcd/etcd"
)

const (
	// ErrorKeyNotFound is the key not found error code from etcd
	ErrorKeyNotFound = 100
	// ErrorCompareFailed is the key compare failed error code from etcd
	ErrorCompareFailed = 101
	// ErrorNodeExist is the key exists failed error code from etcd
	ErrorNodeExist = 105
)

// ErrLockInUse is returned when a lock is in use
var ErrLockInUse = errors.New("lock in use")

// AcquireLock attempts to create a new lock. If the lock already exists it returns an error
func AcquireLock(identifier string, namespace string, durationInSeconds uint64) error {
	client := NewEtcdClient()
	_, err := client.Create(namespace, identifier, durationInSeconds)
	if err != nil {
		eerr, ok := err.(*etcd.EtcdError)
		if ok && eerr.ErrorCode == ErrorNodeExist {

		} else {
			log.Println(err)
			return err
		}
	}

	_, err = client.CompareAndSwap(namespace, identifier, durationInSeconds, identifier, 0)
	if err != nil {
		eerr, ok := err.(*etcd.EtcdError)
		if ok && eerr.ErrorCode == ErrorCompareFailed {
			return ErrLockInUse
		}

		log.Println(err)
		return err
	}
	return err
}

// ReleaseLock releases an existing lock
func ReleaseLock(identifier string, namespace string) error {
	if err := AcquireLock(identifier, namespace, 10); err != nil {
		return err
	}

	client := NewEtcdClient()
	_, err := client.CompareAndDelete(namespace, identifier, 0)
	return err
}
