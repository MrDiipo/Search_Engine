package partition

import (
	"golang.org/x/xerrors"
	"net"
	"os"
	"strconv"
	"strings"
)

var (
	getHostname = os.Hostname
	lookupSRV   = net.LookupSRV
	// ErrPartitionDataAvailableYet is returned by the SRV-aware partition
	// detector to indicate that SRV records for this target application are
	// not yet available.
	ErrPartitionDataAvailableYet = xerrors.Errorf("no partition data available yet")
)

// Detector is implemented by types that can assign a clustered application
type Detector interface {
	PartitionInfo() (int, int, error)
}

// FromSRVRecords detects the number of partitions by performing an SRV query
// and counting the number of results.
type FromSRVRecords struct {
	srvName string
}

// DetectFromSRVRecords This detector is meant to be used in conjuction with a Stateful Set in a
// kubernetes environment.
func DetectFromSRVRecords(srvName string) FromSRVRecords {
	return FromSRVRecords{srvName: srvName}
}

// PartitionInfo implements PartitionDetector.
func (det FromSRVRecords) PartitionInfo() (int, int, error) {
	hostname, err := getHostname()
	if err != nil {
		return -1, -1, xerrors.Errorf("partition detector: unable to detect host name: %w", err)
	}
	tokens := strings.Split(hostname, "-")
	partition, err := strconv.ParseInt(tokens[len(tokens)-1], 10, 32)
	if err != nil {
		return -1, -1, xerrors.Errorf("partition detector: unable to extract partition number from host name suffix")
	}
	_, addrs, err := lookupSRV("", "", det.srvName)
	if err != nil {
		return -1, -1, ErrPartitionDataAvailableYet
	}
	return int(partition), len(addrs), nil
}

// Fixed is a dummy PartitionDetector implementation that always returns back
// the same partition details.
type Fixed struct {
	// The assigned partition number.
	Partition int
	// The number of partitions.
	NumPartitions int
}

// PartitionInfo implements PartitionDetector
func (det Fixed) PartitionInfo() (int, int, error) {
	return det.Partition, det.NumPartitions, nil
}
