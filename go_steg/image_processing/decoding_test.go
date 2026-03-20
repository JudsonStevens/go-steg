package image_processing

import (
	"go-steg/cli/helpers"
	"reflect"
	"testing"
)

func TestMultiCarrierDecodeByFileNames(t *testing.T) {
	type args struct {
		carrierFileNames []string
		password         string
		outputFileDir    string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Test MultiCarrierDecodeByFileNames",
			args: args{
				carrierFileNames: []string{"../../go_steg/pics/testPhotoOutput/carrierPhoto-0-embedded.png"},
				password:         "testPassword",
				outputFileDir:    "../../go_steg/pics/testPhotoOutput",
			},
		},
	}
	for _, tt := range tests {
		helpers.UseMask = true
		t.Run(tt.name, func(t *testing.T) {
			if err := MultiCarrierDecodeByFileNames(tt.args.carrierFileNames, tt.args.password, tt.args.outputFileDir); (err != nil) != tt.wantErr {
				t.Errorf("MultiCarrierDecodeByFileNames() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_align(t *testing.T) {
	type args struct {
		dataBytes []byte
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "Test Align with 1 byte",
			args: args{
				dataBytes: []byte{1},
			},
			want: []byte{1, 0, 0, 0},
		},
		{
			name: "Test Align with 2 bytes",
			args: args{
				dataBytes: []byte{1, 2},
			},
			want: []byte{1, 2, 0, 0},
		},
		{
			name: "Test Align with 3 bytes",
			args: args{
				dataBytes: []byte{1, 2, 3},
			},
			want: []byte{1, 2, 3, 0},
		},
		{
			name: "Test Align with 4 bytes",
			args: args{
				dataBytes: []byte{1, 2, 3, 4},
			},
			want: []byte{1, 2, 3, 4},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := align(tt.args.dataBytes); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("align() = %v, want %v", got, tt.want)
			}
		})
	}
}
