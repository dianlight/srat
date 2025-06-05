package dbom

import (
	"database/sql/driver"
	"fmt"
	"strings"
)

type MounDataFlag struct {
	Name       string
	NeedsValue bool   // Add NeedsValue field
	FlagValue  string // Add Value field for flags like "uid=<arg>"
}

type MounDataFlags []MounDataFlag

func (self *MounDataFlags) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	svalue, ok := value.(string)
	if !ok {
		return fmt.Errorf("invalid value type for MounDataFlags: %T", value)
	}
	for _, flag := range strings.Split(svalue, ",") {
		if strings.Contains(flag, "=") {
			// Extract the value after the '='
			parts := strings.SplitN(flag, "=", 2)
			if len(parts) == 2 {
				self.Add(MounDataFlag{
					Name:       parts[0],
					NeedsValue: true,
					FlagValue:  parts[1],
				})
			}
		} else if flag != "" {
			self.Add(MounDataFlag{
				Name:       flag,
				NeedsValue: false,
			})
		}
	}
	return nil
}

func (self *MounDataFlags) Add(value MounDataFlag) error {
	*self = append(*self, value)
	return nil
}

// Value implements the driver.Valuer interface.
// It converts the MounDataFlags to a comma-separated string of flags.
// If a flag needs a value, it will be formatted as "name=value".
// Otherwise, it will be just the name of the flag.
func (self MounDataFlags) Value() (driver.Value, error) {
	flags := make([]string, len(self))
	for ix, flag := range self {
		if flag.NeedsValue {
			flags[ix] = fmt.Sprintf("%s=%s", flag.Name, flag.FlagValue)
		} else {
			flags[ix] = flag.Name
		}
	}
	return strings.Join(flags, ","), nil
}

func (MounDataFlags) GormDataType() string {
	return "text"
}
