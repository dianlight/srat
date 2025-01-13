package mapper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockMappable struct {
	Value3 string `json:"value3"`
}
type TestStruct struct {
	Value string `json:"value"`
}

func (m MockMappable) To(dst any) (bool, error) {
	switch v := dst.(type) {
	case *TestStruct:
		*v = TestStruct{Value: "mapped"}
		return true, nil
	default:
		return false, nil
	}
}

type MockUnMappable struct {
	Value2 string `json:"value2"`
}

func (t *TestStruct) From(src any) (bool, error) {
	switch v := src.(type) {
	case MockUnMappable:
		t.Value = v.Value2
		return true, nil
	default:
		return false, nil
	}
}

func TestMapWithMappableSource(t *testing.T) {

	//var src Mappable[TestStruct] = MockMappable{}
	src := MockMappable{}
	dst := &TestStruct{}

	err := Map(dst, src)

	require.NoError(t, err)
	assert.Equal(t, "mapped", dst.Value)
}

func TestMapWithMappableSourceNoMatch(t *testing.T) {
	type TestStruct struct {
		Value3 string
		Age    int
		IsVIP  bool
	}
	//var src Mappable[TestStruct] = MockMappable{}
	src := MockMappable{
		Value3: "unmapped*",
	}
	dst := &TestStruct{}

	err := Map(dst, src)

	require.NoError(t, err)
	assert.Equal(t, "unmapped*", dst.Value3)
}

func TestMapWithMappableDestination(t *testing.T) {

	//var src Mappable[TestStruct] = MockMappable{}
	src := MockUnMappable{
		Value2: "unmapped",
	}
	dst := &TestStruct{}

	err := Map(dst, src)

	require.NoError(t, err)
	assert.Equal(t, "unmapped", dst.Value)
}

func TestMapFromPtrWithMappableDestination(t *testing.T) {

	//var src Mappable[TestStruct] = MockMappable{}
	src := MockUnMappable{
		Value2: "unmapped",
	}
	dst := &TestStruct{}

	err := Map(dst, &src)

	require.NoError(t, err)
	assert.Equal(t, "unmapped", dst.Value)
}

func TestMapFromMapStringAny(t *testing.T) {
	type TestStruct struct {
		Name  string
		Age   int
		IsVIP bool
	}

	src := map[string]any{
		"Nothing": "No value",
		"Name":    "John Doe",
		"Age":     30,
		"IsVIP":   true,
	}

	dst := &TestStruct{}

	err := Map(dst, src)

	require.NoError(t, err)
	assert.Equal(t, "John Doe", dst.Name)
	assert.Equal(t, 30, dst.Age)
	assert.True(t, dst.IsVIP)
}

func TestMapFromMapStringPointerAny(t *testing.T) {
	type TestStruct struct {
		Name  string
		Age   int
		IsVIP bool
	}

	src := map[string]any{
		"Nothing": "No value",
		"Name":    "John Doe",
		"Age":     30,
		"IsVIP":   true,
	}

	dst := &TestStruct{}

	err := Map(dst, &src)

	require.NoError(t, err)
	assert.Equal(t, "John Doe", dst.Name)
	assert.Equal(t, 30, dst.Age)
	assert.True(t, dst.IsVIP)
}

func TestMapFromSliceAny(t *testing.T) {
	type TestStruct struct {
		Name  string
		Value int
	}

	src := []any{
		map[string]any{"Name": "Item1", "Value": 10},
		map[string]any{"Name": "Item2", "Value": 20},
	}

	var dst []TestStruct

	err := Map(&dst, src)

	require.NoError(t, err)
	assert.Len(t, dst, 2)
	assert.Equal(t, "Item1", dst[0].Name)
	assert.Equal(t, 10, dst[0].Value)
	assert.Equal(t, "Item2", dst[1].Name)
	assert.Equal(t, 20, dst[1].Value)
}

func TestMapWithUnsupportedSourceType(t *testing.T) {
	type TestStruct struct {
		Value string
	}

	src := 42 // An integer is an unsupported source type
	dst := &TestStruct{}

	err := Map(dst, src)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Unsupported source type: int for destination")
}

