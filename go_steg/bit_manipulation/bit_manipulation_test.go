package bit_manipulation

import (
	"reflect"
	"testing"
)

func TestQuartersOfBytes16(t *testing.T) {
	type args struct {
		intToSplit uint16
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "Test Splitting 0",
			args: args{
				intToSplit: 0,
			},
			want: []byte{0, 0, 0, 0},
		},
		{
			name: "Test Splitting 1",
			args: args{
				intToSplit: 1,
			},
			want: []byte{0, 0, 0, 1},
		},
		{
			name: "Test Splitting 350",
			args: args{
				intToSplit: 350,
			},
			want: []byte{1, 1, 3, 2},
		},
		{
			name: "Test Splitting 10000",
			args: args{
				intToSplit: 10000,
			},
			want: []byte{0, 1, 0, 0},
		},
		{
			name: "Test Splitting 65535",
			args: args{
				intToSplit: 65535,
			},
			want: []byte{3, 3, 3, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := QuartersOfBytes16(tt.args.intToSplit); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("QuartersOfBytes16() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestQuartersOfBytes32(t *testing.T) {
	type args struct {
		intToSplit uint32
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "Test Splitting 0",
			args: args{
				intToSplit: 0,
			},
			want: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			name: "Test Splitting 1",
			args: args{
				intToSplit: 1,
			},
			want: []byte{0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			name: "Test Splitting 350",
			args: args{
				intToSplit: 350,
			},
			want: []byte{1, 1, 3, 2, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			name: "Test Splitting 4294967295",
			args: args{
				intToSplit: 4294967295,
			},
			want: []byte{3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := QuartersOfBytes32(tt.args.intToSplit); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("QuartersOfBytes32() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestQuartersOfBytes64(t *testing.T) {
	type args struct {
		intToSplit uint64
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "Test Splitting 0",
			args: args{
				intToSplit: 0,
			},
			want: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			name: "Test Splitting 1",
			args: args{
				intToSplit: 1,
			},
			want: []byte{0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			name: "Test Splitting 350",
			args: args{
				intToSplit: 350,
			},
			want: []byte{1, 1, 3, 2, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			name: "Test Splitting 1000000000000000000",
			args: args{
				intToSplit: 1000000000000000000,
			},
			want: []byte{0, 0, 0, 0, 0, 0, 0, 0, 1, 2, 1, 0, 2, 2, 1, 3, 2, 3, 0, 3, 2, 3, 1, 2},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := QuartersOfBytes64(tt.args.intToSplit); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("QuartersOfBytes64() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReturnMaskDifference(t *testing.T) {
	type args struct {
		maskInt     int32
		multiplier  int32
		firstIndex  int16
		secondIndex int16
		colorInt    uint8
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Test Mask Difference is False",
			args: args{
				maskInt:     1,
				multiplier:  1,
				firstIndex:  0,
				secondIndex: 1,
				colorInt:    1,
			},
			want: false,
		},
		{
			name: "Test Mask Difference is True",
			args: args{
				maskInt:     8,
				multiplier:  1,
				firstIndex:  28,
				secondIndex: 27,
				colorInt:    16,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ReturnMaskDifference(tt.args.maskInt, tt.args.multiplier, tt.args.firstIndex, tt.args.secondIndex, tt.args.colorInt); got != tt.want {
				t.Errorf("ReturnMaskDifference() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSplitByteIntoQuarters(t *testing.T) {
	type args struct {
		b byte
	}
	tests := []struct {
		name string
		args args
		want [4]byte
	}{
		{
			name: "Test Splitting 0",
			args: args{
				b: 0,
			},
			want: [4]byte{0, 0, 0, 0},
		},
		{
			name: "Test Splitting 1",
			args: args{
				b: 1,
			},
			want: [4]byte{0, 0, 0, 1},
		},
		{
			name: "Test Splitting 255",
			args: args{
				b: 255,
			},
			want: [4]byte{3, 3, 3, 3},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SplitByteIntoQuarters(tt.args.b); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SplitByteIntoQuarters() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_clearLastTwoBits(t *testing.T) {
	type args struct {
		b byte
	}
	tests := []struct {
		name string
		args args
		want byte
	}{
		{
			name: "Test Clearing 0",
			args: args{
				b: 0,
			},
			want: 0,
		},
		{
			name: "Test Clearing 1",
			args: args{
				b: 1,
			},
			want: 0,
		},
		{
			name: "Test Clearing 255",
			args: args{
				b: 255,
			},
			want: 252,
		},
		{
			name: "Test Clearing 66",
			args: args{
				b: 66,
			},
			want: 64,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := clearLastTwoBits(tt.args.b); got != tt.want {
				t.Errorf("clearLastTwoBits() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetLastTwoBits(t *testing.T) {
	type args struct {
		b          byte
		valueToSet byte
	}
	tests := []struct {
		name string
		args args
		want byte
	}{
		{
			name: "Test Setting 0",
			args: args{
				b:          0,
				valueToSet: 0,
			},
			want: 0,
		},
		{
			name: "Test Setting 1",
			args: args{
				b:          0,
				valueToSet: 1,
			},
			want: 1,
		},
		{
			name: "Test Setting 3",
			args: args{
				b:          0,
				valueToSet: 3,
			},
			want: 3,
		},
		{
			name: "Test Setting 3 from 255",
			args: args{
				b:          255,
				valueToSet: 3,
			},
			want: 255,
		},
		{
			name: "Test Setting 2 from 255",
			args: args{
				b:          255,
				valueToSet: 2,
			},
			want: 254,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SetLastTwoBits(tt.args.b, tt.args.valueToSet); got != tt.want {
				t.Errorf("SetLastTwoBits() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetLastTwoBits(t *testing.T) {
	type args struct {
		b byte
	}
	tests := []struct {
		name string
		args args
		want byte
	}{
		{
			name: "Test Getting 0",
			args: args{
				b: 0,
			},
			want: 0,
		},
		{
			name: "Test Getting 1",
			args: args{
				b: 1,
			},
			want: 1,
		},
		{
			name: "Test Getting 3",
			args: args{
				b: 3,
			},
			want: 3,
		},
		{
			name: "Test Getting 255",
			args: args{
				b: 255,
			},
			want: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetLastTwoBits(tt.args.b); got != tt.want {
				t.Errorf("GetLastTwoBits() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConstructByteFromQuartersAsSlice(t *testing.T) {
	type args struct {
		b []byte
	}
	tests := []struct {
		name string
		args args
		want byte
	}{
		{
			name: "Test Constructing 0",
			args: args{
				b: []byte{0, 0, 0, 0},
			},
			want: 0,
		},
		{
			name: "Test Constructing 1",
			args: args{
				b: []byte{0, 0, 0, 1},
			},
			want: 1,
		},
		{
			name: "Test Constructing 255",
			args: args{
				b: []byte{3, 3, 3, 3},
			},
			want: 255,
		},
		{
			name: "Test Constructing 65",
			args: args{
				b: []byte{1, 0, 0, 1},
			},
			want: 65,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ConstructByteFromQuartersAsSlice(tt.args.b); got != tt.want {
				t.Errorf("ConstructByteFromQuartersAsSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConstructByteFromQuarters(t *testing.T) {
	type args struct {
		first  byte
		second byte
		third  byte
		fourth byte
	}
	tests := []struct {
		name string
		args args
		want byte
	}{
		{
			name: "Test Constructing 0",
			args: args{
				first:  0,
				second: 0,
				third:  0,
				fourth: 0,
			},
			want: 0,
		},
		{
			name: "Test Constructing 1",
			args: args{
				first:  0,
				second: 0,
				third:  0,
				fourth: 1,
			},
			want: 1,
		},
		{
			name: "Test Constructing 255",
			args: args{
				first:  3,
				second: 3,
				third:  3,
				fourth: 3,
			},
			want: 255,
		},
		{
			name: "Test Constructing 65",
			args: args{
				first:  1,
				second: 0,
				third:  0,
				fourth: 1,
			},
			want: 65,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ConstructByteFromQuarters(tt.args.first, tt.args.second, tt.args.third, tt.args.fourth); got != tt.want {
				t.Errorf("ConstructByteFromQuarters() = %v, want %v", got, tt.want)
			}
		})
	}
}
