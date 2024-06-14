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