func TestMapWithEmptyMapStringAnySource(t *testing.T) {
	type TestStruct struct {
		Name  string
		Value int
	}

	src := map[string]any{}
	dst := &TestStruct{Name: "Original", Value: 42}

	err := Map(dst, src)

	require.NoError(t, err)
	assert.Equal(t, "Original", dst.Name)
	assert.Equal(t, 42, dst.Value)
}
func TestMapWithNilDestination(t *testing.T) {
	var dst *string
	src := "test"

	err := Map(dst, src)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Unsupported source type: string")
}
func TestMapWithEmptySliceAnySource(t *testing.T) {
	type TestStruct struct {
		Name  string
		Value int
	}

	src := []any{}
	dst := &TestStruct{Name: "Original", Value: 42}

	err := Map(dst, src)

	require.NoError(t, err)
	assert.Equal(t, "Original", dst.Name)
	assert.Equal(t, 42, dst.Value)
}
func TestMapPreservesExistingValues(t *testing.T) {
	type TestStruct struct {
		Name  string
		Age   int
		Email string
	}

	dst := &TestStruct{
		Name:  "John Doe",
		Age:   30,
		Email: "john@example.com",
	}

	src := map[string]any{
		"Name": "Jane Doe",
		"Age":  35,
	}

	err := Map(dst, src)

	require.NoError(t, err)
	assert.Equal(t, "Jane Doe", dst.Name)
	assert.Equal(t, 35, dst.Age)
	assert.Equal(t, "john@example.com", dst.Email)
}
func TestMapWithNestedStructures(t *testing.T) {
	type Address struct {
		Street string
		City   string
	}

	type Person struct {
		Name    string
		Age     int
		Address Address
	}

	src := map[string]any{
		"Name": "John Doe",
		"Age":  30,
		"Address": map[string]any{
			"Street": "123 Main St",
			"City":   "Anytown",
		},
	}

	dst := &Person{}

	err := Map(dst, src)

	require.NoError(t, err)
	assert.Equal(t, "John Doe", dst.Name)
	assert.Equal(t, 30, dst.Age)
	assert.Equal(t, "123 Main St", dst.Address.Street)
	assert.Equal(t, "Anytown", dst.Address.City)
}
func TestMapTypeConversions(t *testing.T) {
	type TestStruct struct {
		IntValue    int
		FloatValue  float64
		StringValue string
		BoolValue   bool
		//		SliceValue  []int
		//		MapValue    map[string]int
	}

	src := map[string]any{
		"IntValue":    float64(42),
		"FloatValue":  "3.14",
		"StringValue": 123,
		"BoolValue":   "true",
		"SliceValue":  []any{1, 2, 3},
		"MapValue":    map[string]any{"key": 42},
	}

	dst := &TestStruct{}

	err := Map(dst, src)

	//pretty.Println(src, dst)

	require.NoError(t, err)
	assert.Equal(t, 42, dst.IntValue)
	assert.InDelta(t, 3.14, dst.FloatValue, 0.01)
	assert.Equal(t, "{", dst.StringValue)
	//assert.Equal(t, []int{1, 2, 3}, dst.SliceValue)
	//assert.Equal(t, map[string]int{"key": 42}, dst.MapValue)
	assert.True(t, dst.BoolValue)
}

func TestMapFromMapToSlice(t *testing.T) {
	type TestStruct struct {
		Name string
		Age  int
	}

	src := []map[string]any{
		{"Name": "John Doe", "Age": 30},
		{"Name": "Jane Doe", "Age": 35},
	}

	dst := make([]TestStruct, 0, len(src))

	err := Map(&dst, src)

	require.NoError(t, err)
	assert.Len(t, dst, 2)
	assert.Equal(t, "John Doe", dst[0].Name)
	assert.Equal(t, 30, dst[0].Age)
	assert.Equal(t, "Jane Doe", dst[1].Name)
	assert.Equal(t, 35, dst[1].Age)
}

func TestMapFromStructToSlice(t *testing.T) {
	type TestStruct struct {
		Name  string      `mapper:"key"`
		Value interface{} `mapper:"value"`
	}

	type TestStuctSource struct {
		Pippo       int
		Pluto       string
		Minni       string
		ZioPaperone map[string]string
	}

	src := TestStuctSource{
		Pippo: 30,
		Pluto: "John Doe",
		Minni: "Jane Doe",
		ZioPaperone: map[string]string{
			"Age": "30",
		},
	}

	dst := make([]TestStruct, 0, 4)

	err := Map(&dst, src)

	require.NoError(t, err)
	assert.Len(t, dst, 4)
	assert.Equal(t, "Pippo", dst[0].Name)
	assert.Equal(t, 30, dst[0].Value)
	assert.Equal(t, "Pluto", dst[1].Name)
	assert.Equal(t, "John Doe", dst[1].Value)
	assert.Equal(t, "Minni", dst[2].Name)
	assert.Equal(t, "Jane Doe", dst[2].Value)
	assert.Equal(t, "ZioPaperone", dst[3].Name)
	assert.Equal(t, map[string]string{
		"Age": "30",
	}, dst[3].Value)
}

func TestMapFromStructToSliceType(t *testing.T) {
	type TestStruct struct {
		Name  string      `mapper:"key"`
		Value interface{} `mapper:"value"`
	}

	type TestStructs []TestStruct

	type TestStuctSource struct {
		Pippo       int
		Pluto       string
		Minni       string
		ZioPaperone map[string]string
	}

	src := TestStuctSource{
		Pippo: 30,
		Pluto: "John Doe",
		Minni: "Jane Doe",
		ZioPaperone: map[string]string{
			"Age": "30",
		},
	}

	var dst TestStructs

	err := Map(&dst, src)

	require.NoError(t, err)
	assert.Len(t, dst, 4)
	assert.Equal(t, "Pippo", dst[0].Name)
	assert.Equal(t, 30, dst[0].Value)
	assert.Equal(t, "Pluto", dst[1].Name)
	assert.Equal(t, "John Doe", dst[1].Value)
	assert.Equal(t, "Minni", dst[2].Name)
	assert.Equal(t, "Jane Doe", dst[2].Value)
	assert.Equal(t, "ZioPaperone", dst[3].Name)
	assert.Equal(t, map[string]string{
		"Age": "30",
	}, dst[3].Value)
}

func TestMapFromSliceTypeToStruct(t *testing.T) {
	type TestStruct struct {
		Name  string      `mapper:"key"`
		Value interface{} `mapper:"value"`
	}

	type TestStructs []TestStruct

	type TestStuctDestination struct {
		Pippo       int
		Pluto       string
		Minni       string
		ZioPaperone map[string]string
	}

	src := TestStructs{
		{Name: "Pippo", Value: 30},
		{Name: "Pluto", Value: "John Doe"},
		{Name: "Minni", Value: "Jane Doe"},
		{Name: "ZioPaperone", Value: map[string]string{"Age": "30"}},
	}

	var dst TestStuctDestination

	err := Map(&dst, src)

	require.NoError(t, err)
	assert.Equal(t, 30, dst.Pippo)
	assert.Equal(t, "John Doe", dst.Pluto)
	assert.Equal(t, "Jane Doe", dst.Minni)
	assert.Equal(t, map[string]string{"Age": "30"}, dst.ZioPaperone)
}
