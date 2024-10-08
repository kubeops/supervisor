// Code generated by go-enum DO NOT EDIT.
// Version:
// Revision:
// Build Date:
// Built By:

package v1alpha1

import (
	"fmt"
	"strings"
)

const (
	// SinglestoreOpsRequestTypeUpdateVersion is a SinglestoreOpsRequestType of type UpdateVersion.
	SinglestoreOpsRequestTypeUpdateVersion SinglestoreOpsRequestType = "UpdateVersion"
	// SinglestoreOpsRequestTypeHorizontalScaling is a SinglestoreOpsRequestType of type HorizontalScaling.
	SinglestoreOpsRequestTypeHorizontalScaling SinglestoreOpsRequestType = "HorizontalScaling"
	// SinglestoreOpsRequestTypeVerticalScaling is a SinglestoreOpsRequestType of type VerticalScaling.
	SinglestoreOpsRequestTypeVerticalScaling SinglestoreOpsRequestType = "VerticalScaling"
	// SinglestoreOpsRequestTypeVolumeExpansion is a SinglestoreOpsRequestType of type VolumeExpansion.
	SinglestoreOpsRequestTypeVolumeExpansion SinglestoreOpsRequestType = "VolumeExpansion"
	// SinglestoreOpsRequestTypeRestart is a SinglestoreOpsRequestType of type Restart.
	SinglestoreOpsRequestTypeRestart SinglestoreOpsRequestType = "Restart"
	// SinglestoreOpsRequestTypeConfiguration is a SinglestoreOpsRequestType of type Configuration.
	SinglestoreOpsRequestTypeConfiguration SinglestoreOpsRequestType = "Configuration"
	// SinglestoreOpsRequestTypeReconfigureTLS is a SinglestoreOpsRequestType of type ReconfigureTLS.
	SinglestoreOpsRequestTypeReconfigureTLS SinglestoreOpsRequestType = "ReconfigureTLS"
)

var ErrInvalidSinglestoreOpsRequestType = fmt.Errorf("not a valid SinglestoreOpsRequestType, try [%s]", strings.Join(_SinglestoreOpsRequestTypeNames, ", "))

var _SinglestoreOpsRequestTypeNames = []string{
	string(SinglestoreOpsRequestTypeUpdateVersion),
	string(SinglestoreOpsRequestTypeHorizontalScaling),
	string(SinglestoreOpsRequestTypeVerticalScaling),
	string(SinglestoreOpsRequestTypeVolumeExpansion),
	string(SinglestoreOpsRequestTypeRestart),
	string(SinglestoreOpsRequestTypeConfiguration),
	string(SinglestoreOpsRequestTypeReconfigureTLS),
}

// SinglestoreOpsRequestTypeNames returns a list of possible string values of SinglestoreOpsRequestType.
func SinglestoreOpsRequestTypeNames() []string {
	tmp := make([]string, len(_SinglestoreOpsRequestTypeNames))
	copy(tmp, _SinglestoreOpsRequestTypeNames)
	return tmp
}

// SinglestoreOpsRequestTypeValues returns a list of the values for SinglestoreOpsRequestType
func SinglestoreOpsRequestTypeValues() []SinglestoreOpsRequestType {
	return []SinglestoreOpsRequestType{
		SinglestoreOpsRequestTypeUpdateVersion,
		SinglestoreOpsRequestTypeHorizontalScaling,
		SinglestoreOpsRequestTypeVerticalScaling,
		SinglestoreOpsRequestTypeVolumeExpansion,
		SinglestoreOpsRequestTypeRestart,
		SinglestoreOpsRequestTypeConfiguration,
		SinglestoreOpsRequestTypeReconfigureTLS,
	}
}

// String implements the Stringer interface.
func (x SinglestoreOpsRequestType) String() string {
	return string(x)
}

// IsValid provides a quick way to determine if the typed value is
// part of the allowed enumerated values
func (x SinglestoreOpsRequestType) IsValid() bool {
	_, err := ParseSinglestoreOpsRequestType(string(x))
	return err == nil
}

var _SinglestoreOpsRequestTypeValue = map[string]SinglestoreOpsRequestType{
	"UpdateVersion":     SinglestoreOpsRequestTypeUpdateVersion,
	"HorizontalScaling": SinglestoreOpsRequestTypeHorizontalScaling,
	"VerticalScaling":   SinglestoreOpsRequestTypeVerticalScaling,
	"VolumeExpansion":   SinglestoreOpsRequestTypeVolumeExpansion,
	"Restart":           SinglestoreOpsRequestTypeRestart,
	"Configuration":     SinglestoreOpsRequestTypeConfiguration,
	"ReconfigureTLS":    SinglestoreOpsRequestTypeReconfigureTLS,
}

// ParseSinglestoreOpsRequestType attempts to convert a string to a SinglestoreOpsRequestType.
func ParseSinglestoreOpsRequestType(name string) (SinglestoreOpsRequestType, error) {
	if x, ok := _SinglestoreOpsRequestTypeValue[name]; ok {
		return x, nil
	}
	return SinglestoreOpsRequestType(""), fmt.Errorf("%s is %w", name, ErrInvalidSinglestoreOpsRequestType)
}

// MustParseSinglestoreOpsRequestType converts a string to a SinglestoreOpsRequestType, and panics if is not valid.
func MustParseSinglestoreOpsRequestType(name string) SinglestoreOpsRequestType {
	val, err := ParseSinglestoreOpsRequestType(name)
	if err != nil {
		panic(err)
	}
	return val
}
