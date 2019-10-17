package polling

import (
	"fmt"

	"github.com/spf13/viper"
)

// NewTestingDataSource is for unit test.
// Having a testing data source is for unit test mocking
var NewTestingDataSource = func(_ *viper.Viper, source string) (IDataSource, error) {
	return nil, fmt.Errorf("this is a testing datasource for unit tests only")
}
